// Web UI. Reports generator
package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func ReportStreams(res http.ResponseWriter, req *http.Request) {
	var tbody [][]string

	vars := setupHTTP(&res, req)
	data := make(map[string]interface{})
	if vars["group"] != "" {
		data["title"] = fmt.Sprintf("List of streams for %s", vars["group"])
	} else {
		data["title"] = "List of streams"
	}
	if vars["group"] != "" {
		data["thead"] = []string{"Name", "Checks", "Errors"}
	} else {
		data["thead"] = []string{"Group", "Name", "Checks", "Errors"}
	}
	data["isreport"] = true
	for gname := range cfg.GroupParams {
		if vars["group"] != "" && gname != strings.ToLower(vars["group"]) {
			continue
		}
		for _, stream := range *cfg.GroupStreams[gname] {
			tbody = append(tbody,
				[]string{href(fmt.Sprintf("/rpt/%s", gname), gname),
					href(fmt.Sprintf("/rpt/%s/%s", gname, stream.Name), stream.Name),
					"0", "0"})
		}
	}
	data["tbody"] = tbody
	Page.ExecuteTemplate(res, "report-stream-list", data)
}

func ReportStreamInfo(res http.ResponseWriter, req *http.Request) {
	vars := setupHTTP(&res, req)
	data := make(map[string]interface{})
	data["title"] = fmt.Sprintf("%s/%s info", vars["group"], vars["stream"])
	data["isreport"] = true
	data["history"] = fmt.Sprintf("/rpt/%s/%s/history", vars["group"], vars["stream"])
	last, err := LoadLastStats(Key{vars["group"], vars["stream"]})
	if err == nil {
		data["url"] = last.Task.URI
	}
	data["slowcount"] = 0
	data["timeoutcount"] = 0
	data["httpcount"] = 0
	data["formatcount"] = 0
	hist, err := LoadHistoryStats(Key{vars["group"], vars["stream"]})
	if err == nil {
		for _, val := range *hist {
			switch val.ErrType {
			case SLOW, VERYSLOW:
				data["slowcount"] = data["slowcount"].(int) + 1
			case CTIMEOUT, RTIMEOUT:
				data["timeoutcount"] = data["timeoutcount"].(int) + 1
			case BADLENGTH, BODYREAD, REFUSED, BADSTATUS, BADURI:
				data["httpcount"] = data["httpcount"].(int) + 1
			case LISTEMPTY, BADFORMAT:
				data["formatcount"] = data["formatcount"].(int) + 1
			}
		}
	}
	Page.ExecuteTemplate(res, "report-stream-info", data)
}

func ReportStreamHistory(res http.ResponseWriter, req *http.Request) {
	var severity string
	var tbody [][]string

	data := make(map[string]interface{})
	vars := setupHTTP(&res, req)
	hist, err := LoadHistoryStats(Key{vars["group"], vars["stream"]})
	if err != nil {
		http.Error(res, "Stream not found or not tested yet.", http.StatusNotFound)
		return
	}
	if vars["stamp"] != "" { // отобразить подробности по ошибке
		for _, val := range *hist {
			stamp, err := strconv.ParseInt(vars["stamp"], 10, 64)
			if err != nil {
				goto FullHistory
			}
			if val.Started == time.Unix(0, stamp) {
				if vars["idx"] == "" {
					res.Write([]byte(fmt.Sprintf("GET %s\n\n", val.Task.URI)))
					val.Headers.Write(res)
					res.Write([]byte("\n"))
					res.Write(val.Body.Bytes())
				} else {
					idx, err := strconv.Atoi(vars["idx"])
					if err != nil {
						goto FullHistory
					}
					if len(val.SubResults) >= idx+1 {
						sub := val.SubResults[idx]
						res.Write([]byte(fmt.Sprintf("GET %s\n\n", sub.Task.URI)))
						sub.Headers.Write(res)
						res.Write([]byte("\n"))
						res.Write(sub.Body.Bytes())
					}
				}
				return
			}
		}
	}

FullHistory:
	data["title"] = fmt.Sprintf("%s/%s checks history", vars["group"], vars["stream"])
	data["isreport"] = true
	data["thead"] = []string{"Check type", "Date/time", "Check status", "HTTP status", "Time elapsed", "Content length", "Raw result"}
	for i := len(*hist) - 1; i >= 0; i-- { //_, val := range *data {
		val := (*hist)[i]
		switch {
		case val.ErrType == SUCCESS:
			severity = "info"
		case val.ErrType < WARNING_LEVEL:
			severity = "warning"
		case val.ErrType >= WARNING_LEVEL:
			severity = "error"
		default:
			severity = "success"
		}
		tbody = append(tbody,
			[]string{severity,
				"master",
				val.Started.String(),
				StreamErr2String(val.ErrType),
				val.HTTPStatus,
				val.Elapsed.String(),
				strconv.FormatInt(val.ContentLength, 10),
				href(fmt.Sprintf("%d/raw", val.Started.UnixNano()), "show raw result")})
		for idx, sub := range val.SubResults {
			switch {
			case sub.ErrType == SUCCESS:
				severity = "info"
			case sub.ErrType < WARNING_LEVEL:
				severity = "warning"
			case sub.ErrType >= WARNING_LEVEL:
				severity = "error"
			default:
				severity = "success"
			}
			tbody = append(tbody,
				[]string{severity,
					"media",
					sub.Started.String(),
					StreamErr2String(sub.ErrType),
					sub.HTTPStatus,
					sub.Elapsed.String(),
					strconv.FormatInt(sub.ContentLength, 10),
					href(fmt.Sprintf("%d/%d/raw", val.Started.UnixNano(), idx), "show raw result")})
		}
	}
	data["tbody"] = tbody
	Page.ExecuteTemplate(res, "report-stream-history", data)
}
