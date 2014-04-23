// OBSOLETED by webui-report.go
// Web reports generator
package main

import (
	"fmt"
	"github.com/hoisie/mustache" // TODO migrate back to native golang templates
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
		if key.Curhour == curhour {
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
			case REFUSED:
				errtype = "cr"
			default:
				continue // count only errors listed above
			}
			if _, exists := tmptbl[key.Group]; !exists {
				tmptbl[key.Group] = make(map[string]map[string]string)
			}
			if _, exists := tmptbl[key.Group][key.Name]; !exists {
				tmptbl[key.Group][key.Name] = make(map[string]string)
			}
			tmptbl[key.Group][key.Name][errtype] = strconv.FormatInt(int64(val), 10)
			tmptbl[key.Group][key.Name]["group"] = key.Group
			tmptbl[key.Group][key.Name]["name"] = key.Name
			tmptbl[key.Group][key.Name]["uri"] = key.URI
			switch { // how much errors per hour?
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

// Helper. For "last*" report.
func rprtLastAddRow(values *[]map[string]string, value StreamStats, critical bool) {
	var severity, report string

	switch {
	case value.Last.ErrType > WARNING_LEVEL && value.Last.ErrType < ERROR_LEVEL:
		if critical {
			return
		}
		severity = "warning"
	case value.Last.ErrType > ERROR_LEVEL && value.Last.ErrType < CRITICAL_LEVEL:
		if critical {
			return
		}
		severity = "error"
	case value.Last.ErrType > CRITICAL_LEVEL:
		severity = "critical"
	default:
		return
	}
	if critical {
		report = "last-critical"
	} else {
		report = "last"
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
		"report":        report, // report name
	})
}
