package pkg

const (
	// Success 成功返回OK
	Success = 0
)

// ArcDataInfo include arc value
type ArcDataInfo struct {
	Time int64   `db:"time"`
	Data float32 `db:"data"`
}

// ArcDataData contains tempratures of a sensor
type ArcDataData struct {
	SensorID string      `json:"sensorid,omitempty"`
	Data     ArcDataInfo `json:"data,omitempty"`
	Count    int64       `json:"count"`
}

// ArcDataResponse is the response for getting temprature
type ArcDataResponse struct {
	Code int           `json:"code,omitempty"`
	Msg  string        `json:"msg,omitempty"`
	Data []ArcDataData `json:"data,omitempty"`
}

// SensorIDResponse is the response for getting sensor ids
type SensorIDResponse struct {
	Code int      `json:"code,omitempty"`
	Msg  string   `json:"msg,omitempty"`
	Data []string `json:"data,omitempty"`
}

// FileResponse is the response for downloading
type FileResponse struct {
	Code int        `json:"code,omitempty"`
	Msg  string     `json:"msg,omitempty"`
	Data SensorItem `json:"data"`
}

// SearchListsResponse is the response for getting file info
type SearchListsResponse struct {
	Code int          `json:"code,omitempty"`
	Msg  string       `json:"msg,omitempty"`
	Data []SensorItem `json:"data"`
}

// RecordCopyTimeResponse -
type RecordCopyTimeResponse struct {
	Code int    `json:"code,omitempty"`
	Msg  string `json:"msg,omitempty"`
	Data string `json:"data,omitempty"`
}

// SensorItem response sensor information
type SensorItem struct {
	SensorID     string      `json:"sensorid,omitempty"`      // 传感器id
	DataType     string      `json:"data_type,omitempty"`     // 数据类型
	SampleRate   string      `json:"sample_rate,omitempty"`   // 采样率
	Query        SensorQuery `json:"query,omitempty"`         // 数据请求方式
	Channel      int         `json:"channel,omitempty"`       // 通道数
	TimeFrom     int64       `json:"time_from,omitempty"`     // 开始时间(精度到毫秒)
	TimeTo       int64       `json:"time_to,omitempty"`       // 结束时间(精度到毫秒)
	TimeDuration int64       `json:"time_duration,omitempty"` // 时间长度(毫秒)
	DataSize     int64       `json:"data_size,omitempty"`     // 数据大小(字节)
}

// SensorQuery query details
type SensorQuery struct {
	URL      string `json:"url,omitempty"`
	Scheme   string `json:"scheme,omitempty"`
	Domain   string `json:"domain,omitempty"`
	Port     int    `json:"port,omitempty"`
	FullPath string `json:"full_path,omitempty"`
	Path     string `json:"path,omitempty"`
	SensorID string `json:"sensorid,omitempty"`
	Type     string `json:"type,omitempty"`
	TimeFrom int64  `json:"time_from,omitempty"`
	TimeTo   int64  `json:"time_to,omitempty"`
}
