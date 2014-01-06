// The code keeps streams statistics and program internal statistics.
// Statistics output to files and to JSON HTTP API.
package main

import (
	"expvar"
	"log"
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
	var oldestStoredTime time.Time
	var debugStatsCount = expvar.NewInt("stats-count")

	statq = make(chan StreamStats, 8192) // receive stats
	stats := make(map[StatKey]Result)    // global statistics with timestamps aligned to minutes

	// storage maintainance period
	timer := time.Tick(4 * time.Second) // TODO 2 мин

	for {
		select {
		case state := <-statq: // receive new statitics data for saving
			alignedToMinute := state.Last.Started.Truncate(1 * time.Minute)
			stats[StatKey{state.Stream.Type, state.Stream.Group, state.Stream.Name, alignedToMinute}] = state.Last
			//log.Printf("stored %+v", StatKey{state.Stream.Type, state.Stream.Group, state.Stream.Name, alignedToMinute})
			if oldestStoredTime.IsZero() {
				oldestStoredTime = alignedToMinute
			} else if oldestStoredTime.After(alignedToMinute) {
				oldestStoredTime = alignedToMinute
			}
			debugStatsCount.Add(1)

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
			log.Println("Cleanup routine entered. Cache len: ", len(stats), oldestStoredTime)
			if len(cfg.Groups) == 0 || len(stats) == 0 {
				goto cleanupExit
			}
			//log.Printf("%v\n", stats) XXX
			for group, streams := range cfg.Groups {
				for _, stream := range streams {
					for min := oldestStoredTime; min.Before(oldestStoredTime.Add(60 * time.Minute)); min.Add(1 * time.Minute) {
						//log.Printf("00--> %+v\n", stats[StatKey{group.Type, group.Name, stream, oldestStoredTime}])
						if _, ok := stats[StatKey{group.Type, group.Name, stream, min}]; ok {
							log.Printf("%+v deleted.\n", StatKey{group.Type, group.Name, stream, min})
							delete(stats, StatKey{group.Type, group.Name, stream, min})
						}
					}
				}
			}
			oldestStoredTime = oldestStoredTime.Add(1 * time.Minute)
		cleanupExit:
			log.Println("Cleanup routine exited. Cache len: ", len(stats), oldestStoredTime)
		}
	}

}

// Put result of probe task to statistics.
func SaveStats(stream Stream, last *Result) {
	statq <- StreamStats{Stream: stream, Last: *last}
}

// Get statistics of probe tasks for the period.
func LoadStats(group Group, stream Stream, from time.Time, to time.Time) map[StatKey]Result {
	return nil
}
