// Web UI. Reports generator
package main

import (
	"fmt"
	"net/http"
	"strconv"
	"text/template"
	"time"
)

type PageMeta struct {
	Title     string
	IsStatus  bool
	IsControl bool
	IsReport  bool
}

type PageTable struct {
	PageMeta
	Head []string
	Body [][]string
}

func ReportStreamInfo(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Server", SERVER)
	tmpl := template.Must(template.New("ReportStreamInfo").Parse(tmpltReportStreamInfo))
	test := make(map[string]TestMe)
	test["раз"] = TestMe{1}
	test["два"] = TestMe{2}
	tmpl.Execute(res, test)

}

func ReportStreamHistory(res http.ResponseWriter, req *http.Request) {
	var severity string

	vars := setupHTTP(&res, req)
	data, err := LoadHistoryStats(vars["group"], vars["stream"])
	if err != nil {
		http.Error(res, "Stream not found or not tested yet.", http.StatusNotFound)
		return
	}
	if vars["stamp"] != "" { // отобразить подробности по ошибке
		for _, val := range *data {
			stamp, err := strconv.ParseInt(vars["stamp"], 10, 64)
			if err != nil {
				goto FullHistory
			}
			if val.Started == time.Unix(0, stamp) {
				res.Write([]byte(fmt.Sprintf("==================================================\nGET %s\n\n", val.Task.URI)))
				val.Headers.Write(res)
				res.Write([]byte("\n"))
				res.Write(val.Body.Bytes())
				if val.SubResults != nil {
					for _, sub := range val.SubResults {
						res.Write([]byte(fmt.Sprintf("\n==================================================\nGET %s\n\n", sub.Task.URI)))
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
	table := PageTable{PageMeta: PageMeta{Title: fmt.Sprintf("%s/%s checks history", vars["group"], vars["stream"]), IsReport: true},
		Head: []string{"Time", "Error", "Status", "Content length", "Raw result"}}
	for _, val := range *data {
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
		table.Body = append(table.Body,
			[]string{severity,
				val.Started.String(),
				StreamErr2String(val.ErrType),
				val.HTTPStatus,
				strconv.FormatInt(val.ContentLength, 10),
				fmt.Sprintf("<a target=\"blank\" href=\"%d\">show raw result</a>", val.Started.UnixNano())})
	}
	Page.ExecuteTemplate(res, "report-stream-history", table)
}
