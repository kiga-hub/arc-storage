package kafka

import (
	"encoding/binary"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/kiga-hub/arc/protocols"
	"github.com/spf13/cast"
)

// ArcBuffer -
type ArcBuffer struct {
	srv      *Client
	sensorID uint64
	max      int32
	lasttime int64
	mutex    sync.Mutex
	count    int32
	rows     []MessageData
}

// ArcBuffer insert insert arc data to database
func (n *ArcBuffer) insert() error {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	defer atomic.StoreInt32(&n.count, 0)

	// send to kafka
	if err := n.srv.SendResultToKafka(n); err != nil {
		return err
	}

	return nil
}

// flush -
func (n *ArcBuffer) flush() error {
	if n.count > n.max {
		return fmt.Errorf("arc count more than max")
	}
	if atomic.LoadInt32(&n.count) > 0 {
		return n.insert()
	}
	return nil
}

// append -
func (n *ArcBuffer) append(data ...MessageData) error {
	n.mutex.Lock()
	defer n.mutex.Unlock()

	for i, v := range data {
		if i == 0 && v.Ts == n.lasttime {
			continue
		}

		// 缓存
		count := atomic.LoadInt32(&n.count)
		if count >= n.max {
			return nil
		}

		n.lasttime = v.Ts
		n.rows[count] = v
		atomic.AddInt32(&n.count, 1)

	}
	return nil
}

// sendToKafka -
func (s *Server) sendToKafka(id uint64, firmware uint16, timestamp int64, sm *protocols.SegmentArc) error {
	if sm == nil || !s.status {
		return nil
	}
	key := uint64(firmware) + (id << 2)
	buff, ok := s.numericalBuffer.Load(key)
	if !ok {
		buff, _ = s.numericalBuffer.LoadOrStore(key,
			&ArcBuffer{
				sensorID: id,
				max:      8192,
				rows:     make([]MessageData, 8192),
				srv:      s.srv,
			},
		)
	}

	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, id)
	sensorID := fmt.Sprintf("%X", b[2:])

	return buff.(*ArcBuffer).append(MessageData{
		Ts:       timestamp / 1e3,
		SensorID: sensorID,
		ArcData:  cast.ToFloat64(sm.Data),
	})
}
