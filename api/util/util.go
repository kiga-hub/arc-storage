package util

// // GetAudioDataV2 - get all of sensor data. filetype: audio/vibrate
// func GetAudioDataV2(host, sensorID string, from, to time.Time, inside bool, logger logging.ILogger) ([]byte, error) {
// 	if sensorID == "" {
// 		return nil, fmt.Errorf("empty sensor id")
// 	}

// 	cfg := &client.TransportConfig{
// 		Host:     host,
// 		BasePath: "/",
// 		Schemes:  []string{"http"},
// 	}
// 	transport := httptransport.New(cfg.Host, cfg.BasePath, cfg.Schemes)
// 	transport.Consumers["audio/wav"] = runtime.ByteStreamConsumer()
// 	transport.Producers["audio/wav"] = runtime.ByteStreamProducer()
// 	arc := client.New(transport, nil)
// 	ctx, cancelFunc := context.WithTimeout(context.Background(), 30*time.Second)
// 	defer cancelFunc()

// 	f := from.UnixNano() / 1e3
// 	t := to.UnixNano() / 1e3

// 	params := history_raw_data.NewGetrawParamsWithContext(ctx)
// 	params.Sensorid = sensorID
// 	params.From = &f
// 	params.To = &t
// 	params.Type = "audio"
// 	params.Inside = &inside

// 	buf := bytes.NewBuffer(make([]byte, 0, 384044))
// 	ok, err := arc.HistoryRawData.Getraw(params, buf)
// 	if err != nil {
// 		logger.Errorw("arc.HistoryRawData.Getraw", "err", err)
// 	} else if !swag.IsZero(ok) && !swag.IsZero(ok.Payload) {
// 		r := make([]byte, buf.Len())
// 		copy(r, buf.Bytes())
// 		return r, nil
// 	}

// 	logger.Errorw("data not found", "ok", ok, "err", err, "sensorID", sensorID, "from", *params.From, "to", *params.To, "systemTime_UTC", time.Now().UTC(), "inside", *params.Inside)
// 	return nil, fmt.Errorf("data not found %s from: %v to: %v ,systemTime: %v, inside: %v", sensorID, *params.From, *params.To, time.Now().UTC(), *params.Inside)
// }
