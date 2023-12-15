package pkg

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/kiga-hub/arc/protocols"
)

// parsedFrame 帧数据
type parsedFrame struct {
	timestamp       time.Time
	idString        string
	filenamesuffix  string
	id              []byte
	dataToStore     []byte
	dataToStoreSize int
	idUint64        uint64
}

// decodeWorker 解析
func (arc *ArcStorage) decodeWorker(input chan []byte, output chan decodeResult) {
	for j := range input {
		output <- arc.decode(j)
	}
}

// decodeResult 解析结果
type decodeResult struct {
	err   error
	items []*parsedFrame
}

// decode 解析
func (arc *ArcStorage) decode(srcdata []byte) decodeResult {
	result := decodeResult{
		items: []*parsedFrame{},
	}
	l := len(srcdata)

	data := make([]byte, l)
	copy(data, srcdata)

	for index := 0; index < l; {
		f := &protocols.Frame{}
		copy(f.Head[:], data[index:index+4])
		if !bytes.Equal(f.Head[:], protocols.Head[:]) {
			// utils.Hexdump("test", data)
			arc.logger.Errorf("bytes.Equal(f.Head[:]:%s, protocols.Head[:]:%s ,%d)", fmt.Sprintf("%X", f.Head[:]), fmt.Sprintf("%X", protocols.Head[:]), index)
			break
		}

		index += 4
		if index >= l {
			arc.logger.Errorf("index:%d >=l:%d break", index, l)
			break
		}

		bytesBuffer := bytes.NewBuffer(data[index : index+4])
		err := binary.Read(bytesBuffer, binary.BigEndian, &f.Size)
		if err != nil {
			arc.logger.Error("binary.Read(bytesBuffer, binary.BigEndian, &f.Size)")
			break
		}

		index += 4
		if index >= l {
			arc.logger.Errorf("index:%d >=l:%d break", index, l)
			break
		}

		bytesBuffer = bytes.NewBuffer(data[index : index+8])
		err = binary.Read(bytesBuffer, binary.BigEndian, &f.Timestamp)
		if err != nil {
			arc.logger.Error("binary.Read(bytesBuffer, binary.BigEndian, &f.Timestamp)")
			break
		}
		index += 8
		if index >= l {
			arc.logger.Errorf("index:%d >=l:%d break", index, l)
			break
		}

		copy(f.ID[:], data[index:index+6])
		index += 6
		if index >= l {
			arc.logger.Errorf("index:%d >=l:%d break", index, l)
			break
		}

		//
		dataToStore := data[index-protocols.DefaultHeadLength : index+int(f.Size)-int(protocols.LengthWithoutData)]
		fmt.Println("dataToStore len :", len(dataToStore))
		fmt.Println("dataToStore:", dataToStore)

		if len(data) < index+int(f.Size)-int(protocols.LengthWithoutData) {
			arc.logger.Errorf("len(data):%d < index:%d", len(data), index)
			break
		}

		err = f.DataGroup.Decode(data[index : index+int(f.Size)-int(protocols.LengthWithoutData)])
		if err != nil {
			arc.logger.Error("DataGroup.Decode", err)
			break
		}
		index += int(f.Size) - int(protocols.LengthWithoutData)
		if index >= l {
			arc.logger.Errorf("index:%d >=l:%d break", index, l)
			break
		}

		bytesBuffer = bytes.NewBuffer(data[index : index+2])
		err = binary.Read(bytesBuffer, binary.BigEndian, &f.Crc)
		if err != nil {
			arc.logger.Error("binary.Read(bytesBuffer, binary.BigEndian, &f.Crc)")
			break
		}
		index += 2
		if index >= l {
			arc.logger.Errorf("index:%d >=l:%d break", index, l)
			break
		}
		f.End = data[len(data)-1]
		index++

		pf, err := arc.parseFrame(f, dataToStore)
		if err == nil && pf != nil {
			result.items = append(result.items, pf)
			f = nil
		} else if err != nil {
			arc.logger.Errorf("parseFrame\n", err)
		}
	}

	return result
}

// parseFrame TODO move to common project
func (arc *ArcStorage) parseFrame(f *protocols.Frame, dataToSave []byte) (*parsedFrame, error) {
	// ID 94c96000c248 []byte{0x94,0xC9,0x60,0x00,0xC2,0x48}
	idUint64 := ByteToUInt64(f.ID[:])

	idString := string(fmt.Sprintf("%X", f.ID[:]))

	suffix := idString + "_" + idString

	timestamp := time.Unix(0, f.Timestamp*1e3)

	if arc.kafka != nil {
		if err := arc.kafka.Write(idUint64, f); err != nil {
			arc.logger.Errorw("WriteTokafkaErr", "err", err)
		}
	}

	return &parsedFrame{
		id:              f.ID[:],
		idString:        idString,
		idUint64:        idUint64,
		timestamp:       timestamp,
		dataToStore:     dataToSave,
		dataToStoreSize: len(dataToSave),
		filenamesuffix:  suffix,
	}, nil
}

// ByteToUInt64 传感器[]byte 转为uint64
func ByteToUInt64(sensorID []byte) uint64 {
	return uint64(sensorID[5]) |
		uint64(sensorID[4])<<8 |
		uint64(sensorID[3])<<16 |
		uint64(sensorID[2])<<24 |
		uint64(sensorID[1])<<32 |
		uint64(sensorID[0])<<40
}

// ParsedFrameConstructor - 构造Frame结构
func ParsedFrameConstructor(segmentData protocols.ISegment, id uint64, timestamp time.Time) ([]byte, error) {
	group := protocols.NewDefaultDataGroup()

	// 添加数据段到组
	group.AppendSegment(segmentData)
	if err := group.Validate(); err != nil {
		return nil, err
	}

	fmt.Printf("ParsedFrameConstructor: time: %v,id: %d, size: %d", timestamp, id, segmentData.Size())
	// 添加组到包
	framePackage := protocols.NewDefaultFrame()
	framePackage.SetID(id)
	framePackage.Timestamp = timestamp.UnixNano() / 1e3
	framePackage.SetDataGroup(group)

	// 打包二进制
	buf := make([]byte, framePackage.Size+9)
	_, err := framePackage.Encode(buf)
	if err != nil {
		return nil, err
	}

	// 数据包有效判断
	if err := protocols.FrameValidate(buf); err != nil {
		return nil, err
	}

	return buf, nil
}
