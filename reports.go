// Web reports generator
package main

import (
	//	"fmt"
	"github.com/hoisie/mustache"
	"strconv"
	"time"
)

type ReportData struct {
	Vars      map[string]string
	TableData []map[string]string // array of table rows
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
func rprtAddRow(values *[]map[string]string, value StreamStats) {
	var severity string

	if value.Last.ErrType > SUCCESS || value.Last.Elapsed >= 10*time.Second {
		switch value.Last.ErrType {
		case SUCCESS, SLOW, VERYSLOW:
			severity = "warning"
		default:
			severity = "error"
		}
		*values = append(*values, map[string]string{
			"uri":           value.Stream.URI,
			"name":          value.Stream.Name,
			"group":         value.Stream.Group,
			"status":        value.Last.HTTPStatus,
			"contentlength": strconv.FormatInt(value.Last.ContentLength, 10),
			"started":       value.Last.Started.Format(TimeFormat),
			"elapsed":       value.Last.Elapsed.String(),
			"error":         StreamErrText(value.Last.ErrType),
			"totalerrs":     strconv.FormatUint(uint64(value.Last.TotalErrs), 10),
			"severity":      severity,
		})
	}
}
