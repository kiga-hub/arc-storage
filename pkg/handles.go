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
	// TDEngineAPI tdengine api
	TDEngineAPI = "Temperature etc."
	// DataBaseName arc
	DataBaseName = "arc"
)

// ArcStorageVersion add to file suffix name
var ArcStorageVersion = "appversion"

const (
	// TypeTemperature "T"
	TypeTemperature = "T"
)

// RecordTimeRangeParam -
type RecordTimeRangeParam struct {
	Result string      `json:"result" xml:"result" form:"result" query:"result"`
	Time   []TimeRange `json:"time"  xml:"time" form:"time" query:"time"`
}

// TimeRange -
type TimeRange struct {
	From string `json:"from" xml:"from" form:"from" query:"from"`
	To   string `json:"to" xml:"to" form:"to" query:"to"`
}

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
	audioFileStore    *arc_volume.ArcVolumeCache // 音频读写文件系统
	audioCache        *cache.DataCacheRepo       // 查询实时Audio数据的缓存
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
	// 配置数据大小校验
	err := protocols.ConfigFrame(math.MaxUint32 / 2)
	if err != nil {
		return nil, err
	}

	audioFileStore, err := arc_volume.NewArcVolumeCache(logger, config, arc_volume.DataTypeMap[TypeTemperature])
	if err != nil {
		return nil, err
	}

	// 初始化指标采集模块
	pi := monitor.NewPackageInterrupt()
	src := monitor.NewSampleRateChanged()
	g := monitor.NewGRPC()
	order := monitor.NewOutOfOrder()
	cacheRead := monitor.NewCacheRead()
	m, err := metric.NewHandlerMonitor(pi, src, g, order, cacheRead)
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
		audioFileStore:    audioFileStore,
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
		// 实时查询
		db.audioCache = cache.NewCacheRepo(logger)
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
			// 定时检查是否超时
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
		// channel处理最优分配时，timeoutChan只需要分配传感器数量的容量
		arc.timeoutChans[i] = make(chan uint64, arc.config.Work.ChanCapacity)
		go arc.decodeWorker(arc.decodeJobChans[i], arc.decodeResultChans[i])
		go arc.handleDecodeResult(sigchan, i)
	}

	arc.serviceIsClosing = false

	// 检查超时
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
	// 等待所有数据落盘
	time.Sleep(time.Second * 20)
	arc.logger.Info("Quit!")
	arc.audioFileStore.SafeClose()

	// audio cache stop
	if arc.audioCache != nil {
		arc.audioCache.Close()
	}
}

// handleDecodeResult -
func (arc *ArcStorage) handleDecodeResult(sigchan chan os.Signal, workindex int) {
	for {
		select {
		case sig := <-sigchan:
			arc.Close()
			arc.logger.Errorf("Caught signal %v: terminating\n", sig)
		case id := <-arc.timeoutChans[workindex]: // 超时处理
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

				var temperature *protocols.SegmentTemperature
				var err error
				if temperature, err = datagroup.GetTSegment(); err != nil {
					arc.logger.Debugw("temperature", "id", item.idString, "err", err)
				}

				audiodata := make([]byte, len(temperature.Bytes))
				copy(audiodata, temperature.Bytes)

				if arc.config.Cache.Enable {
					// 音频缓存
					dataPoint := &dataCache.DataPoint{
						ID:   item.idUint64,
						Time: item.timestamp, //us
						Data: temperature.Bytes,
					}
					// 实时数据缓存
					arc.audioCache.Input(dataPoint)
				}

				var afi *arc_volume.ArcVolume

				firstHalfSize := 0
				secondHalfSize := 0
				isRateChanged := false
				isStopMinChanged := false

				a, isAfiExist := arc.audioFileStore.DataCache.Load(item.idUint64)
				// 首次存fileCache，不存在则新建
				if !isAfiExist {
					buffer := bytes.NewBuffer([]byte{})
					buffer.Grow(len(temperature.Bytes) * 2)
					// 写入音频buffer
					buffer.Write(audiodata)
					afi = &arc_volume.ArcVolume{
						Dir:        arc.config.Work.DataPath,
						CreateTime: item.timestamp,
						FirmWare:   item.filenamesuffix,
						SensorID:   item.idString,
						Buffer:     buffer,
						Version:    ArcStorageVersion,
						Type:       TypeTemperature,
					}

					if item.isEnd {
						// 收到得第一个包为强制落盘,写入文件存储
						if err := arc.audioFileStore.PreWriteToFileCache(afi, item.timestamp, secondHalfSize); err != nil {
							arc.logger.Errorw("storeToAduioBigFile", "err", err)
						}
						afi = nil
					} else {
						arc.audioFileStore.DataCache.Store(item.idUint64, afi)
					}

					isAfiExist = false
				}

				if isAfiExist {
					afi = a.(*arc_volume.ArcVolume)

					if item.timestamp.Before(afi.LastTimestamp) {
						arc.exportMetrics.SetOutOfOrderLabelValues(item.idString)
						arc.logger.Errorw("timestampOutOfOrder", "sensorID", item.idString, "lastTimestamp", afi.LastTimestamp,
							"itemTimeStamp", item.timestamp, "isInterrupt", item.isInterrupt)
					}
					// 判断包号是否连续或者中断结束
					if item.isInterrupt || item.isEnd {
						arc.exportMetrics.SetInterruptLabelValues(item.idString)
						arc.logger.Debugw("packageIsInterrupt", "id", item.idString, "time", item.timestamp, "isInterrupt", item.isInterrupt, "isEnd", item.isEnd)
					}

					// 采样率不变 & 包号连续 | 强制落盘,解析数据写入buffer
					if (!isRateChanged && !item.isInterrupt) || item.isEnd {
						afi.Buffer.Write(audiodata)
					}

					// 每分钟落盘，buffer写入文件
					// 数据包不连续，且接收到的为第一个包则不进行落盘操作
					if (afi.Buffer.Len() > 0 && item.isInterrupt) || isRateChanged || isStopMinChanged || item.isEnd {
						arc.logger.Infow("saveAudioDataToFile", "sensorID", item.idString, "itemTimestamp", item.timestamp,
							"FileCreateTime", afi.CreateTime, "FileSaveTime", afi.SaveTime.UTC(),
							"isInterrupt", item.isInterrupt, "sampleRateChanged", isRateChanged, "isStopMinChanged", isStopMinChanged, "isEnd", item.isEnd,
							"dataSize", len(audiodata), "afiSize", afi.Buffer.Len(), "sampleRate", afi.SampleRate)

						if item.timestamp.Before(afi.SaveTime) {
							arc.logger.Debugw(
								"storeDataTimestampError", "sensorID", item.idString, "timeStamp", item.timestamp,
								"createTime", afi.CreateTime, "saveTime", afi.SaveTime.UTC(),
							)
						}

						// 存储音频数据到文件系统
						if err := arc.audioFileStore.PreWriteToFileCache(afi, item.timestamp, secondHalfSize); err != nil {
							arc.logger.Errorw("storeToAduioBigFile", "err", err)
						}
						arc.logger.Debugw("PreWriteToFileCache", "id", item.idString, "timeStamp", item.timestamp, "size", afi.Buffer.Len(), "startTime", afi.CreateTime)

						// add idstirng to gossipKVCache
						if arc.gossipKVCache != nil {
							arc.sensorIDsChan <- []string{item.idString}
						}

						afi.Update(afi.CreateTime)
					}

					afi.SaveTime = item.timestamp.UTC() //item.timestamp.UTC()
					afi.LastTimestamp = item.timestamp
					afi.Buffer.Write(audiodata)

					// 包号不连续，采样率变化，保存数据到下一个缓存中
					if item.isInterrupt || isRateChanged {
						afi.Buffer.Write(audiodata)
						afi.CreateTime = item.timestamp.UTC()
					} else if secondHalfSize > 0 {
						arc.logger.Infow("secondHalfSize", "sensorID", item.idString, "firstHalfSizeSize", firstHalfSize, "secondHalfSizeSize", secondHalfSize, "afiCreateTime", afi.CreateTime.UTC(), "afiSaveTime", afi.SaveTime.UTC())
						afi.Buffer.Write(audiodata[firstHalfSize:])
					}

					if item.isEnd {
						arc.audioFileStore.DataCache.Delete(item.idUint64)
					} else {
						// 存储当前时间，用于超时处理
						arc.timeoutSyncMap.Store(item.idUint64, time.Now().UTC())
					}
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
	a, isAfiExist := arc.audioFileStore.DataCache.Load(sensorid)
	if !isAfiExist {
		return
	}

	afi := a.(*arc_volume.ArcVolume)
	arc.logger.Debugw("loadAndStoreTimeOutData", "len", afi.Buffer.Len())

	if afi.Buffer.Len() < 1 {
		return
	}

	arc.logger.Warnw("loadAndStoreTimeOutData", "id", afi.SensorID, "len", afi.Buffer.Len(), "afi.CreateTime", afi.CreateTime)
	if err := arc.audioFileStore.PreWriteToFileCache(afi, afi.SaveTime, 0); err != nil {
		arc.logger.Errorw("storeToAduioBigFile", "err", err)
		return
	}
	arc.audioFileStore.DataCache.Delete(sensorid)
	// delete timeout map elem
	arc.timeoutSyncMap.Delete(sensorid)
}

// timeOutSyncMapWalk - 超时大于1分钟，进行落盘
func (arc *ArcStorage) timeOutSyncMapWalk(key, value interface{}) bool {
	// 定时1分钟判断是否超时，但是落盘也是按照1分钟进行落盘，时间有可能会重叠，由60s，改为90s
	// TODO 待测
	interaval := time.Now().UTC().Sub(value.(time.Time)).Seconds()
	if interaval >= float64(arc.config.Work.TimeOut) && interaval <= float64(arc.config.Work.TimeOut)+60 {
		arc.logger.Warnw("checkTimeOut", "keyID", key.(uint64))
		arc.timeoutChans[key.(uint64)&uint64(arc.config.Work.WorkCount-1)] <- key.(uint64)
	}

	return true
}

// quitSyncMapWalk - 退出进行落盘
func (arc *ArcStorage) quitSyncMapWalk(key, value interface{}) bool {
	arc.timeoutChans[key.(uint64)&uint64(arc.config.Work.WorkCount-1)] <- key.(uint64)
	return true
}
