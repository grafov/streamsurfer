// The code keeps streams statistics and program internal statistics.
// Statistics output to files and to JSON HTTP API.
package main

import (
	"errors"
	//"github.com/garyburd/redigo/redis"
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
	statIn    chan StatInQuery
	statOut   chan StatOutQuery
	resultIn  chan ResultInQuery
	resultOut chan ResultOutQuery
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
	Key
	Error *Result
}

/* Структуры данных для статистики:

map текущих состояний:
 Key{Type, Group, Name}
  Result

map результатов по времени:
 Key{Type, Group, Name}
  time.Time
    Result
*/

// Elder
// Gather results history and statistics from all streamboxes
func StatKeeper() {
	var results map[Key][]*Result = make(map[Key][]*Result)
	var stats map[Key]Stats = make(map[Key]Stats)

	statIn = make(chan StatInQuery, 8192) // receive stats
	statOut = make(chan StatOutQuery, 8)  // send stats
	resultIn = make(chan ResultInQuery, 4096)
	resultOut = make(chan ResultOutQuery, 8)

	// storage maintainance period
	timer := time.Tick(12 * time.Second)

	for {
		select {
		case state := <-statIn: // receive new statitics data for saving
			stats[Key{state.Stream.Group, state.Stream.Name}] = state.Last
		case key := <-statOut:
			if val, ok := stats[key.Key]; ok {
				key.ReplyTo <- &val
			} else {
				key.ReplyTo <- nil
			}
		case state := <-resultIn: // incoming results from streamboxes
			results[Key{state.Stream.Group, state.Stream.Name}] = append(results[Key{state.Stream.Group, state.Stream.Name}], state.Last)
			/* TODO
			Последние 30 минут лежат в памяти, остальное в редисе. Если редиса нет, глубокая статистика недоступна.

			*/
		case key := <-resultOut:
			if val, ok := results[key.Key]; ok {
				key.ReplyTo <- val
			} else {
				key.ReplyTo <- nil
			}

		case <-timer: // cleanup old history entries
			// log.Println("Cleanup routine entered. Cache len: ", len(stats), oldestStoredTime)
			if len(cfg.GroupParams) == 0 || len(stats) == 0 {
				goto cleanupExit
			}

			for _, result := range results {
				// for e := streamstats.Front(); e != nil; e = e.Next() {
				// 	if time.Since(e.Value.(Result).Started) > 3*time.Minute { // XXX set 30 min
				// 		streamstats.Remove(e)
				// 	}
				// }
				// XXX очистка не работает!
				for _, val := range result {
					if time.Since(val.Started) >= 2*time.Minute {
						result = result[1:] // we have ordered by time list
					}
				}
			}
		cleanupExit:
		}
	}
}

// Put result of probe task to statistics.
func SaveStats(stream Stream, last Stats) {
	statIn <- StatInQuery{Stream: stream, Last: last}
}

// Получить состояние по последней проверке.
func LoadStats(key Key) (*Stats, error) {
	result := make(chan *Stats)
	statOut <- StatOutQuery{Key: key, ReplyTo: result}
	data := <-result
	if data != nil {
		return data, nil
	} else {
		return nil, errors.New("stats not found")
	}
}

// Put result of probe task to statistics.
func SaveResult(stream Stream, last *Result) {
	resultIn <- ResultInQuery{Stream: stream, Last: last}
}

// Получить состояние по последней проверке.
func LoadLastResult(key Key) (*Result, error) {
	result := make(chan []*Result)
	resultOut <- ResultOutQuery{Key: key, ReplyTo: result}
	data := <-result
	if data != nil {
		return data[len(data)-1], nil
	} else {
		return nil, errors.New("result not found")
	}
}

// Получить статистику по каналу за всё время наблюдения.
func LoadHistoryResults(key Key) ([]*Result, error) {
	result := make(chan []*Result)
	resultOut <- ResultOutQuery{Key: key, ReplyTo: result}
	data := <-result
	if data != nil {
		return data, nil
	} else {
		return nil, errors.New("result not found")
	}
}
