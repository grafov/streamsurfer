// HTTP API to control probe service for representing collected data
package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	"time"
)

const (
	SERVER = "HLS Probe II"
)

// Elder.
func HttpAPI(cfg *Config) {
	r := mux.NewRouter()
	r.HandleFunc("/", rootAPI)
	r.HandleFunc("/rprt", rprtGroupAll).Methods("GET")
	r.HandleFunc("/rprt/g/{group}", rprtGroup).Methods("GET")
	r.HandleFunc("/rprt/g/{group}/errors", rprtGroupErrors).Methods("GET")
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

func rprtGroupAll(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Server", SERVER)
	res.Write([]byte("Сводный отчёт по группам."))
}

func rprtGroup(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Server", SERVER)
	res.Write([]byte("HLS Probe at service."))
}

// Group errors report
func rprtGroupErrors(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Server", SERVER)
	res.Write(ReportGroupErrors(mux.Vars(req)))
}
