// The code keeps streams statistics and program internal statistics.
// Statistics output to files and to JSON HTTP API.
package main

import (
	"expvar"
	"sync"
	"time"
)

var (
	logq chan LogMessage
	//statq chan Stats
)

var StatsGlobals = struct {
	TotalMonitoringPoints     int
	TotalHTTPMonitoringPoints int
	TotalHLSMonitoringPoints  int
	TotalHDSMonitoringPoints  int
	MonitoringState           bool // is inet available?
}{}

var statq chan StreamStats

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
	var storedStats = expvar.NewInt("stored-stats")
	var oldestStoredTime time.Time

	statq = make(chan StreamStats, 8192) // receive stats
	stats := make(map[StatKey]Result)    // global statistics with timestamps aligned to minutes

	timer := make(chan time.Time)
	go func() {
		// storage maintainance period
		time.Tick(1 * time.Minute)
	}()

	for {
		select {
		case state := <-statq: // receive new statitics data for saving
			alignedToMinute := state.Last.Started.Truncate(time.Minute)
			stats[StatKey{state.Stream, alignedToMinute}] = state.Last
			if oldestStoredTime.After(alignedToMinute) {
				oldestStoredTime = alignedToMinute
			}
			storedStats.Add(1)

			// Дальше устаревшая статистика, надо выпилить
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

		case <-timer: // cleanup old history entries

		}
	}
}

// Put result of probe task to statistics.
func SaveStats(stream Stream, last *Result) {
	statq <- StreamStats{Stream: stream, Last: *last}
}
