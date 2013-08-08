// Web reports generator
package main

import (
	"github.com/hoisie/mustache"
	"sync"
	"time"
)

type StreamState struct {
	Stream Stream
	Result TaskResult
}

var reports chan StreamState
var ReportedStreams = struct {
	sync.RWMutex
	data map[string]map[string]StreamState
}{data: make(map[string]map[string]StreamState)}

func ReportKeeper(cfg *Config) {
	reports = make(chan StreamState, 1024)
	select {
	case stream := <-reports:
		ReportedStreams.Lock()
		ReportedStreams.data[stream.Stream.Group][stream.Stream.Name] = stream
		ReportedStreams.Unlock()
	default:
		time.Sleep(100 * time.Millisecond)
	}
}

// Wrapper to send report to ReportKeeper
func Report(stream Stream, result *TaskResult) {
	reports <- StreamState{Stream: stream, Result: *result}
}

func ReportGroupErrors(vars map[string]string) []byte {
	ReportedStreams.RLock()
	/*	for key, value := range ReportedStreams.data {

												   	}*/
	ReportedStreams.Unlock()
	page := mustache.Render(ReportGroupErrorsTemplate, map[string]string{"c": "world"})
	return []byte(page)
}
