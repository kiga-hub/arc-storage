package util

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"

	"github.com/kiga-hub/arc/logging"
)

// TestFolderWritable stat of folder
func TestFolderWritable(folder string, logger logging.ILogger) (err error) {
	fileInfo, err := os.Stat(folder)
	if err != nil {
		return err
	}
	if !fileInfo.IsDir() {
		return errors.New("not a valid folder")
	}
	perm := fileInfo.Mode().Perm()
	logger.Debug("Folder", folder, "Permission:", perm)
	if 0200&perm != 0 {
		return nil
	}
	return errors.New("not writable")
}

// GetFileSize get file size
func GetFileSize(file *os.File) (size int64, err error) {
	var fi os.FileInfo
	if fi, err = file.Stat(); err == nil {
		size = fi.Size()
	}
	return
}

// RemoveFolders -
func RemoveFolders(dir string) error {
	exist, err := PathExists(dir)
	if err != nil {
		return err
	}
	if exist && strings.Contains(dir, ".ldb") {
		err := os.RemoveAll(dir)
		if err != nil {
			return err
		}
	}
	return nil
}

// CheckFileExists  check file is exist
func CheckFileExists(filename string) bool {
	var exist = true
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		exist = false
	}
	return exist
}

// PathExists -
func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// CheckFile checkfile
func CheckFile(filename string) (exists, canRead, canWrite bool, modTime time.Time, fileSize int64) {
	exists = true
	fi, err := os.Stat(filename)
	if os.IsNotExist(err) {
		exists = false
		return
	}
	if fi.Mode()&0400 != 0 {
		canRead = true
	}
	if fi.Mode()&0200 != 0 {
		canWrite = true
	}
	modTime = fi.ModTime()
	fileSize = fi.Size()
	return
}

// GetFileList -
func GetFileList(path string, arcfiles map[string][]string, key string) {
	fs, _ := ioutil.ReadDir(path)
	for _, file := range fs {
		if file.IsDir() {
			key = path + file.Name()
			GetFileList(path+file.Name(), arcfiles, key)
		} else {
			arcfiles[string(key)] = append(arcfiles[string(key)], path+"/"+file.Name())
		}
	}
}

// GetBigFileLists -
func GetBigFileLists(path string, arcfiles map[string]string, key string) error {
	fs, err := ioutil.ReadDir(path)
	if err != nil {
		return err
	}
	for _, file := range fs {
		if strings.Contains(file.Name(), ".wav") {
			start, _, _, _, _, err := GetTimeRangeFromFileName(file.Name())
			if err != nil {
				return err
			}
			startstr := TimeStringReplace(start)
			arcfiles[startstr] = path + "/" + file.Name()
		}
	}
	return nil
}

// SeekBigFileHeader -
func SeekBigFileHeader(filename string, start, size int64) ([]byte, error) {
	b := make([]byte, size)
	n := 0
	var err error
	f, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0644) //os.O_TRUNC|
	if err != nil {
		return []byte{}, fmt.Errorf("file create failed. err: " + err.Error())
	}
	defer f.Close()

	_, err = f.Seek(start, io.SeekStart)
	if err != nil {
		return []byte{}, fmt.Errorf("f.Seek: " + err.Error())
	}

	n, err = f.Read(b)
	if err != nil {
		return []byte{}, fmt.Errorf("f.Read: " + err.Error())
	}

	return b[:n], nil
}

// GetFileListName -
func GetFileListName(path string, arcfiles map[string]string, key string) error {
	fs, _ := ioutil.ReadDir(path)
	for _, file := range fs {
		if file.IsDir() {
			//	key += file.Name()[:2]
			err := GetFileListName(path+file.Name(), arcfiles, key) // key+file.Name()[:17]
			if err != nil {
				return err
			}
		} else {
			if len(file.Name()) > 34 { //20211028080000000460
				start, _, _, _, _, err := GetTimeRangeFromFileName(file.Name())
				if err != nil {
					return err
				}
				startstr := TimeStringReplace(start)
				arcfiles[startstr] = path + "/" + file.Name()

			}
		}
	}
	return nil
}

// GetFileListNamebyCreateTime -
func GetFileListNamebyCreateTime(path string, key string) (string, error) {
	fs, _ := ioutil.ReadDir(path)
	for _, file := range fs {
		if !file.IsDir() {
			start, _, _, _, _, err := GetTimeRangeFromFileName(file.Name())
			if err != nil {
				return "", err
			}
			startstr := TimeStringReplace(start)
			if key == startstr {
				return path + "/" + file.Name(), nil
			}
		}
	}
	return "", nil
}

// GetTimeRangeFromFileName -
func GetTimeRangeFromFileName(filename string) (start, end time.Time, sensorid, status, samplerate string, err error) {
	basename := path.Base(filename)
	strSlice := strings.Split(basename, "_")
	l := len(strSlice)
	if l == 6 {
		if len(strSlice[0]) == 20 && len(strSlice[1]) == 20 {
			startstr := strSlice[0][:14] + "." + strSlice[0][14:]
			startdate, err := time.Parse("20060102150405.999999", startstr)
			if err != nil || startdate.Unix() < 0 {
				return time.Now(), time.Now(), "", "", "", err
			}
			endstr := strSlice[1][:14] + "." + strSlice[1][14:]
			enddate, err := time.Parse("20060102150405.999999", endstr)
			if err != nil || enddate.Unix() < 0 {
				return time.Now(), time.Now(), "", "", "", err
			}
			return startdate, enddate, "", "", strSlice[2], nil
		}
	}
	if l == 9 {
		if len(strSlice[2]) == 20 && len(strSlice[3]) == 20 {
			startstr := strSlice[2][:14] + "." + strSlice[2][14:]
			startdate, err := time.Parse("20060102150405.999999", startstr)
			if err != nil || startdate.Unix() < 0 {
				return time.Now(), time.Now(), "", "", "", err
			}
			endstr := strSlice[3][:14] + "." + strSlice[3][14:]
			enddate, err := time.Parse("20060102150405.999999", endstr)
			if err != nil || enddate.Unix() < 0 {
				return time.Now(), time.Now(), "", "", "", err
			}
			return startdate, enddate, "", "", strSlice[4], nil
		}
	}
	if l == 10 {
		if len(strSlice[2]) == 20 && len(strSlice[3]) == 20 {
			startstr := strSlice[2][:14] + "." + strSlice[2][14:]
			startdate, err := time.Parse("20060102150405.999999", startstr)
			if err != nil || startdate.Unix() < 0 {
				return time.Now(), time.Now(), "", "", "", err
			}
			endstr := strSlice[3][:14] + "." + strSlice[3][14:]
			enddate, err := time.Parse("20060102150405.999999", endstr)
			if err != nil || enddate.Unix() < 0 {
				return time.Now(), time.Now(), "", "", "", err
			}
			return startdate, enddate, strSlice[0], strSlice[4], strSlice[5], nil
		}
	}
	return time.Now(), time.Now(), "", "", "", errors.New("get time range failed")
}

// GetLastCreateFile - get last create file name
func GetLastCreateFile(path string) string {
	fs, _ := ioutil.ReadDir(path)
	var modtime time.Time
	lastfilename := ""
	for _, file := range fs {
		if modtime.Unix() > 0 {
			// 获取最近一次新建文件名
			if modtime.Before(file.ModTime()) {
				modtime = file.ModTime()
				lastfilename = file.Name()
			}
		} else {
			modtime = file.ModTime()
			lastfilename = file.Name()
		}
	}
	// fmt.Println(lastfilename)

	return lastfilename
}
