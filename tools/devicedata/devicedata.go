package devicedata

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"github.com/kiga-hub/arc-storage/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
)

// Cmd -
var Cmd = &cobra.Command{
	Use:   "devicedata",
	Short: "run devicedata",
	Run: func(cmd *cobra.Command, args []string) {
		run()
	},
}

// ParamMinuteFormat - 时间参数格式
const ParamMinuteFormat = "20060102150405"

// AudioType - 数据类型枚举：音频
const AudioType = "audio"

// VibrateType - 数据类型枚举：震动
const VibrateType = "vibrate"

// SaveFileBaseDir - 结果数据保存跟目录
const SaveFileBaseDir = "./data"

var errNotFoundData = errors.New("not found data")

type params struct {
	host  string
	dir   string
	start string
	end   string
}

var param params

func init() {
	Cmd.Flags().StringVarP(&param.host, "addr", "a", "", "服务器IP（required）")
	Cmd.Flags().StringVarP(&param.dir, "dir", "d", "", "源数据目录（required）")
	Cmd.Flags().StringVarP(&param.start, "start", "s", "", "要查询的开始时间，格式: yyyymmddhhiiss（required）")
	Cmd.Flags().StringVarP(&param.end, "end", "e", "", "要查询的结束时间，格式: yyyymmddhhiiss（required）")

	if err := Cmd.MarkFlagRequired("addr"); err != nil {
		panic(err)
	}
	if err := Cmd.MarkFlagRequired("dir"); err != nil {
		panic(err)
	}
	if err := Cmd.MarkFlagRequired("start"); err != nil {
		panic(err)
	}
	if err := Cmd.MarkFlagRequired("end"); err != nil {
		panic(err)
	}
}

func run() {

	//时间转换
	start, err := time.ParseInLocation(ParamMinuteFormat, param.start, time.UTC)
	if err != nil {
		panic(err)
	}
	end, err := time.ParseInLocation(ParamMinuteFormat, param.end, time.UTC)
	if err != nil {
		panic(err)
	}
	if end.Before(start) || end.Equal(start) {
		panic(errors.New("start can't Later and equal than end"))
	}
	if end.Sub(start).Minutes() > 60 {
		panic(errors.New("start to end max over 60 minutes"))
	}

	//数据目录
	if err := initDir(getDataDir(AudioType)); err != nil {
		panic(err)
	}
	if err := initDir(getDataDir(VibrateType)); err != nil {
		panic(err)
	}

	//获取设备id
	deviceIds, err := getDeviceIds()
	if err != nil {
		panic(fmt.Sprintf("%+v", err))
	}
	log.Println("开始获取以下设备号的数据")
	log.Println(deviceIds)

	//开始拿数据
	var wg sync.WaitGroup
	wg.Add(len(deviceIds) * 2)
	for _, id := range deviceIds {
		go func(id string) {
			defer wg.Done()
			if err := makeData(id, AudioType, start, end); err != nil {
				if errors.Is(err, errNotFoundData) {
					log.Printf("%s: %s", id, err.Error())
				} else {
					log.Printf("%s: %+v", id, err)
				}
			}
		}(id)
		go func(id string) {
			defer wg.Done()
			if err := makeData(id, VibrateType, start, end); err != nil {
				if errors.Is(err, errNotFoundData) {
					log.Printf("%s: %s", id, err.Error())
				} else {
					log.Printf("%s: %+v", id, err)
				}
			}
		}(id)
	}
	wg.Wait()

	log.Println("ok!")
	os.Exit(0)
}

func getDeviceIds() ([]string, error) {

	params := url.Values{}
	params.Set("page", "1")
	params.Set("limit", "10000")

	var u url.URL
	u.Scheme = "http"
	u.Host = param.host
	u.Path = "/api/device/v1/collector/pagelist"
	u.RawQuery = params.Encode()
	addr := u.String()

	resp, err := httpGet(addr, nil)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return nil, err
	}

	//处理状态码
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(fmt.Sprintf("http status code %d", resp.StatusCode))
	}

	//处理返回结果
	byteData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("resp.Body:%v", resp.Body))
	}

	type collectorList struct {
		Serial string `json:"Serial"`
	}
	type ListCollectorResponse struct {
		Code int              `json:"code"`
		Msg  string           `json:"msg"`
		Data []*collectorList `json:"data"`
	}

	var data ListCollectorResponse
	if err := json.Unmarshal(byteData, &data); err != nil {
		return nil, errors.WithStack(err)
	}

	if data.Code != 200 {
		return nil, errors.New(data.Msg)
	}

	//提取设备id数据
	var ids []string
	for _, v := range data.Data {
		ids = append(ids, v.Serial)
	}

	return ids, nil
}

func makeData(id, fileType string, start, end time.Time) error {

	byteData, err := getWavData(id, fileType, start, end)
	if err != nil {
		return err
	}

	//创建文件
	fileName := getFileName(getDataDir(fileType), id)
	file, err := os.Create(fileName)
	if file != nil {
		defer file.Close()
	}
	if err != nil {
		return errors.Wrap(err, id)
	}

	//写入bytes
	_, err = file.Write(byteData)
	if err != nil {
		return errors.Wrap(err, id)
	}

	return nil
}

func getDataDir(fileType string) string {
	return SaveFileBaseDir + "/" + param.start + "-" + param.end + "/" + fileType
}

// 获取音频、震动文件内容
func getWavData(sensorID, fileType string, t1, t2 time.Time) ([]byte, error) {

	channel := 1
	channelmulbits := 2
	if fileType == "vibrate" {
		channel = 3
		channelmulbits = 6
	}

	// 获取以天为单位的时间范围
	daysdiffer, count, err := util.GetDaysDiffer(t1.UTC().Format("2006-01-02 15:04:05"), t2.UTC().Format("2006-01-02 15:04:05"))
	if err != nil {
		return nil, err
	}
	if len(daysdiffer) < 1 {
		return nil, errNotFoundData
	}

	// 获取大文件列表
	bigfilelists := make(map[string]string, count) // map[createtime]fullpath
	for _, day := range daysdiffer {
		path := param.dir + "/" + sensorID + "/" + day + "/" + fileType
		err := util.GetBigFileLists(path, bigfilelists, "")
		if err != nil {
			continue
		}
	}
	if len(bigfilelists) < 1 {
		return nil, errNotFoundData
	}

	var timestamplist []string
	for key := range bigfilelists {
		timestamplist = append(timestamplist, key)
	}
	sort.Strings(timestamplist)

	t1str := t1.UTC().Format("20060102150405") + "000000"
	t2str := t2.UTC().Format("20060102150405") + "000000"

	var starttimestamps []string
	for _, creattime := range timestamplist {
		_, end, _, _, _, err := util.GetTimeRangeFromFileName(bigfilelists[creattime])
		if err != nil {
			continue
		}
		// 开始时间大于文件保存结束时间，忽略
		if t1str >= util.TimeStringReplace(end) {
			continue
		}
		// 结束时间小于文件创建时间，退出
		if t2str <= creattime {
			break
		}
		starttimestamps = append(starttimestamps, creattime)
	}
	if len(starttimestamps) < 1 {
		return nil, errNotFoundData
	}
	sort.Strings(starttimestamps)

	type filepathlistV struct {
		bigFile string
		start   time.Time
	}
	var filepathlist []filepathlistV
	sampleratestr := ""
	// 遍历当前文件夹下得目录
	for _, v := range starttimestamps {
		bigFile := bigfilelists[v]

		_, err = os.Stat(bigFile)
		if os.IsNotExist(err) {
			continue
		}

		fileName := bigFile[strings.LastIndex(bigFile, "/")+1:]
		startdate, _, _, _, sam, err := util.GetTimeRangeFromFileName(fileName)
		if err != nil {
			continue
		}

		// 采样率或者有中断数据，返回
		if sampleratestr == "" {
			sampleratestr = sam
		}
		if sampleratestr != sam {
			return nil, errors.New("sample rate is changed or include 'T' type file")
		}

		filepathlist = append(filepathlist, filepathlistV{
			bigFile: bigFile,
			start:   startdate,
		})
	}
	if len(filepathlist) < 1 {
		return nil, errNotFoundData
	}

	// 获取数据文件大小
	finalsize := cast.ToInt(t2.Sub(t1).Seconds()) * cast.ToInt(sampleratestr) * channelmulbits

	var Response bytes.Buffer
	var buffer bytes.Buffer

	for i, tmp := range filepathlist {
		v := tmp.bigFile
		startdate := tmp.start

		// 只有一个文件符合条件，
		if len(filepathlist) == 1 {
			// 从开始时间进行切分
			cutduration := t1.Sub(startdate).Seconds()
			// 切分时间长度
			cutindex := cutduration * float64(cast.ToInt(sampleratestr))
			// 查找数据, 写入Bufffer并返回
			data, err := util.SeekBigFileHeader(v, cast.ToInt64(cutindex)*cast.ToInt64(channelmulbits)+44, cast.ToInt64(finalsize))
			if err != nil {
				continue
			}
			wavheader, err := util.ConvertPCMToWavHeader(
				finalsize,
				channel,
				cast.ToInt(sampleratestr),
				16,
			)
			if err != nil {
				return nil, err
			}
			Response.Write(wavheader)
			Response.Write(data)

			return Response.Bytes(), nil
		}

		// 存在多个文件，第一个文件进行切分，后续进行拼接
		if i == 0 && len(filepathlist) != 1 {
			// 从开始时间进行切分
			cutduration := t1.Sub(startdate).Seconds()
			// 切分时间长度
			cutindex := cutduration * float64(cast.ToInt(sampleratestr))

			fileInfo, err := os.Stat(v)
			if err != nil {
				break
			}
			datasize := fileInfo.Size() - 44
			data, err := util.SeekBigFileHeader(v, cast.ToInt64(cutindex)*cast.ToInt64(channelmulbits)+44, datasize)
			if err != nil {
				break
			}
			buffer.Write(data)
		}

		// 中间部分文件，对数据不做处理，全部读取并直接拼接
		if i != 0 && len(filepathlist) != 1 && i != len(filepathlist)-1 {
			fileInfo, err := os.Stat(v)
			if err != nil {
				return nil, err
			}
			datasize := fileInfo.Size() - 44
			data, err := util.SeekBigFileHeader(v, 44, datasize)
			if err != nil {
				break
			}
			buffer.Write(data)
		}

		// 对最后一个文件进行处理，
		if i == len(filepathlist)-1 && len(filepathlist) != 1 {
			// 从开始时间进行切分
			cutduration := t2.Sub(startdate).Seconds()
			// 切分时间长度
			cutindex := cutduration * float64(cast.ToInt(sampleratestr))

			data, err := util.SeekBigFileHeader(v, cast.ToInt64(cutindex)*cast.ToInt64(channelmulbits)+44, cast.ToInt64(finalsize-int(buffer.Len())))
			if err != nil {
				break
			}
			buffer.Write(data)

			wavheader, err := util.ConvertPCMToWavHeader(
				buffer.Len(),
				channel,
				cast.ToInt(sampleratestr),
				16,
			)
			if err != nil {
				return nil, err
			}
			Response.Write(wavheader)
			Response.Write(buffer.Bytes())

			return Response.Bytes(), nil
		}
	}

	return nil, errNotFoundData
}

func initDir(dir string) error {
	b, err := isDirExist(dir)
	if err != nil {
		return errors.WithStack(err)
	}
	if !b {
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

func isDirExist(dir string) (bool, error) {
	_, err := os.Stat(dir)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func getFileName(dir, fileName string) string {
	return dir + "/" + fileName + ".wav"
}

func httpGet(url string, headers map[string]string) (*http.Response, error) {
	var req *http.Request
	var err error

	req, err = http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, errors.Wrap(err, url)
	}

	//增加header选项
	for k, v := range headers {
		req.Header.Add(k, v)
	}

	//发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("req:%v", req))
	}

	return resp, nil
}
