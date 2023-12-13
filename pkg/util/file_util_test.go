package util

import (
	"io/ioutil"
	"os"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/zap"
)

func CaseTestFolderWritable(t *testing.T) {
	logger, err := zap.NewProduction()
	if err != nil {
		t.Fatalf("NewProduction %v", err)
	}
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

	Convey("TestFolderWritable", t, func() {
		Convey("1", func() {
			err := TestFolderWritable(dir, logger.Sugar())
			So(err, ShouldBeNil)
		})
		Convey("2", func() {
			err := TestFolderWritable(file.Name(), logger.Sugar())
			So(err, ShouldNotBeNil)
		})
	})

	Convey("GetFileSize", t, func() {
		_, err := GetFileSize(file)
		So(err, ShouldBeNil)
	})

	Convey("FileExists", t, func() {
		b := CheckFileExists(file.Name())
		So(b, ShouldBeTrue)
	})

	Convey("pathExists", t, func() {
		Convey("true", func() {
			b, err := PathExists(file.Name())
			So(b, ShouldBeTrue)
			So(err, ShouldBeNil)
		})
		Convey("false", func() {
			b, err := PathExists(file.Name() + "1")
			So(b, ShouldBeFalse)
			So(err, ShouldBeNil)
		})
	})

	Convey("CheckFile", t, func() {
		exists, canRead, canWrite, _, _ := CheckFile(file.Name())
		So(exists, ShouldBeTrue)
		So(canRead, ShouldBeTrue)
		So(canWrite, ShouldBeTrue)
	})
}

func CaseRemoveFolders(t *testing.T) {
	Convey("RemoveFolders", t, func() {
		err := os.Mkdir("./tmp.ldb", 0777)
		So(err, ShouldBeNil)

		dir := "./tmp.ldb"
		err = RemoveFolders(dir)
		So(err, ShouldBeNil)
	})
}

func CaseGetFileList(t *testing.T) {
	err := os.Mkdir("./tmp", 0777)
	if err != nil {
		t.Fatalf("Mkdir err %v", err)
	}
	dir := "./tmp"
	defer os.RemoveAll("./tmp")
	Convey("CaseGetFileList", t, func() {
		Convey("1", func() {
			err = os.Mkdir("./tmp/tmp", 0777)
			if err != nil {
				t.Fatalf("Mkdir err %v", err)
			}

			_, err = os.Create(dir + "/tmp/1.wav")
			if err != nil {
				t.Fatalf("os.Create err %v", err)
			}

			_, err = os.Create(dir + "/tmp/2.wav")
			if err != nil {
				t.Fatalf("os.Create err %v", err)
			}
			files := make(map[string][]string)
			key := ""
			GetFileList(dir, files, key)
			So(key, ShouldEqual, "")
		})
		Convey("2", func() {
			_, err := os.Create(dir + "/20210322072559917_20210322072600012_32000_010000_01_000000.wav")
			if err != nil {
				t.Fatalf("os.Create err %v", err)
			}

			_, err = os.Create(dir + "/20210322072559917_20210322072600012_32000_010000_01_000001.wav")
			if err != nil {
				t.Fatalf("os.Create err %v", err)
			}
			files := make(map[string][]string)
			key := ""
			GetFileList(dir, files, key)
			So(key, ShouldEqual, "")
		})
	})
}

func CaseGetFileListName(t *testing.T) {
	// err := os.Mkdir("./tmp", 0777)
	// if err != nil {
	// 	t.Fatalf("Mkdir err %v", err)
	// }
	// dir := "./tmp"
	// defer os.RemoveAll("./tmp")
	// Convey("CaseGetFileListName", t, func() {
	// 	Convey("1", func() {
	// 		err = os.Mkdir("./tmp/tmp", 0777)
	// 		if err != nil {
	// 			t.Fatalf("Mkdir err %v", err)
	// 		}

	// 		_, err = os.Create(dir + "/tmp/20210322072559917_20210322072600012_32000_010000_01_000000.wav")
	// 		if err != nil {
	// 			t.Fatalf("os.Create err %v", err)
	// 		}

	// 		_, err = os.Create(dir + "/tmp/20210322072559917_20210322072600012_32000_010000_01_000001.wav")
	// 		if err != nil {
	// 			t.Fatalf("os.Create err %v", err)
	// 		}
	// 		key := ""
	// 		files := make(map[string]string)
	// 		err := GetFileListName(dir, files, key)
	// 		So(err, ShouldBeNil)
	// 		So(key, ShouldEqual, "")
	// 	})
	// 	Convey("2", func() {
	// 		_, err := os.Create(dir + "/20210322072559917_20210322072600012_32000_010000_01_000000.wav")
	// 		if err != nil {
	// 			t.Fatalf("os.Create err %v", err)
	// 		}

	// 		_, err = os.Create(dir + "/20210322072559917_20210322072600012_32000_010000_01_000001.wav")
	// 		if err != nil {
	// 			t.Fatalf("os.Create err %v", err)
	// 		}
	// 		files := make(map[string]string)
	// 		key := ""
	// 		err = GetFileListName(dir, files, key)
	// 		So(err, ShouldBeNil)
	// 		So(key, ShouldEqual, "")
	// 	})
	// })
}

func TestFileUtil(t *testing.T) {
	CaseTestFolderWritable(t)
	CaseRemoveFolders(t)
	CaseGetFileList(t)
	CaseGetFileListName(t)
}
