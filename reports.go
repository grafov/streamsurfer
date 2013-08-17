// Web reports generator
package main

import (
	"fmt"
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
		errtype := ""
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
				errtype = "sw"
			case BADSTATUS:
				errtype = "bs"
			case BADURI:
				errtype = "bu"
			case RTIMEOUT:
				errtype = "rt"
			case CTIMEOUT:
				errtype = "ct"
			}
			tmptbl[key.Group][key.Name][errtype] = s
			switch {
			case val == 1:
				tmptbl[key.Group][key.Name][fmt.Sprintf("%s-severity", errtype)] = "info"
			case val > 1 && val <= 6:
				tmptbl[key.Group][key.Name][fmt.Sprintf("%s-severity", errtype)] = "warning"
			case val > 6:
				tmptbl[key.Group][key.Name][fmt.Sprintf("%s-severity", errtype)] = "error"
			default:
				tmptbl[key.Group][key.Name][fmt.Sprintf("%s-severity", errtype)] = "success"
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

// Report last state of reported streams. Only failures counted.
func ReportLast(vars map[string]string, critical bool) []byte {
	var values []map[string]string
	var page string

	ReportedStreams.RLock()
	defer ReportedStreams.RUnlock()

	if _, exists := vars["group"]; exists { // report for selected group
		for _, value := range ReportedStreams.data[vars["group"]] {
			rprtLastAddRow(&values, value, critical)
		}
		page = mustache.Render(ReportGroupLastTemplate, ReportData{Vars: vars, TableData: values})
	} else { // report for all groups
		for _, group := range ReportedStreams.data {
			for _, value := range group {
				rprtLastAddRow(&values, value, critical)
			}
		}
		page = mustache.Render(ReportLastTemplate, ReportData{TableData: values})
	}

	return []byte(page)
}

// Helper.
func rprtLastAddRow(values *[]map[string]string, value StreamStats, critical bool) {
	var severity string

	if value.Last.ErrType > BADREQUEST {
		switch {
		case value.Last.ErrType > BADREQUEST && value.Last.ErrType < BADSTATUS:
			if critical {
				return
			}
			severity = "warning"
		case value.Last.ErrType >= BADSTATUS:
			severity = "error"
		default:
			if critical {
				return
			}
			severity = "unknown"
		}
		*values = append(*values, map[string]string{
			"uri":           value.Stream.URI,
			"name":          value.Stream.Name,
			"group":         value.Stream.Group,
			"status":        value.Last.HTTPStatus,
			"contentlength": strconv.FormatInt(value.Last.ContentLength, 10),
			"started":       value.Last.Started.Format(TimeFormat),
			"elapsed":       strconv.FormatFloat(value.Last.Elapsed.Seconds(), 'f', 3, 64),
			"error":         StreamErrText(value.Last.ErrType),
			"totalerrs":     strconv.FormatUint(uint64(value.Last.TotalErrs), 10),
			"severity":      severity,
		})
	}
}
