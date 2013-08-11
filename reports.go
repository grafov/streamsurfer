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

func ReportMainPage() []byte {
	return []byte(ReportMainPageTemplate)
}

// Errors for 3 last hours for all groups
// TODO number of hours make variable
func Report3Hours(vars map[string]string) []byte {
	var values []map[string]string
	var page string

	ErrHistory.RLock()
	defer ErrHistory.RUnlock()

	now := time.Now()
	curhour := now.Format("06010215")
	//	h2ago := now.Add(-2 * time.Hour).Format("06010215")
	//h3ago := now.Add(-3 * time.Hour).Format("06010215")
	tmptbl := make(map[string]map[string]map[string]string)

	for key, val := range ErrHistory.count {
		s := strconv.FormatInt(int64(val), 10)
		if key.Curhour == curhour {
			if _, exists := tmptbl[key.Group]; !exists {
				tmptbl[key.Group] = make(map[string]map[string]string)
			}
			if _, exists := tmptbl[key.Group][key.Name]; !exists {
				tmptbl[key.Group][key.Name] = make(map[string]string)
			}
			tmptbl[key.Group][key.Name]["group"] = key.Group
			tmptbl[key.Group][key.Name]["name"] = key.Name
			tmptbl[key.Group][key.Name]["uri"] = key.URI
			switch key.ErrType {
			case SLOW, VERYSLOW:
				tmptbl[key.Group][key.Name]["sw"] = s
			case BADSTATUS:
				tmptbl[key.Group][key.Name]["bs"] = s
			case BADURI:
				tmptbl[key.Group][key.Name]["bu"] = s
			case LISTEMPTY:
				tmptbl[key.Group][key.Name]["le"] = s
			case BADFORMAT:
				tmptbl[key.Group][key.Name]["bf"] = s
			case RTIMEOUT:
				tmptbl[key.Group][key.Name]["rt"] = s
			case CTIMEOUT:
				tmptbl[key.Group][key.Name]["ct"] = s
			case HLSPARSER:
				tmptbl[key.Group][key.Name]["hls"] = s
			}
		}
	}

	for _, val := range tmptbl {
		for _, counter := range val {
			values = append(values, counter)
		}
	}
	page = mustache.Render(Report3HoursTemplate, ReportData{TableData: values})

	return []byte(page)
}

func ReportLast(vars map[string]string) []byte {
	var values []map[string]string
	var page string

	ReportedStreams.RLock()
	defer ReportedStreams.RUnlock()

	if _, exists := vars["group"]; exists { // report for selected group
		for _, value := range ReportedStreams.data[vars["group"]] {
			rprtLastAddRow(&values, value)
		}
		page = mustache.Render(ReportGroupLastTemplate, ReportData{Vars: vars, TableData: values})
	} else { // report for all groups
		for _, group := range ReportedStreams.data {
			for _, value := range group {
				rprtLastAddRow(&values, value)
			}
		}
		page = mustache.Render(ReportLastTemplate, ReportData{TableData: values})
	}

	return []byte(page)
}

// Helper.
func rprtLastAddRow(values *[]map[string]string, value StreamStats) {
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
