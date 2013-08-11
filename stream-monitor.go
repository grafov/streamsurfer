package main

import (
	"bufio"
	"fmt"
	"github.com/grafov/m3u8"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

// Control monitoring of a single stream
func StreamMonitor(cfg *Config) {
	var i uint

	sampletasks := make(chan *Task, 2)
	httptasks := make(chan *Task, len(cfg.StreamsHTTP)*4)
	hlstasks := make(chan *Task, len(cfg.StreamsHLS)*4)
	chunktasks := make(chan *Task, (cfg.Params.ProbersHLS+cfg.Params.ProbersHTTP)*8) // TODO тут не задачи, другой тип

	go HealthCheck(sampletasks)
	for i = 0; i < cfg.Params.ProbersHLS; i++ {
		go CupertinoProber(cfg, hlstasks)
	}
	fmt.Printf("%d HLS probers started.\n", cfg.Params.ProbersHLS)
	for i = 0; i < cfg.Params.ProbersHLS+cfg.Params.ProbersHTTP; i++ {
		go MediaProber(cfg, chunktasks)
	}
	fmt.Printf("%d media probers started.\n", cfg.Params.ProbersHLS+cfg.Params.ProbersHTTP)
	for i = 0; i < cfg.Params.ProbersHTTP; i++ {
		go SimpleProber(cfg, httptasks)
	}
	fmt.Printf("%d HTTP monitors started.\n", cfg.Params.ProbersHTTP)
	for _, stream := range cfg.StreamsHLS {
		go StreamBox(cfg, stream, HLS, hlstasks)
	}
	fmt.Printf("%d HLS monitors started.\n", len(cfg.StreamsHLS))
	/*	for _, stream := range cfg.StreamsHTTP {
			go GroupBox(cfg, stream, HTTP, httptasks)
		}
		fmt.Printf("%d HTTP monitors started.\n", len(cfg.StreamsHTTP))
	*/
}

func GroupBox(cfg *Config, stream Stream, streamType StreamType, taskq chan *Task) {
}

// Container incapsulates data and logic about single stream checks.
func StreamBox(cfg *Config, stream Stream, streamType StreamType, taskq chan *Task) {
	var addSleepToBrokenStream time.Duration

	task := &Task{Stream: stream, ReplyTo: make(chan TaskResult)}
	errhistory := make(map[ErrHistoryKey]uint)     // duplicates ErrHistory from stats
	errtotals := make(map[ErrTotalHistoryKey]uint) // duplicates ErrTotalHistory from stats
	go Report(stream, &TaskResult{})
	for {
		taskq <- task
		result := <-task.ReplyTo
		curhour := result.Started.Format("06010215")
		prevhour := result.Started.Add(-1 * time.Hour).Format("06010215")
		errhistory[ErrHistoryKey{Curhour: curhour, ErrType: result.ErrType}]++
		errtotals[ErrTotalHistoryKey{Curhour: curhour}]++
		if errtotals[ErrTotalHistoryKey{Curhour: curhour}] > 6 || errtotals[ErrTotalHistoryKey{Curhour: prevhour}] > 6 {
			addSleepToBrokenStream = 2 * time.Minute
		} else {
			addSleepToBrokenStream = 0
		}
		result.TotalErrs = errtotals[ErrTotalHistoryKey{Curhour: curhour}]
		go Report(stream, &result)
		if result.ErrType != SUCCESS {
			go Log(ERROR, stream, result)
			time.Sleep(time.Duration(rand.Intn(6)+1)*time.Second + addSleepToBrokenStream) // TODO config
		} else {
			if result.Elapsed >= cfg.Params.VerySlowWarningTimeout*time.Second {
				result.ErrType = VERYSLOW
				go Log(WARNING, stream, result)
			} else if result.Elapsed >= cfg.Params.SlowWarningTimeout*time.Second {
				result.ErrType = SLOW
				go Log(WARNING, stream, result)
			}
			time.Sleep(time.Duration(rand.Intn(20)+1)*time.Second + addSleepToBrokenStream) // TODO config
		}
	}
}

// Check & report internet availability
func HealthCheck(taskq chan *Task) {

}

// Probe HTTP without additional protocola parsing.
// Report timeouts and bad statuses.
func SimpleProber(cfg *Config, tasks chan *Task) {
	for {
		task := <-tasks
		result := doTask(cfg, task)
		task.ReplyTo <- *result
		time.Sleep(cfg.Params.TimeBetweenTasks * time.Millisecond)
	}
}

// Parse and probe M3U8 playlists (multi- and single bitrate)
// and report time statistics and errors
func CupertinoProber(cfg *Config, tasks chan *Task) {
	for {
		task := <-tasks
		result := doTask(cfg, task)
		if result.ErrType != CTIMEOUT && result.HTTPCode < 400 {
			verifyHLS(cfg, task, result)
			// вернуть variants и по ним передать задачи в канал CupertinoProber
		}
		task.ReplyTo <- *result
		time.Sleep(cfg.Params.TimeBetweenTasks * time.Millisecond)
	}
}

// Parse and probe media chunk
// and report time statistics and errors
func MediaProber(cfg *Config, taskq chan *Task) {

	for {
		time.Sleep(20 * time.Second)
	}

}

// Helper. Execute stream check task and return result with check status.
func doTask(cfg *Config, task *Task) *TaskResult {
	result := &TaskResult{Started: time.Now(), Elapsed: 0 * time.Second}
	if !strings.HasPrefix(task.URI, "http://") && !strings.HasPrefix(task.URI, "https://") {
		result.ErrType = BADURI
		result.HTTPCode = 0
		result.HTTPStatus = ""
		result.ContentLength = -1
		return result
	}
	client := NewTimeoutClient(cfg.Params.ConnectTimeout*time.Second, cfg.Params.RWTimeout*time.Second)
	req, err := http.NewRequest("GET", task.URI, nil) // TODO в конфиге выбирать метод проверки
	if err != nil {
		result.ErrType = BADURI
		result.HTTPCode = 0
		result.HTTPStatus = ""
		result.ContentLength = -1
		return result
	}
	resp, err := client.Do(req)
	result.Elapsed = time.Since(result.Started)
	if err != nil {
		result.ErrType = UNKNOWN
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
	result.Body = resp.Body // TODO read?
	return result
}

// Helper. Verify HLS specific things.
func verifyHLS(cfg *Config, task *Task, result *TaskResult) {
	defer func() {
		if r := recover(); r != nil {
			//fmt.Println("trace dumped:", r)
			result.ErrType = RTIMEOUT
		}
	}()

	playlist, listType, err := m3u8.Decode(bufio.NewReader(result.Body), false)
	if err != nil {
		result.ErrType = BADFORMAT
	} else {
		switch listType {
		case m3u8.MASTER:
			m := playlist.(*m3u8.MasterPlaylist)
			m.Encode().String()
		case m3u8.MEDIA:
			p := playlist.(*m3u8.MediaPlaylist)
			p.Encode().String()
		default:
			result.ErrType = BADFORMAT
		}
	}
}

// Text representation of stream error
func StreamErrText(err ErrType) string {
	switch err {
	case SUCCESS:
		return "success"
	case SLOW:
		return "slow response"
	case VERYSLOW:
		return "very slow response"
	case BADSTATUS:
		return "bad status"
	case BADURI:
		return "bad URI"
	case LISTEMPTY: // HLS specific
		return "list empty"
	case BADFORMAT: // HLS specific
		return "bad format"
	case RTIMEOUT:
		return "timeout on read"
	case CTIMEOUT:
		return "connection timeout"
	case HLSPARSER:
		return "HLS parser" // debug
	default:
		return "unknown"
	}
}
