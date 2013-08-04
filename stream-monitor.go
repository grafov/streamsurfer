package main

import (
	"bufio"
	"fmt"
	"github.com/grafov/m3u8"
	"net/http"
	"strings"
	"time"
)

// Kinds of streams
const (
	SAMPLE StreamType = iota // internet resources for monitor self checks
	HTTP                     //
	HLS
)

// Error codes (ordered by errors importance).
// If several errors detected then only one with the heaviest weight reported.
const (
	SUCCESS ErrType = iota
	BADSTATUS
	BADURI
	LISTEMPTY // HLS specific
	BADFORMAT // HLS specific
	TIMEOUT
)

type StreamType uint // Type of checked streams
type ErrType uint

// Stream checking task
type Task struct {
	URI  string
	Type StreamType
	//	Name    string
	ReplyTo chan Result
}

// Stream checking result
type Result struct {
	Type          ErrType
	HTTPCode      int    // HTTP status code
	HTTPStatus    string // HTTP status string
	ContentLength int64
	Started       time.Time
	Elapsed       time.Duration
}

// Control monitoring of a single stream
func StreamMonitor(cfg *Config) {
	var i uint

	sampletasks := make(chan *Task, 2)
	httptasks := make(chan *Task, 1024)
	hlstasks := make(chan *Task, 1024)
	chunktasks := make(chan *Task, 1024) // TODO тут не задачи, другой тип

	go HealthCheck(sampletasks)
	for i = 0; i < cfg.Params.ProbersHTTP; i++ {
		go SimpleProber(cfg, httptasks)
	}
	for i = 0; i < cfg.Params.ProbersHLS; i++ {
		go CupertinoProber(cfg, hlstasks)
	}
	for i = 0; i < cfg.Params.ProbersHLS; i++ {
		go MediaProber(cfg, chunktasks)
	}
	for _, stream := range cfg.StreamsHTTP {
		go Stream(stream, HTTP, httptasks)
	}
	for _, stream := range cfg.StreamsHLS {
		go Stream(stream, HLS, hlstasks)
	}
}

// Container for keeping info about each stream checks
func Stream(uri string, streamType StreamType, taskq chan *Task) {
	task := &Task{URI: uri, Type: streamType, ReplyTo: make(chan Result)}
	for {
		taskq <- task
		result := <-task.ReplyTo
		fmt.Printf("%v %s\n", result, uri)
		time.Sleep(3 * time.Second)
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
		_, result := doTask(cfg, task)
		task.ReplyTo <- *result
		time.Sleep(cfg.Params.TimeBetweenTasks * time.Millisecond)
	}
}

// Parse and probe M3U8 playlists (multi- and single bitrate)
// and report time statistics and errors
func CupertinoProber(cfg *Config, tasks chan *Task) {
	for {
		task := <-tasks
		response, result := doTask(cfg, task)
		if result.Type != TIMEOUT {
			verifyHLS(cfg, task, response, result)
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
func doTask(cfg *Config, task *Task) (*http.Response, *Result) {
	result := &Result{Started: time.Now(), Elapsed: 0 * time.Second}
	if !strings.HasPrefix(task.URI, "http://") && !strings.HasPrefix(task.URI, "https://") {
		result.Type = BADURI
		result.HTTPCode = 0
		result.HTTPStatus = ""
		result.ContentLength = -1
		return nil, result
	}
	client := NewTimeoutClient(cfg.Params.ConnectTimeout*time.Second, cfg.Params.RWTimeout*time.Second)
	req, err := http.NewRequest("GET", task.URI, nil) // TODO в конфиге выбирать метод проверки
	if err != nil {
		result.Type = BADURI
		result.HTTPCode = 0
		result.HTTPStatus = ""
		result.ContentLength = -1
		return nil, result
	}
	resp, err := client.Do(req)
	result.Elapsed = time.Since(result.Started)
	if err != nil {
		result.Type = TIMEOUT
		result.HTTPCode = 0
		result.HTTPStatus = ""
		result.ContentLength = -1
		return nil, result
	}
	result.HTTPCode = resp.StatusCode
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		result.Type = BADSTATUS
	}
	result.HTTPStatus = resp.Status
	result.ContentLength = resp.ContentLength
	return resp, result
}

// Helper. Verify HLS specific things.
func verifyHLS(cfg *Config, task *Task, response *http.Response, result *Result) {
	playlist, listType, err := m3u8.Decode(bufio.NewReader(response.Body), false)
	if err != nil {
		fmt.Println(err)
	}
	switch listType {
	case m3u8.MASTER:
		m := playlist.(*m3u8.MasterPlaylist)
		fmt.Printf("%+v", m.Variants[0])
		fmt.Println(m.Encode().String())
	case m3u8.MEDIA:
		p := playlist.(*m3u8.MediaPlaylist)
		fmt.Println(p.Encode().String())
	default:
		result.Type = BADFORMAT
	}
}
