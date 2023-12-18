package pkg

const (
	// Success -
	Success = 0
)

// SensorIDResponse is the response for getting sensor ids
type SensorIDResponse struct {
	Code int      `json:"code,omitempty"`
	Msg  string   `json:"msg,omitempty"`
	Data []string `json:"data,omitempty"`
}

// SensorItem response sensor information
type SensorItem struct {
	SensorID     string      `json:"sensorid,omitempty"`
	DataType     string      `json:"data_type,omitempty"`
	Query        SensorQuery `json:"query,omitempty"`
	TimeFrom     int64       `json:"time_from,omitempty"`
	TimeTo       int64       `json:"time_to,omitempty"`
	TimeDuration int64       `json:"time_duration,omitempty"`
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
