package pkg

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"time"

	"github.com/kiga-hub/arc/protocols"
)

// parsedFrame 帧数据
type parsedFrame struct {
	timestamp       time.Time
	idString        string
	filenamesuffix  string
	sensorType      string
	id              []byte
	dataToStore     []byte
	protocolType    int
	dataToStoreSize int
	idUint64        uint64
	isInterrupt     bool
	isEnd           bool
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

		f.Version = data[index]
		index++
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

		bytesBuffer = bytes.NewBuffer(data[index : index+6])
		err = binary.Read(bytesBuffer, binary.BigEndian, &f.BasicInfo)
		if err != nil {
			arc.logger.Error("binary.Read(bytesBuffer, binary.BigEndian, &f.BasicInfo)")
			break
		}
		index += 6
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
		copy(f.Firmware[:], data[index:index+3])
		index += 3
		if index >= l {
			arc.logger.Errorf("index:%d >=l:%d break", index, l)
			break
		}

		f.Hardware = data[index]
		index++
		if index >= l {
			arc.logger.Errorf("index:%d >=l:%d break", index, l)
			break
		}
		f.Protocol = binary.BigEndian.Uint16(data[index : index+2])
		// copy(f.Protocol, data[index:index+2])
		index += 2
		if index >= l {
			arc.logger.Errorf("index:%d >=l:%d break", index, l)
			break
		}

		copy(f.Flag[:], data[index:index+3])
		index += 3
		if index >= l {
			arc.logger.Errorf("index:%d >=l:%d break", index, l)
			break
		}
		// Firmware, Hardware, Protocol, Flag, DataGroup
		dataToStore := data[index-9 : index+int(f.Size)-32]

		if len(data) < index+int(f.Size)-32 {
			arc.logger.Errorf("len(data):%d < index:%d", len(data), index)
			break
		}

		err = f.DataGroup.Decode(data[index : index+int(f.Size)-32])
		if err != nil {
			arc.logger.Error("DataGroup.Decode", err)
			break
		}
		index += int(f.Size) - 32
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
		// copy(f.End[:], data[index:index+1])
		index++

		//TODO make size limitation a configuration field of arc storage
		if f.Size > math.MaxUint32/2 { //1024*1024
			arc.logger.Errorf("data is too large %d  break", f.Size)
			break
		}

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
	// 协议类型 1-数据包只包含音频数据  2-包含音频、温度、振动数据
	protocolType := int(f.Protocol)
	// 设备类型 sensortype
	firmware := fmt.Sprintf("%X", f.Firmware)
	hardware := fmt.Sprintf("%02X", f.Hardware)
	flag := fmt.Sprintf("%X", f.Flag)
	suffix := firmware + "_" + hardware + "_" + flag

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
		sensorType:      firmware,
		protocolType:    protocolType,
		timestamp:       timestamp,
		dataToStore:     dataToSave,
		dataToStoreSize: len(dataToSave),
		filenamesuffix:  suffix,
		isInterrupt:     !f.IsTimeAlign(),
		isEnd:           f.IsEnd(),
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

// ParsedFrameConstructor - 构造Frame结构，用于上传文件,当前版本支持上传音频振动文件
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
	framePackage.SetProto(2)
	framePackage.SetID(id)
	framePackage.Timestamp = timestamp.UnixNano() / 1e3
	framePackage.Hardware = 0x01
	framePackage.Firmware[0] = 0x0F
	framePackage.SetEndFlag()
	framePackage.SetDataGroup(group)
	framePackage.SetTimeAlignFlag()

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
