package util

import (
	"strconv"
	"strings"
	"time"
)

// GetBetweenDates -
func GetBetweenDates(startdate, enddate string) []string {
	d := []string{}
	timeFormatTpl := "2006-01-02 15:04:05"

	date, err := time.Parse(timeFormatTpl, startdate)
	if err != nil {
		return d
	}
	x := date.Format("20060102")
	date2, err := time.Parse(timeFormatTpl, enddate)
	if err != nil {
		return d
	}
	y := date2.Format("20060102")

	if x == y {
		d = append(d, x)
		return d
	}

	if date2.Before(date) {
		return d
	}

	timeFormatTpl = "20060102"
	date2Str := date2.Format(timeFormatTpl)
	d = append(d, date.Format(timeFormatTpl))
	for {
		date = date.AddDate(0, 0, 1)
		dateStr := date.Format(timeFormatTpl)
		d = append(d, dateStr)
		if dateStr == date2Str {
			break
		}
	}
	return d
}

// GetDaysDiffer -
func GetDaysDiffer(from, to string) ([]string, int, error) {
	day := []string{}
	index := 0
	t1, err := time.Parse("2006-01-02 15:04:05", from)
	if err != nil {
		return nil, 0, err
	}
	t2, err := time.Parse("2006-01-02 15:04:05", to)
	if err != nil {
		return nil, 0, err
	}
	day = append(day, t1.Format("20060102"))
	index++

	for t1.Format("20060102") < t2.Format("20060102") {
		t1 = t1.Add(time.Hour * 24)
		day = append(day, t1.Format("20060102"))
		index++
	}
	return day, index, nil
}

// GetHourDiffer -
func GetHourDiffer(starttime, endtime string) (map[string][]string, int, error) {
	date := map[string][]string{}
	timeTemplate := "2006010215"
	t1, err := time.Parse("2006-01-02 15:04:05", starttime)
	if err != nil {
		return nil, 0, err
	}
	t2, err := time.Parse("2006-01-02 15:04:05", endtime)
	if err != nil {
		return nil, 0, err
	}

	count := 0
	s1 := t1.Format(timeTemplate)
	s2 := t2.Format(timeTemplate)
	int64t1, err := strconv.ParseInt(s1, 10, 64)
	if err != nil {
		return nil, 0, err
	}
	int64t2, err := strconv.ParseInt(s2, 10, 64)
	if err != nil {
		return nil, 0, err
	}
	for {
		if int64t1 <= int64t2 {
			n := strconv.FormatInt(int64t1, 10)[8:]
			a, err := strconv.Atoi(n)
			if err != nil {
				return nil, 0, err
			}
			if a <= 23 {
				key := strconv.FormatInt(int64t1, 10)[:8]
				date[key] = append(date[key], strconv.FormatInt(int64t1, 10)[8:])

				count++
			}
			int64t1++
		} else {
			break
		}
	}

	return date, count, nil
}

// TimeStringReplace -
func TimeStringReplace(timer time.Time) string {
	timestr := strings.Replace(timer.UTC().Format("20060102150405.999999"), ".", "", 1)
	for {
		if len(timestr) < 20 {
			timestr += "0"
		} else {
			break
		}
	}
	return timestr
}

// IsZeroTime 时间格式化: 零点，整小时，整分
func IsZeroTime(timestamp, timestamp2 time.Time, duration string, num int) bool {
	if timestamp.Unix() < 0 {
		return timestamp.Format("200601021504") == timestamp2.Format("200601021504")
	}
	switch duration {
	case "day":
		return time.Date(timestamp.Year(), timestamp.Month(), timestamp.Day()+num, 0, 0, 0, 0, timestamp.Location()).Format("20060102") == timestamp2.Format("20060102")
	case "hour":
		return time.Date(timestamp.Year(), timestamp.Month(), timestamp.Day(), timestamp.Hour()+num, 0, 0, 0, timestamp.Location()).Format("2006010215") == timestamp2.Format("2006010215")
	case "min":
		return time.Date(timestamp.Year(), timestamp.Month(), timestamp.Day(), timestamp.Hour(), timestamp.Minute()+num, 0, 0, timestamp.Location()).Format("200601021504") == timestamp2.Format("200601021504")
	default:
		return timestamp.Format("200601021504") == timestamp2.Format("200601021504")
	}
}
