package util

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func CaselittleEndianIntToHex(t *testing.T) {
	Convey("littleEndianIntToHex", t, func() {
		Convey("case 2", func() {
			b := littleEndianIntToHex(3, 2)
			So(BytesSliceEqual(b, []byte{3, 0}), ShouldBeTrue)
		})
		Convey("case 4", func() {
			b := littleEndianIntToHex(3, 4)
			So(BytesSliceEqual(b, []byte{3, 0, 0, 0}), ShouldBeTrue)
		})
	})

}

func CaseapplyString(t *testing.T) {
	Convey("applyString", t, func() {
		dst := make([]byte, 1)
		s := "0"
		numberOfBytes := 1
		applyString(dst, s, numberOfBytes)
		So(BytesSliceEqual(dst, []byte("0")[:1]), ShouldBeTrue)
	})
}

func CaseLittleEndianInteger(t *testing.T) {
	Convey("applyLittleEndianInteger", t, func() {
		dst := make([]byte, 1)
		i := 1
		numberOfBytes := 1
		applyLittleEndianInteger(dst, i, numberOfBytes)
		So(BytesSliceEqual(dst, []byte{0x00}), ShouldBeTrue)
	})
}

func CaseConvertPCMToWav(t *testing.T) {
	pcm := make([]byte, 44)
	Convey("ConvertPCMToWav", t, func() {
		_, err := ConvertPCMToWav(pcm, 1, 32000, 16)
		So(err, ShouldBeNil)
	})
	Convey("ConvertPCMToWav", t, func() {
		_, err := ConvertPCMToWav(pcm, 1, 32000, 0)
		So(err, ShouldBeNil)
	})
}

func CaseConvertPCMToWavHeader(t *testing.T) {
	Convey("ConvertPCMToWavHeader", t, func() {
		_, err := ConvertPCMToWavHeader(44, 1, 32000, 16)
		So(err, ShouldBeNil)
	})
	Convey("ConvertPCMToWavHeader", t, func() {
		_, err := ConvertPCMToWavHeader(44, 1, 32000, 0)
		So(err, ShouldBeNil)
	})
}
func TestPCM2WAV(t *testing.T) {
	CaselittleEndianIntToHex(t)
	CaseapplyString(t)
	CaseLittleEndianInteger(t)
	CaseConvertPCMToWav(t)
	CaseConvertPCMToWavHeader(t)
}
