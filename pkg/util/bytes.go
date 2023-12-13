package util

import (
	"crypto/md5"
	"fmt"
	"io"
)

// BytesToHumanReadable returns the converted human readable representation of the bytes.
func BytesToHumanReadable(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}

	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.2f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

// big endian

// BytesToUint64 convert bytes to uint64
func BytesToUint64(b []byte) (v uint64) {
	length := uint(len(b))
	for i := uint(0); i < length-1; i++ {
		v += uint64(b[i])
		v <<= 8
	}
	v += uint64(b[length-1])
	return
}

// BytesToUint32 convert bytes to uint32
func BytesToUint32(b []byte) (v uint32) {
	length := uint(len(b))
	for i := uint(0); i < length-1; i++ {
		v += uint32(b[i])
		v <<= 8
	}
	v += uint32(b[length-1])
	return
}

// BytesToUint16 convert bytes to uint64
func BytesToUint16(b []byte) (v uint16) {
	v += uint16(b[0])
	v <<= 8
	v += uint16(b[1])
	return
}

// Uint64toBytes convert uint64 to bytes
func Uint64toBytes(b []byte, v uint64) {
	for i := uint(0); i < 8; i++ {
		b[7-i] = byte(v >> (i * 8))
	}
}

// Uint32toBytes convert uint32 to bytes
func Uint32toBytes(b []byte, v uint32) {
	for i := uint(0); i < 4; i++ {
		b[3-i] = byte(v >> (i * 8))
	}
}

// Uint16toBytes convert uint16 to bytes
func Uint16toBytes(b []byte, v uint16) {
	b[0] = byte(v >> 8)
	b[1] = byte(v)
}

// Uint8toBytes convert uint8 to bytes
func Uint8toBytes(b []byte, v uint8) {
	b[0] = byte(v)
}

// HashStringToLong returns a 64 bit big int
func HashStringToLong(dir string) (v int64, err error) {
	h := md5.New()
	_, err = io.WriteString(h, dir)
	if err != nil {
		return
	}
	b := h.Sum(nil)
	v += int64(b[0])
	v <<= 8
	v += int64(b[1])
	v <<= 8
	v += int64(b[2])
	v <<= 8
	v += int64(b[3])
	v <<= 8
	v += int64(b[4])
	v <<= 8
	v += int64(b[5])
	v <<= 8
	v += int64(b[6])
	v <<= 8
	v += int64(b[7])

	return
}

// HashToInt32 convert bytes to int32
func HashToInt32(data []byte) (v int32, err error) {
	h := md5.New()
	_, err = h.Write(data)
	if err != nil {
		return
	}
	b := h.Sum(nil)

	v += int32(b[0])
	v <<= 8
	v += int32(b[1])
	v <<= 8
	v += int32(b[2])
	v <<= 8
	v += int32(b[3])

	return
}

// Md5 check md5
func Md5(data []byte) (string, error) {
	hash := md5.New()
	_, err := hash.Write(data)
	if err != nil {
		return "", nil
	}
	return fmt.Sprintf("%X", hash.Sum(nil)), nil
}
