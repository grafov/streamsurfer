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
	var severity string

	vars := setupHTTP(&res, req)
	data := make(map[string]interface{})
	if vars["group"] != "" {
		data["title"] = fmt.Sprintf("List of streams for %s", vars["group"])
	} else {
		data["title"] = "List of streams"
	}
	if vars["group"] != "" {
		data["thead"] = []string{"Name", "Checks", "Errors (6 min)", "Errors (6 hours)", "Errors (6 days)"}
	} else {
		data["thead"] = []string{"Group", "Name", "Checks", "Errors (6 min)", "Errors (6 hours)", "Errors (6 days)"}
	}
	data["isactivity"] = true
	for gname := range cfg.GroupParams {
		if vars["group"] != "" && gname != strings.ToLower(vars["group"]) {
			continue
		}
		for _, stream := range *cfg.GroupStreams[gname] {
			hist, err := LoadHistoryResults(Key{gname, stream.Name})
			errcount := 0
			if err == nil {
				for _, val := range hist {
					if val.ErrType > ERROR_LEVEL {
						errcount++
					}
				}
			}
			if errcount > 0 {
				severity = "warning"
			} else {
				severity = ""
			}
			if vars["group"] != "" {
				tbody = append(tbody, []string{
					severity,
					href(fmt.Sprintf("/act/%s/%s", gname, stream.Name), stream.Name),
					"0", strconv.Itoa(errcount), "0", "0"})
			} else {
				tbody = append(tbody, []string{
					severity,
					href(fmt.Sprintf("/act/%s", gname), gname),
					href(fmt.Sprintf("/act/%s/%s", gname, stream.Name), stream.Name),
					"0", strconv.Itoa(errcount), "0", "0"})
			}
		}
	}
	data["tbody"] = tbody
	Page.ExecuteTemplate(res, "report-stream-list", data)
}

func ReportStreamInfo(res http.ResponseWriter, req *http.Request) {
	vars := setupHTTP(&res, req)
	data := make(map[string]interface{})
	data["title"] = fmt.Sprintf("%s/%s info", vars["group"], vars["stream"])
	data["isactivity"] = true
	data["stream"] = vars["stream"]
	data["history"] = fmt.Sprintf("/act/%s/%s/history", vars["group"], vars["stream"])
	last, err := LoadLastResult(Key{vars["group"], vars["stream"]})
	if err == nil {
		data["url"] = last.Task.URI
	}
	data["slowcount"] = 0
	data["timeoutcount"] = 0
	data["httpcount"] = 0
	data["formatcount"] = 0
	hist, err := LoadHistoryResults(Key{vars["group"], vars["stream"]})
	if err == nil {
		for _, val := range hist {
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
	var severity, checktype string
	var tbody [][]string

	data := make(map[string]interface{})
	vars := setupHTTP(&res, req)
	hist, err := LoadHistoryResults(Key{vars["group"], vars["stream"]})
	if err != nil {
		http.Error(res, "Stream not found or not tested yet.", http.StatusNotFound)
		return
	}
	if vars["stamp"] != "" { // отобразить подробности по ошибке
		for _, val := range hist {
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
	data["isactivity"] = true
	data["stream"] = vars["stream"]
	data["thead"] = []string{"Check type", "Date/time", "Check result", "HTTP status", "Time elapsed", "Content length", "Raw result"}
	for i := len(hist) - 1; i >= 0; i-- { //_, val := range *data {
		val := (hist)[i]
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
		if val.Pid == nil { // TODO пофиксить для HTTP/HDS-проверок
			checktype = "master"
		} else {
			checktype = "media"
		}
		tbody = append(tbody,
			[]string{severity,
				span(checktype, "label"),
				val.Started.Format("2006-01-02 15:04:05 -0700"),
				StreamErr2String(val.ErrType),
				val.HTTPStatus,
				val.Elapsed.String(),
				strconv.FormatInt(val.ContentLength, 10),
				href(fmt.Sprintf("%d/raw", val.Started.UnixNano()), "show raw result")})
	}
	data["tbody"] = tbody
	Page.ExecuteTemplate(res, "report-stream-history", data)
}

func ReportIndex(res http.ResponseWriter, req *http.Request) {
	setupHTTP(&res, req)

	data := make(map[string]interface{})
	data["title"] = "Available reports"
	data["isreport"] = true
	Page.ExecuteTemplate(res, "report-index", data)
}

func ReportStreamErrors(res http.ResponseWriter, req *http.Request) {
	data := make(map[string]interface{})
	data["title"] = "Available reports"
	data["isreport"] = true
	Page.ExecuteTemplate(res, "report-stream-info", data)
}
