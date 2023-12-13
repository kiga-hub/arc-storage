package util

import (
	"testing"
	"time"
)

func CaseMaybeSlowdonw(t *testing.T) {
	d, _ := time.ParseDuration("-1m")
	tests := []struct {
		bytesPerSecond int64
		delta          int64
		throttler      *WriteThrottler
	}{
		{
			bytesPerSecond: int64(1),
			delta:          int64(1),
			throttler: &WriteThrottler{
				compactionBytePerSecond: int64(1),
				lastSizeCheckTime:       time.Now().Add(d),
			},
		},
		{
			bytesPerSecond: int64(1),
			delta:          int64(1),
			throttler: &WriteThrottler{
				compactionBytePerSecond: int64(1),
				lastSizeCheckTime:       time.Now(),
			},
		},
	}
	for _, tt := range tests {
		_ = NewWriteThrottler(tt.bytesPerSecond)
		tt.throttler.MaybeSlowdown(tt.delta)
	}
}

func TestThrottler(t *testing.T) {
	CaseMaybeSlowdonw(t)
}
