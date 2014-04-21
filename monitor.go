// Stream parsers and keepers.
package main

import (
	"expvar"
	"fmt"
	"github.com/grafov/bcast"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

// Run monitors for each stream.
func StreamMonitor() {
	var debugvars = expvar.NewMap("streams")
	var requestedTasks = expvar.NewInt("requested-tasks")
	var queueSizeHLSTasks = expvar.NewInt("hls-tasks-queue")
	var executedHLSTasks = expvar.NewInt("hls-tasks-done")
	var expiredHLSTasks = expvar.NewInt("hls-tasks-expired")
	var queueSizeHDSTasks = expvar.NewInt("hds-tasks-queue")
	var executedHDSTasks = expvar.NewInt("hds-tasks-done")
	var expiredHDSTasks = expvar.NewInt("hds-tasks-expired")
	var queueSizeHTTPTasks = expvar.NewInt("http-tasks-queue")
	var executedHTTPTasks = expvar.NewInt("http-tasks-done")
	var expiredHTTPTasks = expvar.NewInt("http-tasks-expired")
	var queueSizeWVTasks = expvar.NewInt("wv-tasks-queue")
	var executedWVTasks = expvar.NewInt("wv-tasks-done")
	var expiredWVTasks = expvar.NewInt("wv-tasks-expired")
	var hlscount, hdscount, wvcount, httpcount int
	var hlsprobecount int

	debugvars.Set("requested-tasks", requestedTasks)
	debugvars.Set("hls-tasks-queue", queueSizeHLSTasks)
	debugvars.Set("hls-tasks-done", executedHLSTasks)
	debugvars.Set("hls-tasks-expired", expiredHLSTasks)
	debugvars.Set("hds-tasks-queue", queueSizeHDSTasks)
	debugvars.Set("hds-tasks-done", executedHDSTasks)
	debugvars.Set("hds-tasks-expired", expiredHDSTasks)
	debugvars.Set("http-tasks-queue", queueSizeHTTPTasks)
	debugvars.Set("http-tasks-done", executedHTTPTasks)
	debugvars.Set("http-tasks-expired", expiredHTTPTasks)
	debugvars.Set("wv-tasks-queue", queueSizeWVTasks)
	debugvars.Set("wv-tasks-done", executedWVTasks)
	debugvars.Set("wv-tasks-expired", expiredWVTasks)

	ctl := bcast.NewGroup()
	go ctl.Broadcasting(0)
	go Heartbeat(ctl)

	// запуск проберов и потоков
	for gname, gdata := range cfg.GroupParams {
		switch gdata.Type {
		case HLS:
			gtasks := make(chan *Task)
			for i := 0; i < gdata.Probers; i++ {
				go CupertinoProber(ctl, gtasks, debugvars)
				hlsprobecount++
			}
			gchunktasks := make(chan *Task)
			for i := 0; i < gdata.MediaProbers; i++ {
				go MediaProber(ctl, HLS, gchunktasks, debugvars)
			}
			for _, stream := range *cfg.GroupStreams[gname] {
				go StreamBox(ctl, stream, HLS, gtasks, debugvars)
				hlscount++
			}
		case HDS:
			gtasks := make(chan *Task)
			for i := 0; i < gdata.Probers; i++ {
				go SanjoseProber(ctl, gtasks, debugvars)
			}
			gchunktasks := make(chan *Task)
			for i := 0; i < gdata.MediaProbers; i++ {
				go MediaProber(ctl, HDS, gchunktasks, debugvars)
			}
			for _, stream := range *cfg.GroupStreams[gname] {
				go StreamBox(ctl, stream, HDS, gtasks, debugvars)
				hdscount++
			}
		case HTTP:
			gtasks := make(chan *Task)
			for i := 0; i < gdata.Probers; i++ {
				go SimpleProber(ctl, gtasks, debugvars)
			}
			for _, stream := range *cfg.GroupStreams[gname] {
				go StreamBox(ctl, stream, HTTP, gtasks, debugvars)
				httpcount++
			}
		case WV:
			gtasks := make(chan *Task)
			for i := 0; i < gdata.Probers; i++ {
				go WidevineProber(ctl, gtasks, debugvars)
			}
			for _, stream := range *cfg.GroupStreams[gname] {
				go StreamBox(ctl, stream, WV, gtasks, debugvars)
				wvcount++
			}
		}
	}

	if hlsprobecount > 0 {
		fmt.Printf("%d HLS probers started.\n", hlsprobecount)
	} else {
		println("No HLS probers started.")
	}
	// if cfg.TotalProbersHDS > 0 {
	// 	fmt.Printf("%d HDS probers started.\n", cfg.TotalProbersHDS)
	// } else {
	// 	println("No HDS probers started.")
	// }
	// if cfg.TotalProbersHTTP > 0 {
	// 	fmt.Printf("%d HTTP probers started.\n", cfg.TotalProbersHTTP)
	// } else {
	// 	println("No HTTP probers started.")
	// }
	// if cfg.TotalProbersWV > 0 {
	// 	fmt.Printf("%d Widevine VOD probers started.\n", cfg.TotalProbersWV)
	// } else {
	// 	println("No Widevine probers started.")
	// }
	if hlscount > 0 {
		StatsGlobals.TotalHLSMonitoringPoints = hlscount
		fmt.Printf("%d HLS monitors started.\n", hlscount)
	} else {
		println("No HLS monitors started.")
	}
	if hdscount > 0 {
		StatsGlobals.TotalHDSMonitoringPoints = hdscount
		fmt.Printf("%d HDS monitors started.\n", hdscount)
	} else {
		println("No HDS monitors started.")
	}
	if httpcount > 0 {
		StatsGlobals.TotalHTTPMonitoringPoints = httpcount
		fmt.Printf("%d HTTP monitors started.\n", httpcount)
	} else {
		println("No HTTP monitors started.")
	}
	if wvcount > 0 {
		StatsGlobals.TotalWVMonitoringPoints = wvcount
		fmt.Printf("%d Widevine monitors started.\n", wvcount)
	} else {
		println("No Widevine monitors started.")
	}

	StatsGlobals.TotalMonitoringPoints = hlscount + hdscount + httpcount + wvcount
}

// Мониторинг и статистика групп потоков.
func GroupBox(ctl *bcast.Group, group string, streamType StreamType, taskq chan *Task, debugvars *expvar.Map) {
}

// Container keep single stream properties and regulary make tasks for appropriate probers.
func StreamBox(ctl *bcast.Group, stream Stream, streamType StreamType, taskq chan *Task, debugvars *expvar.Map) {
	var checkCount uint64 // число прошедших проверок
	var addSleepToBrokenStream time.Duration
	var min, max int
	var command Command
	var online bool = false
	//	var queueLimit uint

	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Stream %s trace: %s\n", stream.Name, r)
		}
	}()

	task := &Task{Stream: stream, ReplyTo: make(chan Result)}
	switch streamType {
	case HTTP:
		task.ReadBody = false
	//	queueLimit = cfg.Params.ProbersHTTP
	case HLS:
		task.ReadBody = true
		// queueLimit = cfg.Params.ProbersHLS
	case HDS:
		task.ReadBody = true
		// queueLimit = cfg.Params.ProbersHDS
	case WV:
		task.ReadBody = false
	default:
		task.ReadBody = false
		// queueLimit = 42 // XXX
	}
	// errhistory := make(map[ErrHistoryKey]uint)     // duplicates ErrHistory from stats
	// errtotals := make(map[ErrTotalHistoryKey]uint) // duplicates ErrTotalHistory from stats
	ctlrcv := ctl.Join() // управление мониторингом

	for {
		select {
		case recv := <-ctlrcv.In:
			command = recv.(Command)
			switch command {
			case START_MON:
				online = true
			case STOP_MON:
				online = false
			}
		default:
			if !online {
				time.Sleep(1 * time.Second)
				continue
			}
			max = int(cfg.Params(stream.Group).TimeBetweenTasks)
			min = int(cfg.Params(stream.Group).TimeBetweenTasks / 4. * 3.)
			time.Sleep(time.Duration(rand.Intn(max-min)+min)*time.Second + addSleepToBrokenStream) // randomize streams order
			task.TTL = time.Now().Add(time.Duration(cfg.Params(stream.Group).TaskTTL * time.Second))
			taskq <- task
			debugvars.Add("requested-tasks", 1)
			result := <-task.ReplyTo
			if result.ErrType == TTLEXPIRED {
				continue
			} else {
				checkCount++
				if checkCount > 2 {
					fmt.Printf("Repeated %d times %s\n", checkCount, task.Name)
				}
			}

			go SaveStats(stream, result)

			// curhour := result.Started.Format("06010215")
			// prevhour := result.Started.Add(-1 * time.Hour).Format("06010215")
			// errhistory[ErrHistoryKey{Curhour: curhour, ErrType: result.ErrType}]++
			// errtotals[ErrTotalHistoryKey{Curhour: curhour}]++
			max = int(cfg.Params(stream.Group).CheckBrokenTime)
			min = int(cfg.Params(stream.Group).CheckBrokenTime / 4. * 3.)

			switch {
			// too much repeatable errors per hour:
			// case errtotals[ErrTotalHistoryKey{Curhour: curhour}] > 6, errtotals[ErrTotalHistoryKey{Curhour: prevhour}] > 6:
			// 	addSleepToBrokenStream = time.Duration(rand.Intn(max-min)+min) * time.Second
			// permanent error, not a timeout:
			case result.ErrType > CRITICAL_LEVEL, result.ErrType == TTLEXPIRED:
				addSleepToBrokenStream = time.Duration(rand.Intn(max-min)+min) * time.Second
			// works ok:
			case result.ErrType == SUCCESS:
				addSleepToBrokenStream = 0
			default:
				addSleepToBrokenStream = 0
			}
			//result.TotalErrs = errtotals[ErrTotalHistoryKey{Curhour: curhour}]

			if result.ErrType != TTLEXPIRED {
				if result.ErrType >= WARNING_LEVEL {
					go Log(ERROR, stream, result)
				} else {
					if result.Elapsed >= cfg.Params(stream.Group).VerySlowWarningTimeout*time.Second {
						result.ErrType = VERYSLOW
						go Log(WARNING, stream, result)
					} else if result.Elapsed >= cfg.Params(stream.Group).SlowWarningTimeout*time.Second {
						result.ErrType = SLOW
						go Log(WARNING, stream, result)
					}
				}
			}
		}
	}
}

// Check & report internet availability. Stop all probers when sample internet resources not available.
// Refs to config option ``samples``.
func Heartbeat(ctl *bcast.Group) {
	var previous bool

	ctlsnr := ctl.Join()

	time.Sleep(1 * time.Second)

	for {
		for _, uri := range cfg.Samples {
			client := NewTimeoutClient(12*time.Second, 6*time.Second)
			req, err := http.NewRequest("HEAD", uri, nil)
			if err != nil {
				fmt.Println("Internet not available. All checks stopped.")
				StatsGlobals.MonitoringState = false
				continue
			}
			_, err = client.Do(req)
			if err != nil {
				StatsGlobals.MonitoringState = false
				continue
			}
			StatsGlobals.MonitoringState = true
		}
		if previous != StatsGlobals.MonitoringState {
			if StatsGlobals.MonitoringState {
				ctlsnr.Send(START_MON)
				fmt.Println("Internet Ok. Monitoring started.")
			} else {
				ctlsnr.Send(STOP_MON)
				fmt.Println("Internet not available. Monitoring stopped.")
			}
		}
		previous = StatsGlobals.MonitoringState
		time.Sleep(4 * time.Second)
	}
}

// Helper for expired tasks. Return result with TTL Expired status.
func TaskExpired(task *Task) *Result {
	result := &Result{Started: time.Now(), Elapsed: 0 * time.Second}
	result.ContentLength = -1
	result.ErrType = TTLEXPIRED
	return result
}

// Helper. Execute stream check task and return result with check status.
func ExecHTTP(task *Task) *Result {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("trace dumped in ExecHTTP:", r)
		}
	}()

	result := &Result{Task: task, Started: time.Now(), Elapsed: 0 * time.Second}
	if !strings.HasPrefix(task.URI, "http://") && !strings.HasPrefix(task.URI, "https://") {
		result.ErrType = BADURI
		result.HTTPCode = 0
		result.HTTPStatus = ""
		result.ContentLength = -1
		return result
	}
	client := NewTimeoutClient(cfg.Params(task.Group).ConnectTimeout*time.Second, cfg.Params(task.Group).RWTimeout*time.Second)
	req, err := http.NewRequest("GET", task.URI, nil) // TODO разделить метод по проберам cfg.Params(task.Group).MethodHTTP
	if err != nil {
		fmt.Println(err)
		result.ErrType = BADURI
		result.HTTPCode = 0
		result.HTTPStatus = ""
		result.ContentLength = -1
		return result
	}
	req.Header.Set("User-Agent", fmt.Sprintf("%s/%s", SURFER, VERSION))
	resp, err := client.Do(req)
	result.Elapsed = time.Since(result.Started)
	if err != nil {
		fmt.Printf("Connect timeout %s: %v\n", result.Elapsed, err)
		if result.Elapsed > cfg.Params(task.Group).ConnectTimeout*time.Second {
			result.ErrType = CTIMEOUT
		} else {
			result.ErrType = REFUSED
		}
		result.HTTPCode = 0
		result.HTTPStatus = ""
		result.ContentLength = -1
		return result
	}
	result.HTTPCode = resp.StatusCode
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		result.ErrType = BADSTATUS
	}
	result.HTTPStatus = resp.Status
	result.ContentLength = resp.ContentLength
	result.Headers = resp.Header
	if task.ReadBody {
		result.RealContentLength, err = result.Body.ReadFrom(resp.Body)
		if err != nil {
			result.ErrType = BODYREAD
		}
	}
	resp.Body.Close()
	if result.RealContentLength > 0 && result.ContentLength != result.RealContentLength {
		result.ErrType = BADLENGTH
	}
	return result
}

// Helper. Verify HLS specific things.
func verifyHLS(cfg *Config, task *Task, result *Result) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("trace dumped in HLS parser:", r)
			result.ErrType = HLSPARSER
		}
	}()
}

// Ограничивать число запросов в ед.времени на ip
// func RateLimiter() {

// }
