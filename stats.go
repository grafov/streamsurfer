// The code keeps streams statistics and program internal statistics.
// Statistics output to files and to JSON HTTP API.
package main

import (
	"errors"
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
	errorsOut chan OutQuery
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

// Elder
// Gather results history and statistics from all streamboxes
func StatKeeper() {
	//var results map[Key][]Result = make(map[Key][]Result)
	var stats map[Key]Stats = make(map[Key]Stats)

	statIn = make(chan StatInQuery, 8192) // receive stats
	statOut = make(chan StatOutQuery, 8)  // send stats
	resultIn = make(chan ResultInQuery, 4096)
	resultOut = make(chan ResultOutQuery, 8)
	errorsOut = make(chan OutQuery, 8)

	// storage maintainance period
	///timer := time.Tick(12 * time.Second)

	for {
		select {
		case state := <-statIn: // receive new statitics data for saving
			stats[Key{state.Stream.Group, state.Stream.Name}] = state.Last

		case key := <-statOut:
			if val, ok := stats[key.Key]; ok {
				key.ReplyTo <- val
			} else {
				key.ReplyTo <- Stats{}
			}

		case state := <-resultIn: // incoming results from streamboxes
			//results[Key{state.Stream.Group, state.Stream.Name}] = append(results[Key{state.Stream.Group, state.Stream.Name}], state.Last)
			RedKeepResult(Key{state.Stream.Group, state.Stream.Name}, state.Last.Started, state.Last)
			RedKeepError(Key{state.Stream.Group, state.Stream.Name}, state.Last.Started, state.Last.ErrType)

		case key := <-resultOut:
			data, err := RedLoadResults(key.Key, time.Now().Add(-24*time.Hour), time.Now())
			if err != nil {
				key.ReplyTo <- nil
			} else {
				key.ReplyTo <- data
			}

		case key := <-errorsOut: // get error list by streams
			data, err := RedLoadErrors(key.Key, key.From, key.To)
			if err != nil {
				key.ReplyTo <- nil
			} else {
				key.ReplyTo <- data
			}
		}
	}
}

// Put result of probe task to statistics.
func SaveStats(stream Stream, last Stats) {
	statIn <- StatInQuery{Stream: stream, Last: last}
}

// Получить состояние по последней проверке.
func LoadStats(key Key) Stats {
	result := make(chan Stats)
	statOut <- StatOutQuery{Key: key, ReplyTo: result}
	data := <-result
	return data
}

// Put result of probe task to statistics.
func SaveResult(stream Stream, last Result) {
	resultIn <- ResultInQuery{Stream: stream, Last: last}
}

// Получить состояние по последней проверке.
func LoadLastResult(key Key) (KeepedResult, error) {
	result := make(chan []KeepedResult)
	resultOut <- ResultOutQuery{Key: key, ReplyTo: result}
	data := <-result
	if data != nil {
		return data[len(data)-1], nil
	} else {
		return KeepedResult{}, errors.New("result not found")
	}
}

// Получить статистику по каналу за всё время наблюдения.
func LoadHistoryResults(key Key) ([]KeepedResult, error) {
	result := make(chan []KeepedResult)
	resultOut <- ResultOutQuery{Key: key, ReplyTo: result}
	data := <-result
	if data != nil {
		return data, nil
	} else {
		return nil, errors.New("result not found")
	}
}

func LoadHistoryErrors(key Key, from time.Duration) ([]ErrType, error) {
	result := make(chan interface{})
	errorsOut <- OutQuery{Key: key, From: time.Now().Add(-from), To: time.Now(), ReplyTo: result}
	data := <-result
	if data != nil {
		return data.([]ErrType), nil
	} else {
		return nil, errors.New("result not found")
	}
}
