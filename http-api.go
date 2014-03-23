// HTTP API to control probe service for representing collected data.
package main

import (
	"expvar"
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	"time"
)

// Elder.
func HttpAPI(cfg *Config) {
	r := mux.NewRouter()
	r.HandleFunc("/debug", expvarHandler).Methods("GET", "HEAD")
	r.HandleFunc("/", rootAPI).Methods("GET", "HEAD")
	// r.HandleFunc("/rprt", rprtMainPage).Methods("GET")
	// r.HandleFunc("/rprt/3hours", rprt3Hours).Methods("GET")
	// r.HandleFunc("/rprt/last", rprtLast).Methods("GET")
	// r.HandleFunc("/rprt/last-critical", rprtLastCritical).Methods("GET")
	// r.HandleFunc("/rprt/g/{group}", rprtGroup).Methods("GET")
	// r.HandleFunc("/rprt/g/{group}/last", rprtGroupLast).Methods("GET")
	// r.HandleFunc("/rprt/g/{group}/last-critical", rprtGroupLastCritical).Methods("GET")
	// r.HandleFunc("/zabbix", zabbixStatus).Methods("GET", "HEAD")                      // text report for all groups to Zabbix
	// r.HandleFunc("/mon/status/{group}", zabbixStatus).Methods("GET", "HEAD")          // text report for selected group to Zabbix
	// r.HandleFunc("/mon/status/{group}/{stream}", zabbixStatus).Methods("GET", "HEAD") // text report for selected group to Zabbix

	// Zabbix autodiscovery protocol (JSON)
	// https://www.zabbix.com/documentation/ru/2.0/manual/discovery/low_level_discovery
	// new API
	r.HandleFunc("/zabbix-discovery", zabbixDiscovery(cfg)).Methods("GET", "HEAD") // discovery data for Zabbix for all groups
	r.HandleFunc("/zabbix-discovery/{group}", zabbixDiscovery(cfg)).Methods("GET")

	// числовое значение ошибки для выбранных группы и канала в диапазоне errlevel from-upto
	r.HandleFunc("/mon/error/{type}/{group}/{stream}/{fromerrlevel}-{uptoerrlevel}", monError).Methods("GET")

	// static and client side
	r.Handle("/css/{{name}}.css", http.FileServer(http.Dir("bootstrap"))).Methods("GET")
	r.Handle("/js/{{name}}.js", http.FileServer(http.Dir("bootstrap"))).Methods("GET")
	r.Handle("/{{name}}.png", http.FileServer(http.Dir("pics"))).Methods("GET")
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
	if err := IndexPageTemplate.Execute(res, PageValues{Stubs.Name, StatsGlobals}); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}
}

// Webhandler. Возвращает text/plain значение ошибки для выбранных группы и канала.
func monError(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Server", SERVER)

	vars := mux.Vars(req)
	if vars["type"] != "" && vars["group"] != "" && vars["stream"] != "" && vars["fromerrlevel"] != "" && vars["uptoerrlevel"] != "" {
		result := LoadLastStats(String2StreamType(vars["type"]), vars["group"], vars["stream"])
		res.Write([]byte(StreamErr2String(result.ErrType)))
	}
}

// func rprtMainPage(res http.ResponseWriter, req *http.Request) {
// 	res.Header().Set("Server", SERVER)
// 	res.Write(ReportMainPage())
// }

// func rprtGroupAll(res http.ResponseWriter, req *http.Request) {
// 	res.Header().Set("Server", SERVER)
// 	res.Write([]byte("Сводный отчёт по группам."))
// }

// func rprtGroup(res http.ResponseWriter, req *http.Request) {
// 	res.Header().Set("Server", SERVER)
// 	res.Write([]byte("HLS Probe at service."))
// }

// // Group errors report
// func rprtGroupLast(res http.ResponseWriter, req *http.Request) {
// 	res.Header().Set("Server", SERVER)
// 	res.Write(ReportLast(mux.Vars(req), false))
// }

// // Group errors report
// func rprtGroupLastCritical(res http.ResponseWriter, req *http.Request) {
// 	res.Header().Set("Server", SERVER)
// 	res.Write(ReportLast(mux.Vars(req), true))
// }

// // Group errors report
// func rprt3Hours(res http.ResponseWriter, req *http.Request) {
// 	res.Header().Set("Server", SERVER)
// 	res.Write(Report3Hours(mux.Vars(req)))
// }

// // Group errors report
// func rprtLast(res http.ResponseWriter, req *http.Request) {
// 	res.Header().Set("Server", SERVER)
// 	res.Write(ReportLast(mux.Vars(req), false))
// }

// // Group errors report
// func rprtLastCritical(res http.ResponseWriter, req *http.Request) {
// 	res.Header().Set("Server", SERVER)
// 	res.Write(ReportLast(mux.Vars(req), true))
// }

// Zabbix integration
// func zabbixStatus(res http.ResponseWriter, req *http.Request) {
// 	res.Header().Set("Server", SERVER)
// 	res.Write(ZabbixStatus(mux.Vars(req)))
// }

// Zabbix integration (with cfg curried)
func zabbixDiscovery(cfg *Config) func(http.ResponseWriter, *http.Request) {
	return func(res http.ResponseWriter, req *http.Request) {
		res.Header().Set("Server", SERVER)
		res.Write(ZabbixDiscoveryWeb(cfg, mux.Vars(req)))
	}
}

func expvarHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Fprintf(w, "{\n")
	first := true
	expvar.Do(func(kv expvar.KeyValue) {
		if !first {
			fmt.Fprintf(w, ",\n")
		}
		first = false
		fmt.Fprintf(w, "%q: %s", kv.Key, kv.Value)
	})
	fmt.Fprintf(w, "\n}\n")
}
