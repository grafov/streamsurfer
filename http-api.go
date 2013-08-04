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
	r.HandleFunc("/", rootAPIHandler)
	fmt.Printf("Listen for API connections at %s\n", cfg.Params.ListenHTTP)
	srv := &http.Server{
		Addr:        cfg.Params.ListenHTTP,
		Handler:     r,
		ReadTimeout: 30 * time.Second,
	}
	srv.ListenAndServe()
}

func rootAPIHandler(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Server", SERVER)
	res.Write([]byte("HLS Probe at service."))
}
