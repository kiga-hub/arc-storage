package arc_volume

import (
	"fmt"
	"testing"
	"time"

	"github.com/kiga-hub/arc-storage/pkg/util"
)

// Test_isFirstCreateFile Test_isFirstCreateFile
func Test_isFirstCreateFile(t *testing.T) {
	tests := []struct {
		testName      string
		fileName      string
		storagestatus string
		sensorid      string
		samplerate    string
		createTime    string
		isFirst       bool
	}{
		{
			testName:      "#1_文件创建时间不同",
			fileName:      "A00000000000_A_20220809051411894669_20220809055700001335_N_48000_000000_00_040780_v1.0.133.wav",
			storagestatus: "N",
			sensorid:      "A00000000000",
			samplerate:    "48000",
			createTime:    "20220809051411894668",
			isFirst:       true,
		},
		{
			testName:      "#2_匹配",
			fileName:      "A00000000000_A_20220809051411894669_20220809055700001335_N_48000_000000_00_040780_v1.0.133.wav",
			storagestatus: "N",
			sensorid:      "A00000000000",
			samplerate:    "48000",
			createTime:    "20220809051411894669",
			isFirst:       false,
		},
		{
			testName:      "#3_传感器ID不同",
			fileName:      "A00000000000_A_20220809051411894669_20220809055700001335_N_48000_000000_00_040780_v1.0.133.wav",
			storagestatus: "N",
			sensorid:      "A00000000001",
			samplerate:    "48000",
			createTime:    "20220809051411894669",
			isFirst:       true,
		},
		{
			testName:      "#4_采样率不同",
			fileName:      "A00000000000_A_20220809051411894669_20220809055700001335_N_48000_000000_00_040780_v1.0.133.wav",
			storagestatus: "N",
			sensorid:      "A00000000000",
			samplerate:    "8000",
			createTime:    "20220809051411894669",
			isFirst:       true,
		},
	}
	for _, test := range tests {
		t.Run(test.testName, func(t *testing.T) {
			tmp := test.createTime[:14] + "." + test.createTime[14:]
			createTime, err := time.Parse("20060102150405.999999", tmp)
			if err != nil {
				t.Errorf(" time.Parse = %v", err)
			}

			if test.isFirst != isFirstCreateFile(test.fileName, test.storagestatus, test.sensorid, test.samplerate, createTime) {
				t.Errorf("createTime = %v, want %v", createTime, test.isFirst)
			}
		})
	}
}

// Test_GetTimeRangeFromFileName -
func Test_GetTimeRangeFromFileName(t *testing.T) {
	create := "20220809051411894669"
	tmp := create[:14] + "." + create[14:]
	createTime, err := time.Parse("20060102150405.999999", tmp)
	if err != nil {
		t.Errorf(" time.Parse = %v", err)
	}

	create2 := "20220809051411894668"
	tmp = create2[:14] + "." + create2[14:]
	createTime2, err := time.Parse("20060102150405.999999", tmp)
	if err != nil {
		t.Errorf(" time.Parse = %v", err)
	}

	tests := []struct {
		testName   string
		fileName   string
		startTime  time.Time
		sensorid   string
		samplerate string
		err        error
		equal      bool
	}{
		{
			testName:   "#1_文件创建时间相同",
			fileName:   "A00000000000_A_20220809051411894669_20220809055700001335_N_48000_000000_00_040780_v1.0.133.wav",
			sensorid:   "A00000000000",
			samplerate: "48000",
			startTime:  createTime,
			err:        nil,
			equal:      true,
		},
		{
			testName:   "#1_文件创建时间不同",
			fileName:   "A00000000000_A_20220809051411894669_20220809055700001335_N_48000_000000_00_040780_v1.0.133.wav",
			sensorid:   "A00000000000",
			samplerate: "48000",
			startTime:  createTime2,
			err:        nil,
			equal:      false,
		},
	}
	for _, test := range tests {
		t.Run(test.testName, func(t *testing.T) {
			start, _, id, _, samp, err := util.GetTimeRangeFromFileName(test.fileName)
			if err != test.err {
				t.Errorf("GetTimeRangeFromFileName = %v", err)
			}
			fmt.Printf("\ncreatetime:%s", test.startTime)
			fmt.Printf("\ncreatetime:%s\n", start)
			fmt.Printf("\ncreatetime:%v", test.startTime)
			fmt.Printf("\ncreatetime:%v\n", start)
			if test.equal != start.Equal(test.startTime) {
				t.Errorf("time = %v\n%v", start, test.startTime)
			}
			if test.equal != (start == test.startTime) {
				t.Errorf("time = %v\n%v", start, test.startTime)
			}
			if id != test.sensorid {
				t.Errorf("sensorid = %v,%v", id, test.sensorid)
			}
			if samp != test.samplerate {
				t.Errorf("samplerate = %v,%v", samp, test.samplerate)
			}
		})
	}
}
