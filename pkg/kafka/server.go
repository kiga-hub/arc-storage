package kafka

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/kiga-hub/arc-storage/pkg/config"
	"github.com/kiga-hub/arc/kafka"
	"github.com/kiga-hub/arc/logging"
	"github.com/kiga-hub/arc/protocols"
)

// Handler -
type Handler interface {
	Start(context.Context)
	Write(uint64, *protocols.Frame) error
	Stop()
}

// Server - 时序数据库管理结构
type Server struct {
	kafka           *kafka.Kafka
	srv             *Client
	numericalBuffer *sync.Map
	status          bool
	config          *config.ArcConfig
	logger          logging.ILogger
}

// New -
func New(opts ...Option) (Handler, error) {
	srv := loadOptions(opts...)

	if srv.kafka == nil {
		return nil, fmt.Errorf("new Clinet fail")
	}

	spew.Dump(kafka.GetConfig())
	spew.Dump(srv.config)

	srv.numericalBuffer = new(sync.Map)
	srv.status = true
	srv.srv = NewKafkaClient(srv.logger, srv.kafka, srv.config)

	return srv, nil
}

// Stop -
func (s *Server) Stop() {
	s.numericalBuffer.Range(func(key, value interface{}) bool {
		if err := value.(*ArcBuffer).flush(); err != nil {
			s.logger.Errorw("kafka numerical flush", "err", err)
		}
		s.numericalBuffer.Delete(key)
		return true
	})

	s.srv.client.Close()
	s.logger.Infow("kafka service close")
}

// Start -
func (s *Server) Start(ctx context.Context) {
	ticker := time.NewTicker(time.Second * time.Duration(s.config.Kafka.Interval))
	defer func() {
		ticker.Stop()
	}()

	s.logger.Infow("kafka service start")

	for {
		select {
		case <-ticker.C:
		case <-ctx.Done():
			s.logger.Infow("kafka service stop")
			return
		}
		s.numericalBuffer.Range(func(key, value interface{}) bool {
			b := value.(*ArcBuffer)
			if err := b.flush(); err != nil {
				s.logger.Errorw("kafka nemerical flush", "err", err)
			}
			return true
		})
	}
}

// Write -
func (s *Server) Write(id uint64, frame *protocols.Frame) error {
	var err error
	var sm *protocols.SegmentArc
	for _, stype := range frame.DataGroup.STypes {
		switch stype {
		case protocols.STypeArc:
			sm, err = frame.DataGroup.GetArcSegment()
			if err != nil {
				s.logger.Errorw("GetNumericalTableSegment", "err", err)
				return err
			}
		}
	}

	// 获取Frame中的Flag字段,和传感器ID组合，作为numericalBuffer 的key
	var mode uint16 = 0

	// 数值表数据入库
	if s.config.Kafka.Enable && sm != nil {
		if err := s.sendToKafka(id, mode, frame.Timestamp, sm); err != nil {
			return err
		}
	}
	return nil
}
