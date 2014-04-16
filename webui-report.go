// Web UI. Reports generator
package main

import (
	"net/http"
	"text/template"
)

func ReportStreamInfo(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Server", SERVER)
	tmpl := template.Must(template.New("test").Parse(tReportStreamInfo))
	test := make(map[string]TestMe)
	test["раз"] = TestMe{1}
	test["два"] = TestMe{2}
	tmpl.Execute(res, test)
	//	res.Write([]byte("Статистика канала."))

}
