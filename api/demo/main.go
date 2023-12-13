package main

// func main() {
// 	cacheAPITest1()
// 	cacheAPITest2()
// 	cacheAPITest3()
// }
// func cacheAPITest1() {
// 	sensorid := "A00000000000"
// 	for i := 0; i < 100; i++ {
// 		now := time.Now()

// 		start := time.Unix(0, cast.ToInt64(1652444064000)*int64(time.Millisecond))
// 		end := time.Unix(0, cast.ToInt64(1652444169000)*int64(time.Millisecond))
// 		fmt.Println("start", start)
// 		fmt.Println("end", end) // 192.168.8.244
// 		bs, err := util.GetAudioDataV2("localhost", sensorid, start, end, false, &logging.NoopLogger{})
// 		if err != nil {
// 			fmt.Printf("sensorid %s,%v\n", sensorid, err)
// 		}
// 		fmt.Printf("sensorid %s,spent %f s, length %d ,bytes\n", sensorid, time.Since(now).Seconds(), len(bs))
// 		time.Sleep(time.Millisecond * time.Duration(100))
// 	}

// }

// func cacheAPITest2() {
// 	baseid := []byte{0xA0, 0x00, 0x00, 0x00, 0x00, 0x00}
// 	id := getClientID(baseid)
// 	sensorids := []string{}

// 	for i := 0; i < 10; i++ {
// 		sensorids = append(sensorids, fmt.Sprintf("%012X", id+uint64(i)))
// 	}

// 	var wg sync.WaitGroup

// 	for _, sensorid := range sensorids {
// 		wg.Add(1)
// 		go func(sensorid string) {
// 			defer func() {
// 				wg.Done()
// 			}()
// 			start := time.Now().Add(-time.Second * time.Duration(5))
// 			end := time.Now()

// 			req, err := http.NewRequest("GET", "http://192.168.8.220/api/data/v1/history/data", nil)
// 			if err != nil {
// 				fmt.Println(err)
// 				return
// 			}

// 			q := req.URL.Query()
// 			q.Add("sensorid", sensorid)
// 			q.Add("from", cast.ToString(start.UnixNano()/1e6))
// 			q.Add("to", cast.ToString(end.UnixNano()/1e6))
// 			q.Add("type", "audio")
// 			req.URL.RawQuery = q.Encode()

// 			// fmt.Println(req.URL.String())

// 			var resp *http.Response
// 			resp, err = http.DefaultClient.Do(req)
// 			if err != nil {
// 				fmt.Println(err)
// 			}
// 			defer resp.Body.Close()

// 			body, err := ioutil.ReadAll(resp.Body)
// 			if err != nil {
// 				fmt.Println("ioutil.ReadAll", sensorid, err)
// 			}

// 			fmt.Printf("sensorid %s, length %d ,bytes\n", sensorid, len(body))
// 			// str {"status":"Error","err":"member not found"}   - 47
// 			time.Sleep(time.Millisecond * time.Duration(800))
// 		}(sensorid)
// 	}
// 	wg.Wait()

// }

// func cacheAPITest3() {
// 	baseid := []byte{0xA0, 0x00, 0x00, 0x00, 0x00, 0x00}
// 	delay := 0
// 	duration := 3
// 	go func() {
// 		for {
// 			now := time.Now().UTC()
// 			delay1, _ := time.ParseDuration("-" + cast.ToString(delay+duration) + "s")
// 			delay2, _ := time.ParseDuration("-" + cast.ToString(delay) + "s")
// 			start := now.Add(delay1)
// 			end := now.Add(delay2)
// 			fmt.Println(start, end)
// 			// cacheHttpRequestTest(baseid, start, end)
// 			cacheAPITest(baseid, start, end)
// 			time.Sleep(time.Second * time.Duration(duration))
// 			fmt.Println("\n\n\n---------------------")
// 		}
// 	}()
// 	select {}
// }

// func cacheAPITest(baseid []byte, start, end time.Time) {
// 	id := getClientID(baseid)
// 	sensorids := []string{}

// 	for i := 0; i < 10; i++ {
// 		sensorids = append(sensorids, fmt.Sprintf("%012X", id+uint64(i)))
// 	}

// 	var wg sync.WaitGroup

// 	for _, sensorid := range sensorids {
// 		wg.Add(1)
// 		go func(sensorid string) {
// 			defer func() {
// 				wg.Done()
// 			}()
// 			now := time.Now()
// 			bs, err := util.GetAudioDataV2("192.168.8.220", sensorid, start, end, false, &logging.NoopLogger{})
// 			if err != nil {
// 				fmt.Printf("sensorid %s,%v\n", sensorid, err)
// 				return
// 			}
// 			fmt.Printf("sensorid %s,spent %d ms, length %d ,bytes\n", sensorid, time.Since(now).Milliseconds(), len(bs))
// 			time.Sleep(time.Millisecond * time.Duration(800))
// 		}(sensorid)
// 	}
// 	wg.Wait()
// }

// // getClientID transfer string 2 ID
// func getClientID(bytes []byte) uint64 {
// 	var res uint64
// 	c := len(bytes)
// 	if c > 6 {
// 		c = 6
// 	}
// 	for i, b := range bytes {
// 		res += uint64(b) << ((c - (i + 1)) * 8)
// 		if i >= 5 {
// 			break
// 		}
// 	}
// 	return res
// }
