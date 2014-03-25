// Probers for dif
package main

import (
	"expvar"
	"fmt"
	"github.com/grafov/bcast"
	"github.com/grafov/m3u8"
	"strings"
	"time"
)

// Probe HTTP without additional protocol parsing.
// SaveStats timeouts and bad statuses.
func SimpleProber(ctl *bcast.Group, tasks chan *Task, debugvars *expvar.Map) {
	var result *Result

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("trace dumped in HTTP prober:", r)
		}
	}()

	for {
		queueCount := debugvars.Get("http-tasks-queue")
		queueCount.(*expvar.Int).Set(int64(len(tasks)))
		task := <-tasks
		if time.Now().Before(task.TTL) {
			result = ExecHTTP(task)
			debugvars.Add("http-tasks-done", 1)
		} else {
			result = TaskExpired(task)
			debugvars.Add("http-tasks-expired", 1)
		}
		task.ReplyTo <- *result
	}
}

// TODO к реализации
// Probe HTTP with additional checks for Widevine.
// Really now only http-range check supported.
func WidevineProber(ctl *bcast.Group, tasks chan *Task, debugvars *expvar.Map) {
	var result *Result

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("trace dumped in Widevine prober:", r)
		}
	}()

	for {
		queueCount := debugvars.Get("wv-tasks-queue")
		queueCount.(*expvar.Int).Set(int64(len(tasks)))
		task := <-tasks
		if time.Now().Before(task.TTL) {
			result = ExecHTTP(task)
			debugvars.Add("wv-tasks-done", 1)
		} else {
			result = TaskExpired(task)
			debugvars.Add("wv-tasks-expired", 1)
		}
		task.ReplyTo <- *result
	}
}

// HTTP Live Streaming support.
// Parse and probe M3U8 playlists (multi- and single bitrate)
// and report time statistics and errors
func CupertinoProber(ctl *bcast.Group, tasks chan *Task, debugvars *expvar.Map) {
	var result *Result

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("trace dumped in HLS prober:", r)
		}
	}()

	for {
		queueCount := debugvars.Get("hls-tasks-queue")
		queueCount.(*expvar.Int).Set(int64(len(tasks)))
		task := <-tasks
		if time.Now().Before(task.TTL) {
			result = ExecHTTP(task)
			if result.ErrType < ERROR_LEVEL && result.HTTPCode < 400 && result.ContentLength > 0 {
				playlist, listType, err := m3u8.Decode(result.Body, true)
				if err != nil {
					result.ErrType = BADFORMAT
				} else {
					switch listType {
					case m3u8.MASTER:
						//fmt.Printf("%+v\n", playlist)
						m := playlist.(*m3u8.MasterPlaylist)
						subresult := make(chan *Result, 24)
						for _, variant := range m.Variants {
							splitted := strings.Split(task.URI, "/")
							splitted[len(splitted)-1] = variant.URI
							suburi := strings.Join(splitted, "/")
							subtask := &Task{Stream: Stream{suburi, HLS, task.Name, task.Title, task.Group}}
							go func(subtask *Task) {
								subresult <- ExecHTTP(subtask)
							}(subtask)

							//tasks <- task
							// XXX
						}
						select {
						case data := <-subresult:
							result.SubResults = append(result.SubResults, data)
						case <-time.After(60 * time.Second):
						}
					case m3u8.MEDIA:
						p := playlist.(*m3u8.MediaPlaylist)
						p.Encode().String()
					default:
						result.ErrType = BADFORMAT
					}
				}
			}
			debugvars.Add("hls-tasks-done", 1)
		} else {
			result = TaskExpired(task)
			debugvars.Add("hls-tasks-expired", 1)
		}

		task.ReplyTo <- *result
		debugvars.Add("hls-tasks-done", 1)
	}
}

// HTTP Dynamic Streaming prober.
// Parse and probe F4M playlists and report time statistics and errors.
func SanjoseProber(ctl *bcast.Group, tasks chan *Task, debugvars *expvar.Map) {
	for {
		task := <-tasks
		result := ExecHTTP(task)
		task.ReplyTo <- *result
		debugvars.Add("hds-tasks-done", 1)
	}
}

// Parse and probe media chunk
// and report time statistics and errors
func MediaProber(ctl *bcast.Group, streamType StreamType, taskq chan *Task, debugvars *expvar.Map) {
	for {
		time.Sleep(20 * time.Second)
	}
}
