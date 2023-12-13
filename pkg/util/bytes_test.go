package util

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func CaseBytesToHumanReadable(t *testing.T) {
	Convey("CaseBytesToHumanReadable", t, func() {
		Convey("1", func() {
			str := BytesToHumanReadable(uint64(10))
			So(str, ShouldEqual, "10 B")
		})
		Convey("2", func() {
			str := BytesToHumanReadable(2048)
			So(str, ShouldEqual, "2.00 KiB")
		})
		Convey("3", func() {
			str := BytesToHumanReadable(uint64(1048576))
			So(str, ShouldEqual, "1.00 MiB")
		})
	})
}

func CaseBytesToUint64(t *testing.T) {
	Convey("BytesToUint64", t, func() {
		v := BytesToUint64([]byte{0x00, 0x10})
		So(v, ShouldEqual, uint64(16))
	})
}

func CaseBytesToUint32(t *testing.T) {
	Convey("BytesToUint32", t, func() {
		v := BytesToUint32([]byte{0x00, 0x10})
		So(v, ShouldEqual, uint32(16))
	})
}

func CaseBytesToUint16(t *testing.T) {
	Convey("BytesToUint16", t, func() {
		v := BytesToUint16([]byte{0x00, 0x10})
		So(v, ShouldEqual, uint16(16))
	})
}

func CaseUint64toBytes(t *testing.T) {
	Convey("Uint64ToBytes", t, func() {
		data := make([]byte, 8)
		Uint64toBytes(data, uint64(16))
		So(BytesSliceEqual(data, []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10}), ShouldBeTrue)
	})
}

func CaseUint32toBytes(t *testing.T) {
	Convey("Uint32toBytes", t, func() {
		data := make([]byte, 4)
		Uint32toBytes(data, uint32(16))
		So(BytesSliceEqual(data, []byte{0x00, 0x00, 0x00, 0x10}), ShouldBeTrue)
	})
}

func CaseUint16toBytes(t *testing.T) {
	Convey("Uint16toBytes", t, func() {
		data := make([]byte, 2)
		Uint16toBytes(data, uint16(16))
		So(BytesSliceEqual(data, []byte{0x00, 0x10}), ShouldBeTrue)
	})
}

func CaseUint8toBytes(t *testing.T) {
	Convey("Uint8toBytes", t, func() {
		data := make([]byte, 1)
		Uint8toBytes(data, uint8(16))
		So(BytesSliceEqual(data, []byte{0x10}), ShouldBeTrue)
	})
}

func CaseHashStringToLong(t *testing.T) {
	Convey("HashStringToLong", t, func() {
		Convey("1", func() {
			v, err := HashStringToLong("/root")
			So(v, ShouldEqual, int64(-8612847859700085591))
			So(err, ShouldBeNil)
		})
		Convey("2", func() {
			v, err := HashStringToLong("")
			So(v, ShouldEqual, int64(-3162216497309240828))
			So(err, ShouldBeNil)
		})
	})
}

func CaseHashToInt32(t *testing.T) {
	Convey("HashToInt32", t, func() {
		v, err := HashToInt32([]byte{0x10})
		So(v, ShouldEqual, int32(1798422010))
		So(err, ShouldBeNil)
	})
}

func CaseMd5(t *testing.T) {
	Convey("Md5", t, func() {
		str, err := Md5([]byte{0x10})
		So(str, ShouldEqual, "6B31BDFA7F9BFECE263381FFA91BD6A9")
		So(err, ShouldBeNil)
	})
}

func TestBytes(t *testing.T) {
	CaseBytesToHumanReadable(t)
	CaseBytesToUint64(t)
	CaseBytesToUint32(t)
	CaseBytesToUint16(t)
	CaseUint64toBytes(t)
	CaseUint32toBytes(t)
	CaseUint16toBytes(t)
	CaseUint8toBytes(t)
	CaseHashStringToLong(t)
	CaseHashToInt32(t)
	CaseMd5(t)

}

func BytesSliceEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}

	if (a == nil) != (b == nil) {
		return false
	}

	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

func StringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	if (a == nil) != (b == nil) {
		return false
	}

	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}
