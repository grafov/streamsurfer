// The code keeps streams statistics and program internal statistics.
// Statistics output to files and to JSON HTTP API.
package main

import (
	"errors"
	"expvar"
	//"github.com/garyburd/redigo/redis"
	"fmt"
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
	var stats map[StatKey][]Result = make(map[StatKey][]Result)

	statIn = make(chan StatInQuery, 8192) // receive stats
	statOut = make(chan StatOutQuery, 4)  // send stats

	// storage maintainance period
	timer := time.Tick(12 * time.Second)

	for {
		select {
		case state := <-statIn: // receive new statitics data for saving
			stats[StatKey{state.Stream.Group, state.Stream.Name}] = append(stats[StatKey{state.Stream.Group, state.Stream.Name}], state.Last)
			debugStatsCount.Add(1)
			// if _, ok := stats[StatKey{state.Stream.Group, state.Stream.Name}]; !ok {
			// 	//stats[StatKey{state.Stream.Group, state.Stream.Name}] = list.New()
			// 	stats[StatKey{state.Stream.Group, state.Stream.Name}] = state.Last
			// } else {
			// 	// stats[StatKey{state.Stream.Group, state.Stream.Name}].PushFront(state.Last)
			// 	stats[StatKey{state.Stream.Group, state.Stream.Name}] = append(stats[StatKey{state.Stream.Group, state.Stream.Name}], state.Last)
			// 	debugStatsCount.Add(1)
			// }
			/* TODO
			Последние 30 минут лежат в памяти, остальное в редисе. Если редиса нет, глубокая статистика недоступна.

			*/
			fmt.Printf("%+v\n", state.Last.SubResults)
		case key := <-statOut:
			if val, ok := stats[key.Key]; ok {
				key.ReplyTo <- val
			} else {
				key.ReplyTo <- nil
			}
		case <-timer: // cleanup old history entries
			// log.Println("Cleanup routine entered. Cache len: ", len(stats), oldestStoredTime)
			if len(cfg.GroupParams) == 0 || len(stats) == 0 {
				goto cleanupExit
			}

			for _, streamstats := range stats {
				// for e := streamstats.Front(); e != nil; e = e.Next() {
				// 	if time.Since(e.Value.(Result).Started) > 3*time.Minute { // XXX set 30 min
				// 		streamstats.Remove(e)
				// 	}
				// }
				for idx, val := range streamstats {
					if time.Since(val.Started) > 3*time.Minute {
						streamstats = append(streamstats[0:idx], streamstats[idx:]...)
					}
				}
			}
		cleanupExit:
		}
	}
}

// Put result of probe task to statistics.
func SaveStats(stream Stream, last Result) {
	statIn <- StatInQuery{Stream: stream, Last: last}
}

// Получить состояние по последней проверке.
func LoadLastStats(group, stream string) (*Result, error) {
	result := make(chan []Result)
	statOut <- StatOutQuery{Key: StatKey{Group: group, Name: stream}, ReplyTo: result}
	data := <-result
	if data != nil {
		return &data[len(data)-1], nil
	} else {
		return nil, errors.New("result not found")
	}
}

// Получить статистику по каналу за всё время наблюдения.
func LoadHistoryStats(group, stream string) (*[]Result, error) {
	result := make(chan []Result)
	statOut <- StatOutQuery{Key: StatKey{Group: group, Name: stream}, ReplyTo: result}
	data := <-result
	if data != nil {
		return &data, nil
	} else {
		return nil, errors.New("result not found")
	}
}
