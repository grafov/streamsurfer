// HTTP API to control probe service for representing collected data
package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	"time"
)

// Elder.
func HttpAPI(cfg *Config) {
	r := mux.NewRouter()
	r.HandleFunc("/", rootAPI)
	r.HandleFunc("/rprt", rprtMainPage).Methods("GET")
	r.HandleFunc("/rprt/3hours", rprt3Hours).Methods("GET")
	r.HandleFunc("/rprt/last", rprtLast).Methods("GET")
	r.HandleFunc("/rprt/g/{group}", rprtGroup).Methods("GET")
	r.HandleFunc("/rprt/g/{group}/last", rprtGroupLast).Methods("GET")
	r.HandleFunc("/zabbix", zabbixStatus).Methods("GET", "HEAD")
	r.HandleFunc("/zabbix/g/{group}", zabbixStatus).Methods("GET")
	r.Handle("/css/{{name}}.css", http.FileServer(http.Dir("bootstrap"))).Methods("GET")
	fmt.Printf("Listen for API connections at %s\n", cfg.Params.ListenHTTP)
	srv := &http.Server{
		Addr:        cfg.Params.ListenHTTP,
		Handler:     r,
		ReadTimeout: 30 * time.Second,
	}
	srv.ListenAndServe()
}

func rootAPI(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Server", SERVER)
	res.Write([]byte("HLS Probe at service."))
}

func rprtMainPage(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Server", SERVER)
	res.Write(ReportMainPage())
}

func rprtGroupAll(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Server", SERVER)
	res.Write([]byte("Сводный отчёт по группам."))
}

func rprtGroup(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Server", SERVER)
	res.Write([]byte("HLS Probe at service."))
}

// Group errors report
func rprtGroupLast(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Server", SERVER)
	res.Write(ReportLast(mux.Vars(req)))
}

// Group errors report
func rprt3Hours(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Server", SERVER)
	res.Write(Report3Hours(mux.Vars(req)))
}

// Group errors report
func rprtLast(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Server", SERVER)
	res.Write(ReportLast(mux.Vars(req)))
}

// Zabbix integration
func zabbixStatus(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Server", SERVER)
	res.Write(ZabbixStatus(mux.Vars(req)))
}
