package pkg

import (
	"bytes"
	"fmt"
	"math"
	"net"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/kiga-hub/arc-storage/pkg/cache"
	"github.com/kiga-hub/arc-storage/pkg/metric"
	"github.com/kiga-hub/arc-storage/pkg/metric/monitor"
	dataCache "github.com/kiga-hub/arc/cache"
	"github.com/kiga-hub/arc/logging"
	microComponent "github.com/kiga-hub/arc/micro/component"
	"github.com/kiga-hub/arc/protobuf/pb"
	"github.com/kiga-hub/arc/protocols"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"

	arcGRPC "github.com/kiga-hub/arc-storage/pkg/arc_grpc"
	"github.com/kiga-hub/arc-storage/pkg/arc_volume"
	"github.com/kiga-hub/arc-storage/pkg/config"
	"github.com/kiga-hub/arc-storage/pkg/kafka"
	"github.com/kiga-hub/arc-storage/pkg/protostream"
)

const (
	// ArcStorageAPI Arc Storage Data API
	ArcStorageAPI = "Arc Storage Data"
	// DataBaseName arc
	DataBaseName = "arc"
)

// ArcStorageVersion add to file suffix name
var ArcStorageVersion = "appversion"

const (
	// TypeArc "Arc"
	TypeArc = "Arc"
)

// ArcStorage arc storage struct
type ArcStorage struct {
	listen            net.Listener
	logger            logging.ILogger
	working           *sync.Mutex
	timeoutSyncMap    *sync.Map
	gossipKVCache     *microComponent.GossipKVCacheComponent
	grpcmessage       chan protostream.ProtoStream
	decodeResultChans []chan decodeResult
	timeoutChans      []chan uint64
	kafka             kafka.Handler
	grpcserver        *grpc.Server
	arcFileStore      *arc_volume.ArcVolumeCache
	arcCache          *cache.DataCacheRepo
	config            *config.ArcConfig
	sensorIDsChan     chan []string
	decodeJobChans    []chan []byte
	once              sync.Once
	isConnectTaos     bool
	serviceIsClosing  bool
	exportMetrics     *metric.HandlerMonitor
}

// NewArcStorage Instantiation object
func NewArcStorage(config *config.ArcConfig, logger logging.ILogger, gossipKVCache *microComponent.GossipKVCacheComponent, k kafka.Handler) (*ArcStorage, error) {
	// configure data size validation
	err := protocols.ConfigFrame(math.MaxUint32 / 2)
	if err != nil {
		return nil, err
	}

	arcFileStore, err := arc_volume.NewArcVolumeCache(logger, config, arc_volume.DataTypeMap[TypeArc])
	if err != nil {
		return nil, err
	}

	// initialize the mtric collection module

	g := monitor.NewGRPC()
	cacheRead := monitor.NewCacheRead()
	m, err := metric.NewHandlerMonitor(g, cacheRead)
	if err != nil {
		return nil, err
	}
	db := &ArcStorage{
		config:            config,
		working:           new(sync.Mutex),
		timeoutSyncMap:    &sync.Map{},
		logger:            logger,
		decodeJobChans:    make([]chan []byte, config.Work.WorkCount),
		decodeResultChans: make([]chan decodeResult, config.Work.WorkCount),
		timeoutChans:      make([]chan uint64, config.Work.WorkCount),
		arcFileStore:      arcFileStore,
		grpcmessage:       make(chan protostream.ProtoStream, 1024),
		isConnectTaos:     false,
		exportMetrics:     m,
	}

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)

	// gossipKVCache Timer
	if gossipKVCache != nil {
		db.gossipKVCache = gossipKVCache
		db.sensorIDsChan = make(chan []string, 1024)
		go db.setDefaultSensorids(sigchan)
		go db.gossipKVCacheTimerTask()
	}

	// cache enable
	if db.config.Cache.Enable {
		// real-time query
		db.arcCache = cache.NewCacheRepo(logger)
	}

	if db.config.Kafka.Enable {
		logger.Info("arc Kafka.Enable", k)
		db.kafka = k
	}

	return db, nil
}

func (arc *ArcStorage) receiveDataTimerTask(sigchan chan os.Signal) {
	ticker := time.NewTicker(time.Minute * 1)
	defer func() {
		ticker.Stop()
	}()

	for {
		select {
		case sig := <-sigchan:
			arc.logger.Errorf("Caught signal %v: terminating\n", sig)
			return
		case <-ticker.C:
			// regularly check the timeout
			arc.timeoutSyncMap.Range(arc.timeOutSyncMapWalk)
		}
	}
}

// setDefaultSensorids -
func (arc *ArcStorage) setDefaultSensorids(sigchan chan os.Signal) {
	for {
		select {
		case sig := <-sigchan:
			arc.logger.Errorf("Caught signal %v: terminating\n", sig)
			return

		case ids := <-arc.sensorIDsChan:
			err := arc.gossipKVCache.HaveSensorIDs(ids)
			if err != nil {
				arc.logger.Errorw("gossipkvcache", "ids", ids, "err", err)
			}
		}
	}
}

// gossipKVCacheTimerTask -
func (arc *ArcStorage) gossipKVCacheTimerTask() {
	if sensorids, err := arc.getSensorIDsfromStorage(); err == nil {
		arc.sensorIDsChan <- sensorids
	} else {
		arc.logger.Errorw("getSensorIDsfrom", "err", err)
	}

	ticker := time.NewTicker(time.Minute * 1)
	defer func() {
		ticker.Stop()
	}()

	for range ticker.C {
		if sensorids, err := arc.getSensorIDsfromStorage(); err == nil {
			arc.sensorIDsChan <- sensorids
		} else {
			arc.logger.Error(err)
		}
		runtime.GC()
	}
}

// Start Connect kafka Store & taoClient
func (arc *ArcStorage) Start(stop chan struct{}) {
	var err error

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGINT, syscall.SIGTERM)
	//decode
	for i := 0; i < arc.config.Work.WorkCount; i++ {
		arc.decodeJobChans[i] = make(chan []byte, arc.config.Work.ChanCapacity*4)
		arc.decodeResultChans[i] = make(chan decodeResult, arc.config.Work.ChanCapacity*4)
		// When the channle handles optimal allcation. timeoutChan only needs to allocate the capacity of the number of sensors.
		arc.timeoutChans[i] = make(chan uint64, arc.config.Work.ChanCapacity)
		go arc.decodeWorker(arc.decodeJobChans[i], arc.decodeResultChans[i])
		go arc.handleDecodeResult(sigchan, i)
	}

	arc.serviceIsClosing = false

	// check for timeout
	go arc.receiveDataTimerTask(sigchan)

	// start gRPC server
	if arc.config.Grpc.Enable {
		arc.logger.Infow("Start gRPC Server", "arc.config.GrpcServer", arc.config.Grpc.Server)
		go arc.gRPCServerStream(sigchan, arc.grpcmessage)

		go func() {
			arc.listen, err = net.Listen("tcp", arc.config.Grpc.Server)
			if err != nil {
				arc.logger.Errorw("gRPCListen", "gRPCServer", arc.config.Grpc.Server, "err", err)
			}

			var kaep = keepalive.EnforcementPolicy{
				PermitWithoutStream: true,
			}

			var kasp = keepalive.ServerParameters{
				Time:    time.Duration(10) * arcGRPC.KeepAliveTime,
				Timeout: time.Duration(3) * arcGRPC.KeepAliveTimeout,
			}
			arc.grpcserver = grpc.NewServer(
				grpc.InitialWindowSize(arcGRPC.InitialWindowSize),
				grpc.InitialConnWindowSize(arcGRPC.InitialConnWindowSize),
				grpc.KeepaliveParams(kasp),
				grpc.KeepaliveEnforcementPolicy(kaep),
				grpc.MaxRecvMsgSize(arcGRPC.MaxRecvMsgSize),
				grpc.MaxSendMsgSize(arcGRPC.MaxSendMsgSize),
			)
			// debugging tool
			// reflection.Register(arc.grpcserver)

			pb.RegisterFrameDataServer(arc.grpcserver, &protostream.FrameData{Grpcmessage: arc.grpcmessage})
			err = arc.grpcserver.Serve(arc.listen)
			if err != nil {
				arc.logger.Errorf("grpcserver.Serve: %s", err)
			}
		}()
	}
	<-stop
	arc.Close()
}

// Close kafka & haystack store
func (arc *ArcStorage) Close() {

	arc.once.Do(func() {
		if arc.config.Grpc.Enable {
			close(arc.grpcmessage)
			arc.grpcserver.Stop()
			arc.listen.Close()
		}
		arc.serviceIsClosing = true
		arc.timeoutSyncMap.Range(arc.quitSyncMapWalk)

	})
	// wait for all data to be written to disk.
	time.Sleep(time.Second * 20)
	arc.logger.Info("Quit!")
	arc.arcFileStore.SafeClose()

	// arc cache stop
	if arc.arcCache != nil {
		arc.arcCache.Close()
	}
}

// handleDecodeResult -
func (arc *ArcStorage) handleDecodeResult(sigchan chan os.Signal, workindex int) {
	for {
		select {
		case sig := <-sigchan:
			arc.Close()
			arc.logger.Errorf("Caught signal %v: terminating\n", sig)
		case id := <-arc.timeoutChans[workindex]: // timeout handling.
			arc.loadAndStoreTimeOutData(id)
		case drc := <-arc.decodeResultChans[workindex]:
			r := drc
			if r.err != nil {
				arc.logger.Error(r.err)
				continue
			}

			for _, rItem := range r.items {
				item := rItem
				data2store := make([]byte, item.dataToStoreSize)
				copy(data2store, item.dataToStore[9:])

				datagroup := &protocols.DataGroup{}
				if err := datagroup.Decode(data2store); err != nil {
					arc.logger.Warnw("ProtocolDataGroupDecode", "id", item.idString, "time", item.timestamp, "err", err)
					continue
				}

				var argSegment *protocols.SegmentArc
				var err error
				if argSegment, err = datagroup.GetArcSegment(); err != nil {
					arc.logger.Debugw("arc", "id", item.idString, "err", err)
				}

				arcData := make([]byte, len(argSegment.Data))
				copy(arcData, argSegment.Data)

				if arc.config.Cache.Enable {
					dataPoint := &dataCache.DataPoint{
						ID:   item.idUint64,
						Time: item.timestamp, //us
						Data: argSegment.Data,
					}
					// real-time data caching
					arc.arcCache.Input(dataPoint)
				}

				var afi *arc_volume.ArcVolume

				secondHalfSize := 0

				a, isAfiExist := arc.arcFileStore.DataCache.Load(item.idUint64)
				// store arcVolume for the first time, create a new one if it does not exist.
				if !isAfiExist {
					buffer := bytes.NewBuffer([]byte{})
					buffer.Grow(len(argSegment.Data) * 2)
					buffer.Write(arcData)
					afi = &arc_volume.ArcVolume{
						Dir:        arc.config.Work.DataPath,
						CreateTime: item.timestamp,
						SensorID:   item.idString,
						Buffer:     buffer,
						Type:       TypeArc,
					}

					arc.arcFileStore.DataCache.Store(item.idUint64, afi)

					isAfiExist = false
				}

				if isAfiExist {
					afi = a.(*arc_volume.ArcVolume)
					afi.SaveTime = item.timestamp.UTC() //item.timestamp.UTC()
					afi.LastTimestamp = item.timestamp
					afi.Buffer.Write(arcData)

					if err := arc.arcFileStore.PreWriteToFileCache(afi, item.timestamp, secondHalfSize); err != nil {
						arc.logger.Errorw("storeToArcBigFile", "err", err)
					}
					arc.logger.Debugw("PreWriteToFileCache", "id", item.idString, "timeStamp", item.timestamp, "size", afi.Buffer.Len(), "startTime", afi.CreateTime)

					// add idstirng to gossipKVCache
					if arc.gossipKVCache != nil {
						arc.sensorIDsChan <- []string{item.idString}
					}

					afi.Update(afi.CreateTime)

					afi.Buffer.Write(arcData)

					// store the current time for timeout handling
					arc.timeoutSyncMap.Store(item.idUint64, time.Now().UTC())

				}
			}
		}
	}
}

// gRPCServerStream -
func (arc *ArcStorage) gRPCServerStream(sigchan chan os.Signal, message chan protostream.ProtoStream) {
	isStop := false

	for {
		if isStop {
			break
		}

		select {
		case sig := <-sigchan:
			arc.logger.Errorf("Caught signal %v: terminating\n", sig)
			return

		case mes := <-message:
			isStop = mes.IsStop

			if len(mes.Key) > 0 {
				arc.exportMetrics.SetGRPCLabelValues(fmt.Sprintf("%X", mes.Key[:]), float64(len(mes.Value)))
				arc.decodeJobChans[ByteToUInt64(mes.Key)&uint64(arc.config.Work.WorkCount-1)] <- mes.Value
				arc.logger.Debugw("gRPCServerStream", "key", mes.Key, "bufferSize", len(mes.Value))
			}
		}
	}
}

// loadAndStoreTimeOutData -
func (arc *ArcStorage) loadAndStoreTimeOutData(sensorid uint64) {
	a, isAfiExist := arc.arcFileStore.DataCache.Load(sensorid)
	if !isAfiExist {
		return
	}

	afi := a.(*arc_volume.ArcVolume)
	arc.logger.Debugw("loadAndStoreTimeOutData", "len", afi.Buffer.Len())

	if afi.Buffer.Len() < 1 {
		return
	}

	arc.logger.Warnw("loadAndStoreTimeOutData", "id", afi.SensorID, "len", afi.Buffer.Len(), "afi.CreateTime", afi.CreateTime)
	if err := arc.arcFileStore.PreWriteToFileCache(afi, afi.SaveTime, 0); err != nil {
		arc.logger.Errorw("storeToAduioBigFile", "err", err)
		return
	}
	arc.arcFileStore.DataCache.Delete(sensorid)
	// delete timeout map elem
	arc.timeoutSyncMap.Delete(sensorid)
}

// timeOutSyncMapWalk - Timeout exceeds 1 minute，trigger a certain operation.
func (arc *ArcStorage) timeOutSyncMapWalk(key, value interface{}) bool {
	interaval := time.Now().UTC().Sub(value.(time.Time)).Seconds()
	if interaval >= float64(arc.config.Work.TimeOut) && interaval <= float64(arc.config.Work.TimeOut)+60 {
		arc.logger.Warnw("checkTimeOut", "keyID", key.(uint64))
		arc.timeoutChans[key.(uint64)&uint64(arc.config.Work.WorkCount-1)] <- key.(uint64)
	}

	return true
}

// quitSyncMapWalk - quit
func (arc *ArcStorage) quitSyncMapWalk(key, value interface{}) bool {
	arc.timeoutChans[key.(uint64)&uint64(arc.config.Work.WorkCount-1)] <- key.(uint64)
	return true
}
