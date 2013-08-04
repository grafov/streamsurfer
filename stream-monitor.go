package main

import (
	"fmt"
	"github.com/grafov/m3u8"
	"net/http"
	"time"
)

// Kinds of streams
const (
	SAMPLE StreamType = iota // internet resources for monitor self checks
	HTTP                     //
	HLS
)

// Error codes
const (
	SUCCESS ErrType = iota
	BADURI
	TIMEOUT
	BADSTATUS
	BADFORMAT
	LISTEMPTY
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
	taskq := make(chan *Task, 1024)
	go HealthCheck(taskq)
	go SimpleProber(taskq)
	go CupertinoProber(taskq)
	go MediaProber(taskq)
	for _, stream := range cfg.StreamsHTTP {
		go Stream(stream, HTTP, taskq)
	}
	for _, stream := range cfg.StreamsHLS {
		go Stream(stream, HLS, taskq)
	}
}

// Container for keeping info about each stream checks
func Stream(uri string, streamType StreamType, taskq chan *Task) {
	task := &Task{URI: uri, Type: streamType, ReplyTo: make(chan Result)}
	for {
		taskq <- task
		result := <-task.ReplyTo
		fmt.Printf("%+v\n", result)
		time.Sleep(3 * time.Second)
	}
}

// Check & report internet availability
func HealthCheck(taskq chan *Task) {

}

// Probe HTTP without additional protocola parsing.
// Report timeouts and bad statuses.
func SimpleProber(taskq chan *Task) {
	for {
		task := <-taskq
		client := NewTimeoutClient(1*time.Second, 2*time.Second)
		result := &Result{Started: time.Now(), Elapsed: 0 * time.Second}
		req, err := http.NewRequest("HEAD", task.URI, nil) // TODO в конфиге выбирать метод проверки
		if err != nil {
			result.Type = BADURI
			task.ReplyTo <- *result
			result.HTTPCode = 0
			result.HTTPStatus = ""
			result.ContentLength = -1
			continue
		}
		resp, err := client.Do(req)
		result.Elapsed = time.Since(result.Started)
		if err != nil {
			result.Type = TIMEOUT
			result.HTTPCode = 0
			result.HTTPStatus = ""
			result.ContentLength = -1
			continue
		}
		result.HTTPCode = resp.StatusCode
		result.HTTPStatus = resp.Status
		result.ContentLength = resp.ContentLength
		task.ReplyTo <- *result
	}
}

// Parse and probe M3U8 playlists (multi- and single bitrate)
// and report time statistics and errors
func CupertinoProber(taskq chan *Task) {

	m3u8.NewMasterPlaylist()
	for {
		time.Sleep(20 * time.Second)
	}

}

// Parse and probe media chunk
// and report time statistics and errors
func MediaProber(taskq chan *Task) {

	for {
		time.Sleep(20 * time.Second)
	}

}
