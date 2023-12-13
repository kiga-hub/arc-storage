package component

import (
	"context"

	"github.com/kiga-hub/arc-storage/pkg"
	"github.com/kiga-hub/arc-storage/pkg/config"
	"github.com/kiga-hub/arc-storage/pkg/kafka"

	platformConf "github.com/kiga-hub/arc/conf"
	"github.com/kiga-hub/arc/configuration"
	kafkaComponent "github.com/kiga-hub/arc/kafka"
	"github.com/kiga-hub/arc/logging"
	"github.com/kiga-hub/arc/micro"
	microComponent "github.com/kiga-hub/arc/micro/component"
	microConf "github.com/kiga-hub/arc/micro/conf"
	"github.com/pangpanglabs/echoswagger/v2"
)

// ArcStorageElementKey is Element Key for arc storage
var ArcStorageElementKey = micro.ElementKey("ArcStorageComponent")

// ArcStorageComponent is Component for arc storage
type ArcStorageComponent struct {
	micro.EmptyComponent
	stopChan      chan struct{}
	handler       *pkg.ArcStorage
	logger        logging.ILogger
	nacosClient   *configuration.NacosClient
	gossipKVCache *microComponent.GossipKVCacheComponent
	kafka         kafka.Handler
	cluster       string
}

// Name of the component
func (c *ArcStorageComponent) Name() string {
	return "ArcStorageComponent"
}

// PreInit called before Init()
func (c *ArcStorageComponent) PreInit(ctx context.Context) error {
	// load config
	config.SetDefaultWorkConfig()
	return nil
}

// Init the component
func (c *ArcStorageComponent) Init(server *micro.Server) error {
	// init
	basicConf := microConf.GetBasicConfig()
	c.cluster = server.PrivateCluster
	if basicConf.IsDynamicConfig {
		c.nacosClient = server.GetElement(&micro.NacosClientElementKey).(*configuration.NacosClient)
	}
	elkvcache := server.GetElement(&micro.GossipKVCacheElementKey)
	if elkvcache != nil {
		c.gossipKVCache = elkvcache.(*microComponent.GossipKVCacheComponent)
	}
	logger := server.GetElement(&micro.LoggingElementKey).(logging.ILogger)
	c.logger = logger

	basicConfig := config.GetConfig()
	var err error
	c.stopChan = make(chan struct{})
	// New Kafka
	elkafka := server.GetElement(&kafkaComponent.ElementKey)
	if elkafka != nil {
		if c.kafka, err = kafka.New(
			kafka.WithKafka(elkafka),
			kafka.WithLogger(c.logger),
		); err != nil {
			c.logger.Errorw("NewServer", "err", err)
			return err
		}
	} else {
		c.logger.Warnw("GetElement", "GetElement", "is null")
	}

	c.handler, err = pkg.NewArcStorage(basicConfig, c.logger, c.gossipKVCache, c.kafka)
	if err != nil {
		return err
	}

	return nil
}

// OnConfigChanged called when dynamic config changed
func (c *ArcStorageComponent) OnConfigChanged(*platformConf.NodeConfig) error {
	return micro.ErrNeedRestart
}

// SetupHandler of echo if the component need
func (c *ArcStorageComponent) SetupHandler(root echoswagger.ApiRoot, base string) error {
	basicConf := microConf.GetBasicConfig()
	selfServiceName := basicConf.Service
	c.handler.SetupWeb(root, base, selfServiceName)

	return nil
}

// Start the component
func (c *ArcStorageComponent) Start(ctx context.Context) error {
	if c.kafka != nil {
		go c.kafka.Start(ctx)
	}

	// start
	go c.handler.Start(c.stopChan)
	return nil
}

// Stop the component
func (c *ArcStorageComponent) Stop(ctx context.Context) error {

	if c.kafka != nil {
		c.kafka.Stop()
	}

	// stop
	c.stopChan <- struct{}{}
	return nil
}
