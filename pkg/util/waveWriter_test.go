package util

import (
	"io/ioutil"
	"os"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func CaseNew(t *testing.T) {
	dir, err := ioutil.TempDir("./", "tmp")
	if err != nil {
		t.Fatalf("ioutil.TempDir %v", err)
	}
	defer os.RemoveAll(dir)

	fileName := dir + "/" + "tmp.wav"

	Convey("New", t, func() {
		_, err := New(fileName, 32000, 1)
		So(err, ShouldBeNil)
	})
}

func CasePCMToWav(t *testing.T) {
	Convey("PCMToWave", t, func() {
		pcm := make([]byte, 88)
		_, err := PCMToWave(32000, 1, pcm)
		So(err, ShouldBeNil)
	})
}

func CasenewWriter(t *testing.T) {
	err := os.Mkdir("./tmp", 0777)
	if err != nil {
		t.Fatalf("Mkdir %v", err)
	}
	dir := "./tmp"
	defer os.RemoveAll(dir)

	tests := []struct {
		fileName     string
		sampleRate   int
		channelCount int
		wanterr      error
	}{
		{
			fileName:     dir + "/" + "test.wav",
			channelCount: 1,
			sampleRate:   32000,
			wanterr:      nil,
		},
	}

	for _, tt := range tests {
		Convey("NewWriter", t, func() {
			writer, err := New(tt.fileName, tt.sampleRate, tt.channelCount)
			if err != tt.wanterr {
				t.Fatalf("NewWriter %v", err)
			}
			Convey("WriteSample16", func() {
				samples := []int16{0x00, 0x00, 0x01, 0x00}
				_, err = writer.WriteSample16(samples)
				if err != tt.wanterr {
					t.Fatalf("WriteSample16 %v", err)
				}
			})
			Convey("Close", func() {
				data := []byte{0x00, 0x00, 0x00, 0x00}
				_, err := writer.Write(data)
				if err != tt.wanterr {
					t.Fatalf("Write %v", err)
				}
				err = writer.Close()
				if err != tt.wanterr {
					t.Fatalf("Close %v", err)
				}
			})
		})
	}
}

func TestWavWriter(t *testing.T) {
	CaseNew(t)
	CasePCMToWav(t)
	CasenewWriter(t)
}
