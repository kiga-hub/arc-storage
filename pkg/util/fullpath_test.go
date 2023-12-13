package util

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func CaseNewFullPath(t *testing.T) {
	dir, err := ioutil.TempDir("./", "tmp")
	if err != nil {
		t.Fatalf("ioutil.TempDir %v", err)
	}
	defer os.RemoveAll(dir)

	file, err := ioutil.TempFile(dir, "tmp.wav")
	if err != nil {
		t.Fatalf("ioutil.TempFile %v", err)
	}
	defer file.Close()

	Convey("NewFullPath", t, func() {
		path := NewFullPath(dir, path.Base(file.Name()))
		So(path, ShouldEqual, file.Name())
	})

	Convey("DirAndName", t, func() {
		Convey("DirAndName", func() {
			fullpath := FullPath(file.Name())
			filedir, name := fullpath.DirAndName()
			So(filedir, ShouldEqual, dir)
			So(name, ShouldEqual, path.Base(file.Name()))
		})
		Convey("/", func() {
			fullpath := FullPath("/")
			filedir, name := fullpath.DirAndName()
			So(filedir, ShouldEqual, "/")
			So(name, ShouldEqual, "")
		})
		Convey("", func() {
			fullpath := FullPath("")
			filedir, name := fullpath.DirAndName()
			So(filedir, ShouldEqual, "/")
			So(name, ShouldEqual, "")
		})
	})

	Convey("Name", t, func() {
		fullpath := FullPath(file.Name())
		name := fullpath.Name()
		So(name, ShouldEqual, path.Base(file.Name()))
	})

	Convey("Child", t, func() {
		Convey("1", func() {
			fullpath := FullPath(dir)
			name := fullpath.Child(path.Base(file.Name()))
			So(name, ShouldEqual, file.Name())
		})
		Convey("2", func() {
			fullpath := FullPath(dir)
			name := fullpath.Child("")
			So(name, ShouldEqual, dir+"/")
		})
	})
	Convey("AsInode", t, func() {
		err := os.Mkdir("./tmptest", 0777)
		So(err, ShouldBeNil)
		tmpdir := "./tmptest"
		defer os.RemoveAll(tmpdir)

		fullpath := FullPath(tmpdir)
		u := fullpath.AsInode()
		So(u, ShouldEqual, uint64(146199663747992255))
	})

	Convey("Split", t, func() {
		Convey("/", func() {
			fullpath := FullPath("/")
			strslice := fullpath.Split()
			So(StringSliceEqual(strslice, []string{}), ShouldBeTrue)
		})
		Convey("[]string{}", func() {
			fullpath := FullPath("./tmp")
			strslice := fullpath.Split()
			So(StringSliceEqual(strslice, []string{"", "tmp"}), ShouldBeTrue)
		})
	})

}

func CaseJoin(t *testing.T) {
	Convey("Join", t, func() {
		Convey("Join", func() {
			str := Join("1", "2", "3")
			So(str, ShouldEqual, "1/2/3")
		})
		Convey("Joinpath", func() {
			str := JoinPath("1", "2", "3")
			So(str, ShouldEqual, FullPath("1/2/3"))
		})
	})

}
func TestFullPath(t *testing.T) {
	CaseNewFullPath(t)
	CaseJoin(t)
}
