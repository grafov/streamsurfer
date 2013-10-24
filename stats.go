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

var StatsGlobals = struct {
	TotalMonitoringPoints     int
	TotalHTTPMonitoringPoints int
	TotalHLSMonitoringPoints  int
	TotalHDSMonitoringPoints  int
	MonitoringState           bool // is inet available?
}{}

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
	data map[string]map[string]StreamStats // map [group][stream]stream-state
}{data: make(map[string]map[string]StreamStats)}

var ErrHistory = struct {
	sync.RWMutex
	count map[ErrHistoryKey]uint
}{count: make(map[ErrHistoryKey]uint)}

var ErrTotalHistory = struct {
	sync.RWMutex
	count map[ErrTotalHistoryKey]uint
}{count: make(map[ErrTotalHistoryKey]uint)}

// Elder
func StatKeeper(cfg *Config) {
	//	statq := make(chan Stats, 1024)
	reports = make(chan StreamStats, 4096)
	for {
		select {
		//case stat := <-statq:
		// pass
		case state := <-reports:
			// Last check results for all streams
			ReportedStreams.Lock()
			if _, exists := ReportedStreams.data[state.Stream.Group]; !exists {
				ReportedStreams.data[state.Stream.Group] = make(map[string]StreamStats)
			}
			ReportedStreams.data[state.Stream.Group][state.Stream.Name] = state
			ReportedStreams.Unlock()
			// Per hour statistics for all streams
			if state.Last.ErrType >= WARNING_LEVEL {
				ErrHistory.Lock()
				curhour := state.Last.Started.Format("06010215")
				ErrHistory.count[ErrHistoryKey{curhour, state.Last.ErrType, state.Stream.Group, state.Stream.Name, state.Stream.URI}]++
				ErrHistory.Unlock()
				ErrTotalHistory.Lock()
				ErrTotalHistory.count[ErrTotalHistoryKey{curhour, state.Stream.Group, state.Stream.Name}]++
				ErrTotalHistory.Unlock()
			}
		}
	}
}

// Wrapper to get reports from streams
func Report(stream Stream, last *Result) {
	reports <- StreamStats{Stream: stream, Last: *last}
}
