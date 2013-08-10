// The code keeps streams statistics and program internal statistics.
// Statistics output to files and to JSON HTTP API.
package main

import (
	"sync"
	"time"
)

var (
	logq  chan LogMessage
	statq chan Stats
)

type Stats struct {
	Source    string
	Operation string
	Started   time.Time
	Elapsed   time.Duration
}

var reports chan StreamStats

// Streams statistics
var ReportedStreams = struct {
	sync.RWMutex
	data map[string]map[string]StreamStats
}{data: make(map[string]map[string]StreamStats)}

// Elder
func StatKeeper(cfg *Config) {
	//	statq := make(chan Stats, 1024)
	reports = make(chan StreamStats, 1024)
	for {
		select {
		//case stat := <-statq:
		// pass
		case state := <-reports:
			ReportedStreams.Lock()
			if _, err := ReportedStreams.data[state.Stream.Group]; !err {
				ReportedStreams.data[state.Stream.Group] = make(map[string]StreamStats)
			}
			ReportedStreams.data[state.Stream.Group][state.Stream.Name] = state
			ReportedStreams.Unlock()
		}
	}
}

// Wrapper to get reports from streams
func Report(stream Stream, last *TaskResult) {
	reports <- StreamStats{Stream: stream, Last: *last}
}
