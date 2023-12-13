package summaryfiles

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"time"

	"github.com/spf13/cobra"
)

// Cmd -
var Cmd = &cobra.Command{
	Use:   "summaryfiles",
	Short: "run summaryfiles",
	Run: func(cmd *cobra.Command, args []string) {
		run()
	},
}

type params struct {
	dir string
}

var param params

func init() {
	Cmd.Flags().StringVarP(&param.dir, "dir", "d", "", "查找目录（required）")
	if err := Cmd.MarkFlagRequired("dir"); err != nil {
		panic(err)
	}
}

//数据类型
type filesInfo struct {
	filesNum  int
	filesSize int64
}
type sensorDataType struct {
	sensorID string
	info     filesInfo
	list     map[string]filesInfo
}

func run() {

	dirs, err := ioutil.ReadDir(param.dir)
	if err != nil {
		panic(err)
	}

	//创建文件
	file, err := os.Create(fmt.Sprintf("data-%s.txt", time.Now().UTC().Format("20060102150405")))
	if file != nil {
		defer file.Close()
	}
	if err != nil {
		panic(err)
	}

	fileCh := make(chan *sensorDataType, 2)
	ch := make(chan struct{})

	go func() {
		defer func() {
			ch <- struct{}{}
		}()
		writeFile(file, fileCh)
	}()

	var wg sync.WaitGroup
	for _, v := range dirs {
		if !v.IsDir() || v.Name() == "." || v.Name() == ".." {
			continue
		}

		//每个传感器目录并行处理
		wg.Add(1)
		go func(sensorIdDir string) {
			defer func() {
				wg.Done()
			}()
			sensorData(sensorIdDir, fileCh)
		}(v.Name())
	}
	wg.Wait()

	close(fileCh)

	<-ch
}

func writeFile(f *os.File, fileCh <-chan *sensorDataType) {
	for data := range fileCh {
		var b bytes.Buffer
		//data结构体处理成[]byte
		b.WriteString(fmt.Sprintf("%s\t%d\t%.2fMB\n", data.sensorID, data.info.filesNum, float64(data.info.filesSize)/float64(1024*1024)))
		for k, v := range data.list {
			b.WriteString(fmt.Sprintf("%s\t%d\t%.2fMB\n", k, v.filesNum, float64(v.filesSize)/float64(1024*1024)))
		}
		b.WriteString("----------------------------------------\n")
		if _, err := f.Write(b.Bytes()); err != nil {
			panic(err)
		}
	}
}

func sensorData(dirSensorID string, fileCh chan<- *sensorDataType) {

	data := &sensorDataType{
		sensorID: dirSensorID,
		info: filesInfo{
			filesNum:  0,
			filesSize: int64(0),
		},
		list: make(map[string]filesInfo),
	}

	defer func() {
		fileCh <- data
	}()

	//查找都有哪些日期的目录
	subDirs, err := ioutil.ReadDir(param.dir + "/" + dirSensorID)
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, vv := range subDirs {
		if !vv.IsDir() || vv.Name() == "." || vv.Name() == ".." {
			continue
		}

		//日期数据收集
		dayInfo := filesInfo{
			filesNum:  0,
			filesSize: 0,
		}

		dayInfo.filesNum, dayInfo.filesSize = rangeFiles(param.dir + "/" + dirSensorID + "/" + vv.Name())

		data.list[vv.Name()] = dayInfo

		data.info.filesNum += dayInfo.filesNum
		data.info.filesSize += dayInfo.filesSize
	}

}

func rangeFiles(dir string) (int, int64) {

	fileNum := 0
	fileSize := int64(0)

	list, err := ioutil.ReadDir(dir)
	if err != nil {
		fmt.Println(err)
		return 0, 0
	}

	for _, v := range list {
		if v.Name() == "." || v.Name() == ".." {
			continue
		}

		num := 0
		size := int64(0)

		if v.IsDir() {
			num, size = rangeFiles(dir + "/" + v.Name())
		} else {
			num = 1
			size = v.Size()
		}

		fileNum += num
		fileSize += size
	}

	return fileNum, fileSize
}

// shell命令实现方式，目前不稳定
//func rangeFilesByCmd(dir string) (int, int64) {
//	var err error
//
//	fileNumByte, err := exec.Command("/bin/bash", "-c", `ls -lR `+dir+` | grep \".wav\" | wc -l`).Output()
//	if err != nil {
//		fmt.Println(err)
//		return 0, 0
//	}
//
//	fileSizeByte, err := exec.Command("/bin/bash", "-c", `find `+dir+` -name "*.wav" -exec ls -l {} \; | cut -d " " -f5 | awk '{sum+=$1}END{print sum}'`).Output()
//	if err != nil {
//		fmt.Println(err)
//		return 0, 0
//	}
//
//	fileNum, err := strconv.Atoi(strings.Trim(strings.TrimSpace(string(fileNumByte)), "\n"))
//	if err != nil {
//		fmt.Println(err)
//		return 0, 0
//	}
//
//	fileSize, err := strconv.Atoi(strings.Trim(strings.TrimSpace(string(fileSizeByte)), "\n"))
//	if err != nil {
//		fmt.Println(err)
//		return 0, 0
//	}
//
//	return fileNum, int64(fileSize)
//}
