package util

import (
	"os"
	"reflect"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/zap"
)

func CaseGzipAndUnGzipFile(t *testing.T) {
	err := os.Mkdir("./tmp", 0777)
	if err != nil {
		t.Fatalf("Mkdir err %v", err)
	}
	subdirectory := "./tmp"
	defer os.RemoveAll(subdirectory)

	filepath := subdirectory + "/" + "tmp.gz"
	dst := subdirectory + "/" + "test.wav"

	Convey("GzipFile", t, func() {
		Convey("GzipFile", func() {
			header := make([]byte, 0)
			data := make([]byte, 0)
			err = GzipFile(subdirectory, filepath, "test.wav", header, data)
			So(err, ShouldBeNil)
		})
		Convey("UnGzipFile", func() {
			err := UnGzipFile(filepath, dst)
			So(err, ShouldBeNil)
		})
	})
}

func CaseGzipDataAndUnGzipData(t *testing.T) {
	logger, err := zap.NewProduction()
	if err != nil {
		t.Fatalf("NewProduction %v", err)
	}
	data := []byte{0x00, 0x01}

	Convey("GzipDataAndUnGzipData", t, func() {
		b, err := GzipData(data, logger.Sugar())
		So(err, ShouldBeNil)

		ub, err := UnGzipData(b, logger.Sugar())
		So(err, ShouldBeNil)
		So(BytesSliceEqual(data, ub), ShouldBeTrue)
	})

}

func CaseIsGzippable(t *testing.T) {
	tests := []struct {
		ext      string
		mtype    string
		wantbool bool
	}{
		{
			ext:      ".svg",
			mtype:    "image/",
			wantbool: true,
		},
		{
			ext:      ".bmp",
			mtype:    "text/",
			wantbool: true,
		},
		{
			ext:      ".zip",
			mtype:    "image/",
			wantbool: false,
		},
		{
			ext:      ".zip",
			mtype:    "",
			wantbool: false,
		},
		{
			ext:      ".pdf",
			mtype:    "",
			wantbool: true,
		},
		{
			ext:      ".php",
			mtype:    "",
			wantbool: true,
		},
		{
			ext:      ".png",
			mtype:    "",
			wantbool: false,
		},
		{
			ext:      "",
			mtype:    "application/test.xml",
			wantbool: true,
		},
		{
			ext:      "",
			mtype:    "application/test.script",
			wantbool: true,
		},
		{
			ext:      "",
			mtype:    "audio/wav",
			wantbool: true,
		},
		{
			ext:      "",
			mtype:    "audio/nil",
			wantbool: true,
		},
	}
	data := []byte{0x00, 0x01}
	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			if got := IsGzippable(tt.ext, tt.mtype, data); !reflect.DeepEqual(got, tt.wantbool) {
				t.Errorf("SiGzippable got:%v want:%v", got, tt.wantbool)
			}
		})
	}

}
func TestCompression(t *testing.T) {
	CaseGzipAndUnGzipFile(t)
	CaseGzipDataAndUnGzipData(t)
	CaseIsGzippable(t)
}
