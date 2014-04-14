// Web UI. Reports generator
package main

import (
	"net/http"
)

func ReportStreamInfo(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Server", SERVER)
	res.Write([]byte("Статистика канала."))
}
