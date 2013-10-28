// Stream parsers and keepers.
package main

import (
	"expvar"
	"fmt"
	"github.com/grafov/bcast"
	"github.com/grafov/m3u8"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

// Run monitors for each stream.
func StreamMonitor(cfg *Config) {
	var i uint
	var debugvars = expvar.NewMap("streams")
	var requestedTasks = expvar.NewInt("requested-tasks")
	var executedHLSTasks = expvar.NewInt("hls-tasks-done")
	var executedHDSTasks = expvar.NewInt("hds-tasks-done")
	var executedHTTPTasks = expvar.NewInt("http-tasks-done")

	debugvars.Set("requested-tasks", requestedTasks)
	debugvars.Set("hls-tasks-done", executedHLSTasks)
	debugvars.Set("hds-tasks-done", executedHDSTasks)
	debugvars.Set("http-tasks-done", executedHTTPTasks)

	// channels for different task types
	httptasks := make(chan *Task, len(cfg.StreamsHTTP)*4)
	hlstasks := make(chan *Task, len(cfg.StreamsHLS)*4)
	hdstasks := make(chan *Task, len(cfg.StreamsHLS)*4)
	chunktasks := make(chan *Task, (cfg.Params.ProbersHLS+cfg.Params.ProbersHDS+cfg.Params.ProbersHTTP)*8) // TODO тут не задачи, другой тип

	ctl := bcast.NewGroup()
	go ctl.Broadcasting(0)
	go Heartbeat(cfg, ctl)

	for i = 0; i < cfg.Params.ProbersHLS; i++ {
		go CupertinoProber(cfg, ctl, hlstasks, debugvars)
	}
	if cfg.Params.ProbersHLS > 0 {
		fmt.Printf("%d HLS probers started.\n", cfg.Params.ProbersHLS)
	}

	for i = 0; i < cfg.Params.ProbersHDS; i++ {
		go SanjoseProber(cfg, ctl, hdstasks, debugvars)
	}
	if cfg.Params.ProbersHDS > 0 {
		fmt.Printf("%d HDS probers started.\n", cfg.Params.ProbersHDS)
	}

	for i = 0; i < cfg.Params.ProbersHTTP; i++ {
		go SimpleProber(cfg, ctl, httptasks, debugvars)
	}
	if cfg.Params.ProbersHTTP > 0 {
		fmt.Printf("%d HTTP probers started.\n", cfg.Params.ProbersHTTP)
	}

	for i = 0; i < cfg.Params.MediaProbers; i++ {
		go MediaProber(cfg, ctl, HLS, chunktasks, debugvars)
	}
	if cfg.Params.MediaProbers > 0 {
		fmt.Printf("%d media probers for HLS started.\n", cfg.Params.MediaProbers)
	}

	for i = 0; i < cfg.Params.MediaProbers; i++ {
		go MediaProber(cfg, ctl, HDS, chunktasks, debugvars)
	}
	if cfg.Params.MediaProbers > 0 {
		fmt.Printf("%d media probers for HDS started.\n", cfg.Params.MediaProbers)
	}

	for _, group := range cfg.GroupsHLS {
		go GroupBox(cfg, ctl, group, HLS, hlstasks, debugvars)
	}

	for _, group := range cfg.GroupsHTTP {
		go GroupBox(cfg, ctl, group, HTTP, httptasks, debugvars)
	}

	for _, stream := range cfg.StreamsHLS {
		go StreamBox(cfg, ctl, stream, HLS, hlstasks, debugvars)
	}
	if len(cfg.StreamsHLS) > 0 {
		StatsGlobals.TotalHLSMonitoringPoints = len(cfg.StreamsHLS)
		fmt.Printf("%d HLS monitors started.\n", StatsGlobals.TotalHLSMonitoringPoints)
	}

	for _, stream := range cfg.StreamsHTTP {
		go StreamBox(cfg, ctl, stream, HTTP, httptasks, debugvars)
	}
	if len(cfg.StreamsHTTP) > 0 {
		StatsGlobals.TotalHTTPMonitoringPoints = len(cfg.StreamsHTTP)
		fmt.Printf("%d HTTP monitors started.\n", StatsGlobals.TotalHTTPMonitoringPoints)
	}
	StatsGlobals.TotalMonitoringPoints = len(cfg.StreamsHTTP) + len(cfg.StreamsHLS) + len(cfg.StreamsHDS)
}

func GroupBox(cfg *Config, ctl *bcast.Group, group string, streamType StreamType, taskq chan *Task, debugvars *expvar.Map) {
}

// Container incapsulates data and logic about single stream checks.
func StreamBox(cfg *Config, ctl *bcast.Group, stream Stream, streamType StreamType, taskq chan *Task, debugvars *expvar.Map) {
	var addSleepToBrokenStream time.Duration
	var min, max int
	var command Command
	var online bool = false

	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Stream %s trace: %s\n", stream.Name, r)
		}
	}()

	task := &Task{Stream: stream, ReplyTo: make(chan Result)}
	switch streamType {
	case HTTP:
		task.ReadBody = false
	case HLS, HDS:
		task.ReadBody = true
	default:
		task.ReadBody = false
	}
	errhistory := make(map[ErrHistoryKey]uint)     // duplicates ErrHistory from stats
	errtotals := make(map[ErrTotalHistoryKey]uint) // duplicates ErrTotalHistory from stats
	ctlrcv := ctl.Join()

	for {
		select {
		case recv := <-*ctlrcv.In:
			command = recv.(Command)
			switch command {
			case START:
				online = true
			case STOP:
				online = false
			case RELOAD:
			default:
			}
		default:
			if !online {
				time.Sleep(50 * time.Millisecond)
				continue
			}
			taskq <- task
			debugvars.Add("requested-tasks", 1)
			result := <-task.ReplyTo
			go SaveStats(stream, &result)
			/*			switch result.Meta.(type) {
						case MetaHLS:
							// сформировать таск для чанклистов
							// taskq <- task
							// result := <-task.ReplyTo
						case MetaHDS:
						default:
						}
			*/
			curhour := result.Started.Format("06010215")
			prevhour := result.Started.Add(-1 * time.Hour).Format("06010215")
			errhistory[ErrHistoryKey{Curhour: curhour, ErrType: result.ErrType}]++
			errtotals[ErrTotalHistoryKey{Curhour: curhour}]++
			max = int(cfg.Params.CheckBrokenTime)
			min = int(cfg.Params.CheckBrokenTime / 4. * 3.)

			switch {
			// too much repeatable errors per hour:
			case errtotals[ErrTotalHistoryKey{Curhour: curhour}] > 6:
			case errtotals[ErrTotalHistoryKey{Curhour: prevhour}] > 6:
				addSleepToBrokenStream = time.Duration(rand.Intn(max-min)+min) * time.Second
			// permanent error, not a timeout:
			case result.ErrType > CRITICAL_LEVEL:
				addSleepToBrokenStream = time.Duration(rand.Intn(max-min)+min) * time.Second
			// works ok:
			case result.ErrType == SUCCESS:
			default:
				addSleepToBrokenStream = 0
			}
			result.TotalErrs = errtotals[ErrTotalHistoryKey{Curhour: curhour}]

			go SaveStats(stream, &result)

			if result.ErrType >= WARNING_LEVEL {
				go Log(ERROR, stream, result)
			} else {
				if result.Elapsed >= cfg.Params.VerySlowWarningTimeout*time.Second {
					result.ErrType = VERYSLOW
					go Log(WARNING, stream, result)
				} else if result.Elapsed >= cfg.Params.SlowWarningTimeout*time.Second {
					result.ErrType = SLOW
					go Log(WARNING, stream, result)
				}
			}
			max = int(cfg.Params.CheckRepeatTime)
			min = int(cfg.Params.CheckRepeatTime / 4. * 3.)
			time.Sleep(time.Duration(rand.Intn(max-min)+min)*time.Millisecond + addSleepToBrokenStream)
		}
	}
}

// Check & report internet availability. Stop all probers when sample internet resources not available.
// Refs to config option ``samples``.
func Heartbeat(cfg *Config, ctl *bcast.Group) {
	var previous bool

	ctlsnr := ctl.Join()

	for {
		for _, uri := range cfg.Samples {
			client := NewTimeoutClient(cfg.Params.ConnectTimeout*time.Second, cfg.Params.RWTimeout*time.Second)
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
				ctlsnr.Send(START)
				fmt.Println("Internet Ok. Monitoring started.")
			} else {
				ctlsnr.Send(STOP)
				fmt.Println("Internet not available. Monitoring stopped.")
			}
		}
		previous = StatsGlobals.MonitoringState
		time.Sleep(12 * time.Second)
	}
}

// Probe HTTP without additional protocola parsing.
// SaveStats timeouts and bad statuses.
func SimpleProber(cfg *Config, ctl *bcast.Group, tasks chan *Task, debugvars *expvar.Map) {
	for {
		task := <-tasks
		result := ExecHTTP(cfg, task)
		task.ReplyTo <- *result
		debugvars.Add("http-tasks-done", 1)
		time.Sleep(cfg.Params.TimeBetweenTasks * time.Millisecond)
	}
}

// HTTP Live Streaming support.
// Parse and probe M3U8 playlists (multi- and single bitrate)
// and report time statistics and errors
func CupertinoProber(cfg *Config, ctl *bcast.Group, tasks chan *Task, debugvars *expvar.Map) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("trace dumped in Cupertino prober:", r)
		}
	}()

	for {
		task := <-tasks
		result := ExecHTTP(cfg, task)
		if result.ErrType > ERROR_LEVEL && result.HTTPCode < 400 && result.ContentLength > 0 {
			playlist, listType, err := m3u8.Decode(result.Body, false)
			if err != nil {
				result.ErrType = BADFORMAT
			} else {
				switch listType {
				case m3u8.MASTER:
					/*					m := playlist.(*m3u8.MasterPlaylist)
										for _, variant := range m.Variants {
											task := &Task{Stream: Stream{variant.URI, HLS, task.Name, task.Group}, ReplyTo: make(chan Result)}
											fmt.Printf("%v\n", task)
											//tasks <- task
											// XXX
										}
					*/
				case m3u8.MEDIA:
					p := playlist.(*m3u8.MediaPlaylist)
					p.Encode().String()
				default:
					result.ErrType = BADFORMAT
				}
			}
		}
		task.ReplyTo <- *result
		debugvars.Add("hls-tasks-done", 1)
		time.Sleep(cfg.Params.TimeBetweenTasks * time.Millisecond)
	}
}

// HTTP Dynamic Streaming prober.
// Parse and probe F4M playlists and report time statistics and errors.
func SanjoseProber(cfg *Config, ctl *bcast.Group, tasks chan *Task, debugvars *expvar.Map) {
	for {
		task := <-tasks
		result := ExecHTTP(cfg, task)
		task.ReplyTo <- *result
		debugvars.Add("hds-tasks-done", 1)
		time.Sleep(cfg.Params.TimeBetweenTasks * time.Millisecond)
	}
}

// Parse and probe media chunk
// and report time statistics and errors
func MediaProber(cfg *Config, ctl *bcast.Group, streamType StreamType, taskq chan *Task, debugvars *expvar.Map) {
	for {
		time.Sleep(20 * time.Second)
	}
}

// Helper. Execute stream check task and return result with check status.
func ExecHTTP(cfg *Config, task *Task) *Result {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("trace dumped in ExecHTTP:", r)
		}
	}()

	result := &Result{Started: time.Now(), Elapsed: 0 * time.Second}
	if !strings.HasPrefix(task.URI, "http://") && !strings.HasPrefix(task.URI, "https://") {
		result.ErrType = BADURI
		result.HTTPCode = 0
		result.HTTPStatus = ""
		result.ContentLength = -1
		return result
	}
	client := NewTimeoutClient(cfg.Params.ConnectTimeout*time.Second, cfg.Params.RWTimeout*time.Second)
	req, err := http.NewRequest(cfg.Params.MethodHTTP, task.URI, nil)
	if err != nil {
		fmt.Println(err)
		result.ErrType = BADURI
		result.HTTPCode = 0
		result.HTTPStatus = ""
		result.ContentLength = -1
		return result
	}
	req.Header.Set("User-Agent", SURFER)
	resp, err := client.Do(req)
	result.Elapsed = time.Since(result.Started)
	if err != nil {
		if result.Elapsed >= cfg.Params.ConnectTimeout*time.Second {
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
