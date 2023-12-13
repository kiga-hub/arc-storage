package util

import (
	"encoding/binary"
)

func littleEndianIntToHex(integer int, numberOfBytes int) (bytes []byte) {
	bytes = make([]byte, numberOfBytes)
	switch numberOfBytes {
	case 2:
		binary.LittleEndian.PutUint16(bytes, uint16(integer))
	case 4:
		binary.LittleEndian.PutUint32(bytes, uint32(integer))
	}
	return
}

func applyString(dst []byte, s string, numberOfBytes int) {
	copy(dst, []byte(s)[:numberOfBytes])
}

func applyLittleEndianInteger(dst []byte, i int, numberOfBytes int) {
	copy(dst, littleEndianIntToHex(i, numberOfBytes)[0:numberOfBytes])
}

type riffChunk struct {
	ChunkID   [4]byte
	ChunkSize [4]byte
	Format    [4]byte
}

func (rc *riffChunk) applyChunkID(chunkID string) {
	applyString(rc.ChunkID[:], chunkID, 4)
}

func (rc *riffChunk) applyChunkSize(chunkSize int) {
	applyLittleEndianInteger(rc.ChunkSize[:], chunkSize, 4)
}

func (rc *riffChunk) applyFormat(format string) {
	applyString(rc.Format[:], format, 4)
}

type fmtSubChunk struct {
	Subchunk1Id   [4]byte
	Subchunk1Size [4]byte
	AudioFormat   [2]byte
	NumChannels   [2]byte
	SampleRate    [4]byte
	ByteRate      [4]byte
	BlockAlign    [2]byte
	BitsPerSample [2]byte
}

func (c *fmtSubChunk) applySubchunk1Id(subchunk1Id string) {
	applyString(c.Subchunk1Id[:], subchunk1Id, 4)
}

func (c *fmtSubChunk) applySubchunk1Size(subchunk1Size int) {
	applyLittleEndianInteger(c.Subchunk1Size[:], subchunk1Size, 4)
}

func (c *fmtSubChunk) applyAudioFormat(audioFormat int) {
	applyLittleEndianInteger(c.AudioFormat[:], audioFormat, 2)
}

func (c *fmtSubChunk) applyNumChannels(numChannels int) {
	applyLittleEndianInteger(c.NumChannels[:], numChannels, 2)
}

func (c *fmtSubChunk) applySampleRate(sampleRate int) {
	applyLittleEndianInteger(c.SampleRate[:], sampleRate, 4)
}

func (c *fmtSubChunk) applyByteRate(byteRate int) {
	applyLittleEndianInteger(c.ByteRate[:], byteRate, 4)
}

func (c *fmtSubChunk) applyBlockAlign(blockAlign int) {
	applyLittleEndianInteger(c.BlockAlign[:], blockAlign, 2)
}

func (c *fmtSubChunk) applyBitsPerSample(bitsPerSample int) {
	applyLittleEndianInteger(c.BitsPerSample[:], bitsPerSample, 2)
}

type dataSubChunk struct {
	Subchunk2Id   [4]byte
	Subchunk2Size [4]byte
}

func (c *dataSubChunk) applySubchunk2Id(subchunk2Id string) {
	applyString(c.Subchunk2Id[:], subchunk2Id, 4)
}

func (c *dataSubChunk) applySubchunk2Size(subchunk2Size int) {
	applyLittleEndianInteger(c.Subchunk2Size[:], subchunk2Size, 4)
}

// ConvertPCMToWav convert pcm bytes to wav
func ConvertPCMToWav(pcm []byte, channels, sampleRate, bitsPerSample int) ([]byte, error) {
	header, err := ConvertPCMToWavHeader(len(pcm), channels, sampleRate, bitsPerSample)
	if err != nil {
		return nil, err
	}
	header = append(header, pcm...)

	return header, err
}

// ConvertPCMToWavHeader convert pcm bytes to a wav header
func ConvertPCMToWavHeader(pcmLength, channels int, sampleRate int, bitsPerSample int) (header []byte, err error) {
	// if channels != 1 && channels != 2 {
	// 	return wav, errors.New("invalid_channels_value")
	// }
	// if sampleRate != 8000 && sampleRate != 16000 && sampleRate != 32000 && sampleRate != 11025 && sampleRate != 6000 && sampleRate != 22050 && sampleRate != 64000 {
	// 	sampleRate = 32000
	// }
	if bitsPerSample != 8 && bitsPerSample != 16 && bitsPerSample != 24 && bitsPerSample != 32 {
		bitsPerSample = 16
	}

	subchunk1Size := 16
	subchunk2Size := pcmLength
	chunkSize := 4 + (8 + subchunk1Size) + (8 + subchunk2Size)

	rc := riffChunk{}
	rc.applyChunkID("RIFF")
	rc.applyChunkSize(chunkSize)
	rc.applyFormat("WAVE")

	fsc := fmtSubChunk{}
	fsc.applySubchunk1Id("fmt ")
	fsc.applySubchunk1Size(subchunk1Size)
	fsc.applyAudioFormat(1)
	fsc.applyNumChannels(channels)
	fsc.applySampleRate(sampleRate)
	fsc.applyByteRate(sampleRate * channels * bitsPerSample / 8)
	fsc.applyBlockAlign(channels * bitsPerSample / 8)
	fsc.applyBitsPerSample(bitsPerSample)

	dsc := dataSubChunk{}
	dsc.applySubchunk2Id("data")
	dsc.applySubchunk2Size(subchunk2Size)

	header = make([]byte, 0, 64)

	header = append(header, rc.ChunkID[:]...)
	header = append(header, rc.ChunkSize[:]...)
	header = append(header, rc.Format[:]...)

	header = append(header, fsc.Subchunk1Id[:]...)
	header = append(header, fsc.Subchunk1Size[:]...)
	header = append(header, fsc.AudioFormat[:]...)
	header = append(header, fsc.NumChannels[:]...)
	header = append(header, fsc.SampleRate[:]...)
	header = append(header, fsc.ByteRate[:]...)
	header = append(header, fsc.BlockAlign[:]...)
	header = append(header, fsc.BitsPerSample[:]...)

	header = append(header, dsc.Subchunk2Id[:]...)
	header = append(header, dsc.Subchunk2Size[:]...)

	return header, err
}

//Prototol version1
// func reOrderArray(data []byte, LR int) []byte {
// 	binary := make([]byte, len(data)/2)
// 	index := 0
// 	for i := 0; i < len(data); i++ {
// 		if i%4 == 0 && LR == 1 {
// 			binary[index] = data[i]
// 			index++
// 			binary[index] = data[i+1]
// 			index++
// 		}
// 		if i%4 == 2 && LR == 0 {
// 			binary[index] = data[i]
// 			index++
// 			binary[index] = data[i+1]
// 			index++
// 		}
// 	}
// 	return binary
// }
