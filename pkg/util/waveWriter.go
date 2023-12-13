package util

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

// WriterParam -
type WriterParam struct {
	Out           io.WriteCloser
	Channel       int
	SampleRate    int
	BitsPerSample int
}

// Writer -
type Writer struct {
	out            io.WriteCloser // 实际写出来的文件和bytes等
	writtenSamples int            // 写入的样本数

	riffChunk *RiffChunk
	fmtChunk  *FmtChunk
	dataChunk *DataWriterChunk
}

// New builds a new OGG Opus writer
func New(fileName string, sampleRate int, channelCount int) (*Writer, error) {
	f, err := os.Create(fileName)
	if err != nil {
		return nil, err
	}
	param := WriterParam{
		Out:           f,
		Channel:       channelCount,
		SampleRate:    sampleRate,
		BitsPerSample: 16,
	}
	writer, err := NewWriter(param)
	if err != nil {
		return nil, f.Close()
	}
	return writer, nil
}

// NewBuffer -
func NewBuffer(sampleRate int, channelCount int) (*Writer, error) {
	param := WriterParam{
		Channel:       channelCount,
		SampleRate:    sampleRate,
		BitsPerSample: 16,
	}
	writer, err := NewWriter(param)
	if err != nil {
		return nil, err
	}
	return writer, nil
}

// NewWriter -
func NewWriter(param WriterParam) (*Writer, error) {
	w := &Writer{}
	w.out = param.Out

	blockSize := uint16(param.BitsPerSample*param.Channel) / 8
	samplesPerSec := uint32(int(blockSize) * param.SampleRate)

	// riff chunk
	w.riffChunk = &RiffChunk{
		ID:         []byte(riffChunkToken),
		FormatType: []byte(waveFormatType),
	}
	// fmt chunk
	w.fmtChunk = &FmtChunk{
		ID:   []byte(fmtChunkToken),
		Size: uint32(fmtChunkSize),
	}
	w.fmtChunk.Data = &WavFmtChunkData{
		WaveFormatType: uint16(1), // PCM
		Channel:        uint16(param.Channel),
		SamplesPerSec:  uint32(param.SampleRate),
		BytesPerSec:    samplesPerSec,
		BlockSize:      uint16(blockSize),
		BitsPerSamples: uint16(param.BitsPerSample),
	}
	// data chunk
	w.dataChunk = &DataWriterChunk{
		ID:   []byte(dataChunkToken),
		Data: bytes.NewBuffer([]byte{}),
	}

	return w, nil
}

// WriteSample8 -
func (w *Writer) WriteSample8(samples []uint8) (int, error) {
	buf := new(bytes.Buffer)

	for i := 0; i < len(samples); i++ {
		err := binary.Write(buf, binary.LittleEndian, samples[i])
		if err != nil {
			return 0, err
		}
	}
	n, err := w.Write(buf.Bytes())
	return n, err
}

// WriteSample16 -
func (w *Writer) WriteSample16(samples []int16) (int, error) {
	buf := new(bytes.Buffer)

	for i := 0; i < len(samples); i++ {
		err := binary.Write(buf, binary.LittleEndian, samples[i])
		if err != nil {
			return 0, err
		}
	}
	n, err := w.Write(buf.Bytes())
	return n, err
}

// Write -
func (w *Writer) Write(p []byte) (int, error) {
	blockSize := int(w.fmtChunk.Data.BlockSize)
	if len(p) < blockSize {
		return 0, fmt.Errorf("writing data need at least %d bytes", blockSize)
	}
	// 写入byte数是BlockSize的倍数
	if len(p)%blockSize != 0 {
		return 0, fmt.Errorf("writing data must be a multiple of %d bytes", blockSize)
	}
	num := len(p) / blockSize

	n, err := w.dataChunk.Data.Write(p)

	if err == nil {
		w.writtenSamples += num
	}
	return n, err
}

type errWriter struct {
	w   io.Writer
	err error
}

// Write -
func (ew *errWriter) Write(order binary.ByteOrder, data interface{}) {
	if ew.err != nil {
		return
	}
	ew.err = binary.Write(ew.w, order, data)
}

// Bytes -
func (w *Writer) Bytes() ([]byte, error) {
	out := bytes.NewBuffer([]byte{})

	data := w.dataChunk.Data.Bytes()
	dataSize := uint32(len(data))
	w.riffChunk.Size = uint32(len(w.riffChunk.ID)) + (8 + w.fmtChunk.Size) + (8 + dataSize)
	w.dataChunk.Size = dataSize

	ew := &errWriter{w: out}

	// riff chunk
	ew.Write(binary.BigEndian, w.riffChunk.ID)
	ew.Write(binary.LittleEndian, w.riffChunk.Size)
	ew.Write(binary.BigEndian, w.riffChunk.FormatType)

	// fmt chunk
	ew.Write(binary.BigEndian, w.fmtChunk.ID)
	ew.Write(binary.LittleEndian, w.fmtChunk.Size)
	ew.Write(binary.LittleEndian, w.fmtChunk.Data)

	//data chunk
	ew.Write(binary.BigEndian, w.dataChunk.ID)
	ew.Write(binary.LittleEndian, w.dataChunk.Size)

	if ew.err != nil {
		return nil, ew.err
	}

	_, err := out.Write(data)
	if err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// Close -
func (w *Writer) Close() error {
	if w.out == nil {
		return fmt.Errorf("not find file handle")
	}

	data, err := w.Bytes()
	if err != nil {
		return err
	}

	if _, err := w.out.Write(data); err != nil {
		return err
	}

	err = w.out.Close()
	if err != nil {
		return err
	}

	return nil
}

// PCMToWave -
func PCMToWave(samplerate int, channelCount int, pcm []byte) ([]byte, error) {
	w, err := NewBuffer(samplerate, channelCount)
	if err != nil {
		return nil, err
	}
	if _, err := w.WriteSample8(pcm); err != nil {
		return nil, err
	}
	return w.Bytes()
}
