// The code keeps streams statistics and program internal statistics.
// Statistics output to files and to JSON HTTP API.
package main

import (
	"errors"
	"expvar"
	"sync"
	"time"
)

var (
	logq chan LogMessage
	//statIn chan Stats
)

var StatsGlobals = struct {
	TotalMonitoringPoints     int
	TotalHTTPMonitoringPoints int
	TotalWVMonitoringPoints   int
	TotalHLSMonitoringPoints  int
	TotalHDSMonitoringPoints  int
	MonitoringState           bool // is inet available?
}{}

// Обмен данными с хранителем статистики.
var (
	statIn  chan StatInQuery
	statOut chan StatOutQuery
)

// // Streams statistics
// var ReportedStreams = struct {
// 	sync.RWMutex
// 	data map[string]map[string]StreamStats // map [group][stream]stream-state
// }{data: make(map[string]map[string]StreamStats)}

var ErrHistory = struct {
	sync.RWMutex
	count map[ErrHistoryKey]uint
}{count: make(map[ErrHistoryKey]uint)}

var ErrTotalHistory = struct {
	sync.RWMutex
	count map[ErrTotalHistoryKey]uint
}{count: make(map[ErrTotalHistoryKey]uint)}

type Incident struct {
	Id uint64
	StatKey
	Error *Result
}

/* Структуры данных для статистики:

map текущих состояний:
 StatKey{Type, Group, Name}
  Result

map результатов по времени:
 StatKey{Type, Group, Name}
  time.Time
    Result
*/

// Elder
func StatKeeper() {
	var debugStatsCount = expvar.NewInt("stats-count")

	statIn = make(chan StatInQuery, 8192)           // receive stats
	statOut = make(chan StatOutQuery, 4)            // send stats
	stats := make(map[StatKey]map[time.Time]Result) // global statistics with timestamps aligned to minutes
	curstats := make(map[StatKey]Result)

	// storage maintainance period
	timer := time.Tick(4 * time.Second) // TODO 2 мин

	for {
		select {
		case state := <-statIn: // receive new statitics data for saving
			curstats[StatKey{state.Stream.Group, state.Stream.Name}] = state.Last
			if _, ok := stats[StatKey{state.Stream.Group, state.Stream.Name}]; !ok {
				stats[StatKey{state.Stream.Group, state.Stream.Name}] = make(map[time.Time]Result)
			}
			stats[StatKey{state.Stream.Group, state.Stream.Name}][state.Last.Started] = state.Last
			debugStatsCount.Add(1)

			/* TODO
			Последние 30 минут лежат в памяти, остальное в редисе. Если редиса нет, глубокая статистика недоступна.

			*/

			// Дальше устаревшая статистика, надо выпилить
			// Last check results for all streams
			// ReportedStreams.Lock()
			// if _, exists := ReportedStreams.data[state.Stream.Group]; !exists {
			// 	ReportedStreams.data[state.Stream.Group] = make(map[string]StreamStats)
			// }
			// ReportedStreams.data[state.Stream.Group][state.Stream.Name] = state
			// ReportedStreams.Unlock()
			// // Per hour statistics for all streams
			// if state.Last.ErrType >= WARNING_LEVEL {
			// 	ErrHistory.Lock()
			// 	curhour := state.Last.Started.Format("06010215")
			// 	ErrHistory.count[ErrHistoryKey{curhour, state.Last.ErrType, state.Stream.Group, state.Stream.Name, state.Stream.URI}]++
			// 	ErrHistory.Unlock()
			// 	ErrTotalHistory.Lock()
			// 	ErrTotalHistory.count[ErrTotalHistoryKey{curhour, state.Stream.Group, state.Stream.Name}]++
			// 	ErrTotalHistory.Unlock()
			// }

		case key := <-statOut:
			if val, ok := curstats[key.Key]; ok {
				key.ReplyTo <- &val
			} else {
				key.ReplyTo <- nil
			}

		case <-timer: // cleanup old history entries
			// log.Println("Cleanup routine entered. Cache len: ", len(stats), oldestStoredTime)
			if len(cfg.GroupParams) == 0 || len(stats) == 0 {
				goto cleanupExit
			}

			for key, streamstats := range stats {
				for moment := range streamstats {
					if moment.Before(time.Now().Add(-3 * time.Hour)) {
						delete(stats[key], moment)
					}
				}
			}

		cleanupExit:
		}
	}

}

// Put result of probe task to statistics.
func SaveStats(stream Stream, last *Result) {
	statIn <- StatInQuery{Stream: stream, Last: *last}
}

// Получить состояние по последней проверке.
func LoadLastStats(group, stream string) (*Result, error) {
	result := make(chan *Result)
	statOut <- StatOutQuery{Key: StatKey{Group: group, Name: stream}, ReplyTo: result}
	data := <-result
	if data != nil {
		return data, nil
	} else {
		return nil, errors.New("result not found")
	}
}

// Получить статистику по каналу за всё время наблюдения.
func LoadHistoryStats(typ StreamType, group Group, stream Stream) *[]Result {
	return nil
}
