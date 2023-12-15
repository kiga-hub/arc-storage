package util

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/kiga-hub/arc/logging"
	"golang.org/x/tools/godoc/util"
)

// GzipFile -
func GzipFile(subdirectory, filepath, filename string, header, data []byte) error {
	err := os.MkdirAll(subdirectory, os.ModePerm)
	if err != nil {
		return fmt.Errorf("os.MkdirAll %s, %v", subdirectory, err)
	}
	fw, err := os.Create(filepath) // 创建gzip包文件，返回*io.Writer
	if err != nil {
		return err
	}
	defer fw.Close()
	// 实例化心得gzip.Writer
	gw := gzip.NewWriter(fw)
	defer gw.Close()

	// 创建gzip.Header
	gw.Header.Name = filename
	// 写入数据到zip包
	_, err = gw.Write(header)
	if err != nil {
		return err
	}
	_, err = gw.Write(data)
	if err != nil {
		return err
	}
	return nil
}

// UnGzipFile -
func UnGzipFile(filepath, filename string) error {
	// 打开gzip文件
	fr, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer fr.Close()

	// // get file stat  fi.Name()
	// fi, err := fr.Stat()
	// if err != nil {
	// 	return err
	// }

	// 创建gzip.Reader
	gr, err := gzip.NewReader(fr)
	if err != nil {
		return err
	}
	defer gr.Close()

	output, err := ioutil.ReadAll(gr)
	if err != nil {
		return err
	}

	// 将包中的文件数据写入
	fw, err := os.Create(filename)
	if err != nil {
		return err
	}
	_, err = fw.Write(output)
	if err != nil {
		return err
	}
	defer fw.Close()
	return nil
}

// GzipData return bytes
func GzipData(input []byte, logger logging.ILogger) ([]byte, error) {
	buf := new(bytes.Buffer)
	w, _ := gzip.NewWriterLevel(buf, flate.BestSpeed)
	if _, err := w.Write(input); err != nil {
		logger.Debug("error compressing data:", err)
		return nil, err
	}
	if err := w.Close(); err != nil {
		logger.Debug("error closing compressed data:", err)
		return nil, err
	}
	return buf.Bytes(), nil
}

// UnGzipData return byte
func UnGzipData(input []byte, logger logging.ILogger) ([]byte, error) {
	buf := bytes.NewBuffer(input)
	r, err := gzip.NewReader(buf)
	if err != nil {
		logger.Debug("NewReader:", err)
	}
	defer r.Close()

	output, err := ioutil.ReadAll(r)
	if err != nil {
		logger.Debug("error uncompressing data:", err)
	}
	return output, err
}

// IsGzippable -
/*
* Default more not to gzip since gzip can be done on client side.
 */
func IsGzippable(ext, mtype string, data []byte) bool {

	shouldBeZipped, iAmSure := IsGzippableFileType(ext, mtype)
	if iAmSure {
		return shouldBeZipped
	}

	isMostlyText := util.IsText(data)

	return isMostlyText
}

// IsGzippableFileType return shouldbezipped & iamsure
/*
* Default more not to gzip since gzip can be done on client side.
 */
func IsGzippableFileType(ext, mtype string) (shouldBeZipped, iAmSure bool) {

	// text
	if strings.HasPrefix(mtype, "text/") {
		return true, true
	}

	// images
	switch ext {
	case ".svg", ".bmp", ".wav":
		return true, true
	}
	if strings.HasPrefix(mtype, "image/") {
		return false, true
	}

	// by file name extension
	switch ext {
	case ".zip", ".rar", ".gz", ".bz2", ".xz":
		return false, true
	case ".pdf", ".txt", ".html", ".htm", ".css", ".js", ".json":
		return true, true
	case ".php", ".java", ".go", ".rb", ".c", ".cpp", ".h", ".hpp":
		return true, true
	case ".png", ".jpg", ".jpeg":
		return false, true
	}

	// by mime type
	if strings.HasPrefix(mtype, "application/") {
		if strings.HasSuffix(mtype, "xml") {
			return true, true
		}
		if strings.HasSuffix(mtype, "script") {
			return true, true
		}

	}

	if strings.HasPrefix(mtype, "arc/") {
		switch strings.TrimPrefix(mtype, "arc/") {
		case "wave", "wav", "x-wav", "x-pn-wav":
			return true, true
		}
	}

	return false, false
}
