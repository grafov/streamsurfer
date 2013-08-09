// Web reports generator
package main

import (
	//	"fmt"
	"github.com/hoisie/mustache"
	"strconv"
	"sync"
	"time"
)

type StreamState struct {
	Stream Stream
	Result TaskResult
}

type ReportData struct {
	Vars      map[string]string
	TableData []map[string]string // array of table rows
}

var reports chan StreamState
var ReportedStreams = struct {
	sync.RWMutex
	data map[string]map[string]StreamState
}{data: make(map[string]map[string]StreamState)}

func ReportKeeper(cfg *Config) {
	reports = make(chan StreamState, 1024)
	for {
		state := <-reports
		ReportedStreams.Lock()
		if _, err := ReportedStreams.data[state.Stream.Group]; !err {
			ReportedStreams.data[state.Stream.Group] = make(map[string]StreamState)
		}
		ReportedStreams.data[state.Stream.Group][state.Stream.Name] = state
		ReportedStreams.Unlock()
	}
}

// Wrapper to send report to ReportKeeper
func Report(stream Stream, result *TaskResult) {
	reports <- StreamState{Stream: stream, Result: *result}
}

func ReportLast(vars map[string]string) []byte {
	var page string
	var values []map[string]string

	ReportedStreams.RLock()
	defer ReportedStreams.RUnlock()

	if _, exists := vars["group"]; exists { // report for selected group
		for _, value := range ReportedStreams.data[vars["group"]] {
			rprtAddRow(&values, value)
		}
		page = mustache.Render(ReportGroupLastTemplate, ReportData{Vars: vars, TableData: values})
	} else { // report for all groups
		for _, group := range ReportedStreams.data {
			for _, value := range group {
				rprtAddRow(&values, value)
			}
		}
		page = mustache.Render(ReportLastTemplate, ReportData{TableData: values})
	}

	return []byte(page)
}

// Helper.
func rprtAddRow(values *[]map[string]string, value StreamState) {
	var severity, error string

	if value.Result.Type > SUCCESS || value.Result.Elapsed >= 10*time.Second {
		if value.Result.Type == SUCCESS {
			severity = "warning"
			error = "timeout"
		} else {
			severity = "error"
			error = StreamErrText(value.Result.Type)
		}
		*values = append(*values, map[string]string{
			"uri":           value.Stream.URI,
			"name":          value.Stream.Name,
			"group":         value.Stream.Group,
			"status":        value.Result.HTTPStatus,
			"contentlength": strconv.FormatInt(value.Result.ContentLength, 10),
			"started":       value.Result.Started.Format(TimeFormat),
			"elapsed":       value.Result.Elapsed.String(),
			"error":         error,
			"severity":      severity,
		})
	}
}
