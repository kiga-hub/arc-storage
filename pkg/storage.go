package pkg

import (
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/kiga-hub/arc-storage/pkg/util"
	"github.com/kiga-hub/arc/utils"

	"github.com/labstack/echo/v4"
	"github.com/spf13/cast"
)

// getSensorIDsfromStorage -
func (arc *ArcStorage) getSensorIDsfromStorage() ([]string, error) {
	var sensorids []string
	sensoridfolders, err := os.ReadDir(arc.config.Work.DataPath)
	if err != nil {
		arc.logger.Errorw("getSensorIDsfromStorage", "dataPath", arc.config.Work.DataPath, "err", err)
		return []string{}, err
	}
	for _, filename := range sensoridfolders {
		if filename.IsDir() {
			sensorids = append(sensorids, filename.Name())
		}
	}
	return sensorids, nil
}

// getSensorIDs Get Sensor IDs
func (arc *ArcStorage) getSensorIDs(c echo.Context) error {
	sensorids, err := arc.getSensorIDsfromStorage()
	if err != nil {
		arc.logger.Errorw("getSensorIDsfromStorage", "err", err)
		return c.JSON(http.StatusNotFound, utils.ResponseV2{
			Code: http.StatusNotFound,
			Msg:  http.StatusText(http.StatusNotFound)},
		)
	}
	if len(sensorids) < 1 {
		return c.JSON(http.StatusNotFound, utils.ResponseV2{
			Code: http.StatusNotFound,
			Msg:  http.StatusText(http.StatusNotFound)},
		)
	}

	// return SensorIDResponse
	return c.JSON(http.StatusOK, utils.ResponseV2{
		Code: Success,
		Msg:  "OK",
		Data: sensorids},
	)
}

// getSensorLists metadata from needle & parse data to buffer
func (arc *ArcStorage) getSensorLists(c echo.Context) error {
	var err error
	sensorIDStr := c.QueryParam("sensorid")
	if sensorIDStr == "" {
		arc.logger.Errorw("sensorid is null", "sensorid", sensorIDStr)
		return c.JSON(http.StatusBadRequest, utils.ResponseV2{
			Code: http.StatusBadRequest,
			Msg:  http.StatusText(http.StatusBadRequest)},
		)
	}

	sensorid := strings.ToUpper(sensorIDStr)
	filetype := c.QueryParam("type")

	var AllowExtMap map[string]bool = map[string]bool{
		"Arc": true,
	}
	if _, ok := AllowExtMap[filetype]; !ok {
		return c.JSON(http.StatusBadRequest, utils.ResponseV2{
			Code: http.StatusBadRequest,
			Msg:  http.StatusText(http.StatusBadRequest)},
		)
	}

	var t1 time.Time
	var t2 time.Time

	from := c.QueryParam("from")
	to := c.QueryParam("to")
	// if a timestamp is passed. it is cosidered as UTC time.
	if util.IsDigit(from) && util.IsDigit(to) {
		if len(from) == 10 && len(to) == 10 {
			t1 = time.Unix(cast.ToInt64(from), 0)
			t2 = time.Unix(cast.ToInt64(to), 0)
		} else if len(from) == 13 && len(to) == 13 {
			t1 = time.Unix(0, cast.ToInt64(from)*1e6)
			t2 = time.Unix(0, cast.ToInt64(to)*1e6)
		} else if len(from) == 16 && len(to) == 16 {
			t1 = time.Unix(0, cast.ToInt64(from)*1e3)
			t2 = time.Unix(0, cast.ToInt64(to)*1e3)
		} else if len(from) == 19 && len(to) == 19 {
			t1 = time.Unix(0, cast.ToInt64(from))
			t2 = time.Unix(0, cast.ToInt64(to))
		} else {
			return c.JSON(http.StatusBadRequest, utils.ResponseV2{
				Code: http.StatusBadRequest,
				Msg:  http.StatusText(http.StatusBadRequest)},
			)
		}
		// if it's not a timestamp. it's considered a string. if no custom timezone is defined. the default is UTC+8.
	} else {
		from = strings.ReplaceAll(from, " ", "T")
		to = strings.ReplaceAll(to, " ", "T")

		//长度不够20即认为没有带时区信息
		if len(from) < 20 {
			from += "+08:00"
		}
		if len(to) < 20 {
			to += "+08:00"
		}

		t1, err = time.Parse(time.RFC3339, from)
		if err != nil || t1.Unix() < 0 {
			return c.JSON(http.StatusBadRequest, utils.ResponseV2{
				Code: http.StatusBadRequest,
				Msg:  http.StatusText(http.StatusBadRequest)},
			)
		}
		t2, err = time.Parse(time.RFC3339, to)
		if err != nil || t2.Unix() < 0 {
			return c.JSON(http.StatusBadRequest, utils.ResponseV2{
				Code: http.StatusBadRequest,
				Msg:  http.StatusText(http.StatusBadRequest)},
			)
		}
	}

	if t2.Before(t1) || t2.Equal(t1) {
		return c.JSON(http.StatusBadRequest, utils.ResponseV2{
			Code: http.StatusBadRequest,
			Msg:  "time t1 = t2"},
		)
	}

	// get the tiem range in days.
	daysdiffer, count, err := util.GetDaysDiffer(t1.UTC().Format("2006-01-02 15:04:05"), t2.UTC().Format("2006-01-02 15:04:05"))
	if err != nil {
		return c.JSON(http.StatusNotFound, utils.ResponseV2{
			Code: http.StatusNotFound,
			Msg:  err.Error()},
		)
	}

	start := time.Now()
	defer func() {
		arc.logger.Infof("get data spend %s\n", time.Since(start).String())
	}()

	// get the file list
	bigfilelists := make(map[string]string, count)

	for _, day := range daysdiffer {
		path := arc.config.Work.DataPath + "/" + sensorid + "/" + day + "/" + filetype
		err := util.GetBigFileLists(path, bigfilelists, "")
		if err != nil {
			arc.logger.Errorw("GetBigFileLists", "dataPath", path, "err", err)
			// traverse the folder. skip if it doest not exist.
			continue
		}
	}

	var timestamplist []string
	var starttimestamps []string

	for key := range bigfilelists {
		timestamplist = append(timestamplist, key)
	}

	sort.Strings(timestamplist)

	for _, creattime := range timestamplist {
		begin, end, _,  err := util.GetTimeRangeFromFileName(bigfilelists[creattime])
		if err != nil {
			arc.logger.Errorw("GetTimeRangeFromFileName", "fileName", bigfilelists[creattime], "err", err)
			return c.JSON(http.StatusNotFound, utils.ResponseV2{
				Code: http.StatusNotFound,
				Msg:  http.StatusText(http.StatusNotFound)},
			)
		}
		// ignore if the start time is greater than the file save end time.
		if end.Before(t1) {
			continue
		}

		// exit is the end time is less than the file creation time.
		if t2.Before(begin) {
			arc.logger.Debugw("compare timestamp", "t2", t2, "begin", begin)
			continue
		}
		starttimestamps = append(starttimestamps, creattime)
	}

	if len(starttimestamps) < 1 {
		return c.JSON(http.StatusNotFound, utils.ResponseV2{
			Code: http.StatusNotFound,
			Msg:  http.StatusText(http.StatusNotFound)},
		)
	}

	// sort according to the time period.
	sort.Strings(starttimestamps)
	searchlists := []SensorItem{}

	for _, v := range starttimestamps {
		fullpath := bigfilelists[v][strings.LastIndex(bigfilelists[v], "/")+1:]
		start, end, _, err := util.GetTimeRangeFromFileName(fullpath)
		if err != nil {
			arc.logger.Errorw("GetTimeRangeFromFileName", "path", fullpath, "err", err)
			return c.JSON(http.StatusNotFound, utils.ResponseV2{
				Code: http.StatusNotFound,
				Msg:  http.StatusText(http.StatusNotFound)},
			)
		}

		duration := end.Sub(start).Microseconds()

		item := SensorItem{
			SensorID:     sensorid,
			DataType:     filetype,
			TimeFrom:     start.UnixNano() / 1e3, // return timestamp at the microsecond level.
			TimeTo:       end.UnixNano() / 1e3,
			TimeDuration: duration,
			Query: SensorQuery{
				URL:      "http://arc-storage/api/data/v1/history/arc?sensorid=" + sensorid + "&type=" + filetype + "&from=" + cast.ToString(start.UnixNano()/1e3) + "&to=" + cast.ToString(end.UnixNano()/1e3),
				Scheme:   "http",
				Domain:   "arc-storage",
				Port:     80,
				FullPath: "api/data/v1/history/arc?sensorid=" + sensorid + "&type=" + filetype + "&from=" + cast.ToString(start.UnixNano()/1e3) + "&to=" + cast.ToString(end.UnixNano()/1e3),
				Path:     "api/data/v1/history/arc",
				SensorID: sensorid,
				Type:     filetype,
				TimeFrom: start.UnixNano() / 1e3,
				TimeTo:   end.UnixNano() / 1e3,
			},
		}
		searchlists = append(searchlists, item)
	}

	return c.JSON(http.StatusOK, utils.ResponseV2{
		Code: Success,
		Msg:  "OK",
		Data: searchlists},
	)
}
