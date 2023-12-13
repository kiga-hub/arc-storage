package util

import (
	"reflect"
	"testing"
)

func CaseGetBetweenDates(t *testing.T) {
	tests := []struct {
		start     string
		end       string
		wantslice []string
	}{
		{
			start:     "[]string{}",
			end:       "2006-01-02 15:04:05",
			wantslice: []string{},
		},
		{
			start:     "2006-01-02 15:04:05",
			end:       "[]string{}",
			wantslice: []string{},
		},
		{
			start:     "2006-01-02 15:04:05",
			end:       "2006-01-02 15:04:05",
			wantslice: []string{"20060102"},
		},
		{
			start:     "2006-01-02 15:04:05",
			end:       "2006-01-01 15:04:05",
			wantslice: []string{},
		},
		{
			start:     "2006-01-02 15:04:05",
			end:       "2006-01-03 15:04:05",
			wantslice: []string{"20060102", "20060103"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.start, func(t *testing.T) {
			if got := GetBetweenDates(tt.start, tt.end); !reflect.DeepEqual(got, tt.wantslice) {
				t.Errorf("GetBetweenDates got:%v want:%v", got, tt.wantslice)
			}
		})
	}
}

func CaseGetHoueDiffer(t *testing.T) {
	date := map[string][]string{}

	tests := []struct {
		start    string
		end      string
		duration string
		wantmap  map[string][]string
		wanterr  error
	}{
		{
			start:    "test err",
			end:      "2006-01-02 15:04:05",
			duration: "1h",
			wantmap:  nil,
			wanterr:  nil,
		},
		{
			start:    "2006-01-02 14:04:05",
			end:      "test err",
			duration: "1h",
			wantmap:  nil,
			wanterr:  nil,
		},
		{
			start:    "2006-01-02 14:04:05",
			end:      "2006-01-02 14:04:05",
			duration: "",
			wantmap:  nil,
			wanterr:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.start, func(t *testing.T) {
			wantmap, count, err := GetHourDiffer(tt.start, tt.end)
			if wantmap != nil && err != tt.wanterr {
				t.Fatalf("GetHourDiffer %v %d", err, count)
			}
		})
	}

	tests = []struct {
		start    string
		end      string
		duration string
		wantmap  map[string][]string
		wanterr  error
	}{
		{
			start:    "2006-01-02 14:04:05",
			end:      "2006-01-02 15:04:05",
			wantmap:  date,
			duration: "1h",
			wanterr:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.start, func(t *testing.T) {
			if _, _, err := GetHourDiffer(tt.start, tt.end); err != tt.wanterr {
				t.Errorf("GetHoueDiffer got:%v want:%v", err, tt.wanterr)
			}
		})
	}
}
func TestTimeUtil(t *testing.T) {
	CaseGetBetweenDates(t)
	CaseGetHoueDiffer(t)
}
