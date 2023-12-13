package util

import (
	"os"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func CaseNewReader(t *testing.T) {
	err := os.Mkdir("./tmp", 0777)
	if err != nil {
		t.Fatalf("Mkdir %v", err)
	}
	dir := "./tmp"
	defer os.RemoveAll(dir)

	wavfileName := dir + "/" + "test.wav"

	Convey("NewWriter", t, func() {
		writer, err := New(wavfileName, 32000, 1)
		if err != nil {
			t.Fatalf("NewWriter %v", err)
		}
		Convey("WriteSample16", func() {
			samples := []int16{0x00, 0x00, 0x00, 0x00}
			_, err = writer.WriteSample16(samples)
			if err != nil {
				t.Fatalf("WriteSample16 %v", err)
			}
		})
		Convey("Close", func() {
			data := []byte{0x00, 0x00, 0x00, 0x00}
			_, err := writer.Write(data)
			if err != nil {
				t.Fatalf("Write %v", err)
			}
			err = writer.Close()
			if err != nil {
				t.Fatalf("Close %v", err)
			}
		})
	})

	Convey("NewReader", t, func() {
		reader, err := NewReader(wavfileName)
		So(err, ShouldBeNil)
		Convey("Class Function", func() {
			_, err = reader.ReadSample()
			So(err, ShouldBeNil)

			_, err = reader.ReadSampleInt()
			So(err, ShouldBeNil)
		})
	})
}

func TestWavReader(t *testing.T) {
	CaseNewReader(t)
}
