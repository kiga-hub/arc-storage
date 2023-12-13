package kafka

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/kiga-hub/arc-storage/pkg/config"
	arcKafka "github.com/kiga-hub/arc/kafka"
	"github.com/kiga-hub/arc/logging"
)

// kafkaMessage -
type kafkaMessage struct {
	Ts   int64         `json:"ts"`   // 消息推送时间
	Data []MessageData `json:"data"` // 数据
}

// MessageData -
type MessageData struct {
	Ts          int64   `json:"ts"`
	SensorID    string  `json:"sensorID"`    // 传感器ID
	Temperature float64 `json:"temperature"` // 温度
}

// Client -
type Client struct {
	client *arcKafka.Kafka
	config *config.ArcConfig
	logger logging.ILogger
}

// NewKafkaClient -
func NewKafkaClient(logger logging.ILogger, k *arcKafka.Kafka, config *config.ArcConfig) *Client {
	return &Client{
		client: k,
		config: config,
		logger: logger,
	}
}

// SendResultToKafka -
func (kc *Client) SendResultToKafka(arr *NumericalKafkaBuffer) error {
	if kc.client == nil {
		return fmt.Errorf("kafka is nil")
	}
	strArray := make([]MessageData, arr.count)
	for i := 0; i < int(arr.count); i++ {
		strArray[i] = MessageData{
			Ts:          arr.rows[i].Ts,
			SensorID:    arr.rows[i].SensorID,
			Temperature: arr.rows[i].Temperature,
		}
	}

	msg := &kafkaMessage{
		Ts:   time.Now().UnixMilli(),
		Data: strArray,
	}

	notice, err := json.Marshal(msg)
	if err != nil {
		kc.logger.Errorw("serialization", "err", err)
		return err
	}
	kc.client.ProduceDataWithTimeKey(kc.config.Kafka.Topic, notice)

	return nil
}
