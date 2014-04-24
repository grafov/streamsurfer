// HTTP API to control probe service for representing collected data.
package main

import (
	"expvar"
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	"os"
	"strconv"
	"text/template"
	"time"
)

var Page *template.Template

// Elder.
func HttpAPI() {
	var err error

	Page, err = template.ParseGlob("templates/*.tmpl")
	if err != nil {
		fmt.Printf("Error in template with error %s", err)
		os.Exit(1)
	}

	r := mux.NewRouter()
	r.HandleFunc("/debug", expvarHandler).Methods("GET", "HEAD")
	r.HandleFunc("/", rootAPI).Methods("GET", "HEAD")

	/* Monitoring interface (for humans and robots)
	 */
	// Show stream list for all groups
	r.HandleFunc("/act", ActivityIndex).Methods("GET")
	// Show stream list for the group
	r.HandleFunc("/act/{group}", ActivityIndex).Methods("GET")
	// Информация о потоке и сводная статистика
	r.HandleFunc("/act/{group}/{stream}", ReportStreamInfo).Methods("GET")
	r.HandleFunc("/act/{group}/{stream}/", ReportStreamInfo).Methods("GET")
	// История ошибок
	r.HandleFunc("/act/{group}/{stream}/history", ReportStreamHistory).Methods("GET")
	// Вывод результата проверки для мастер-плейлиста
	r.HandleFunc("/act/{group}/{stream}/{stamp:[0-9]+}/raw", ReportStreamHistory).Methods("GET")
	// Вывод результата проверки для вложенных проверок
	r.HandleFunc("/act/{group}/{stream}/{stamp:[0-9]+}/{idx:[0-9]+}/raw", ReportStreamHistory).Methods("GET")

	/* Zabbix integration
	 */
	// Discovery data for Zabbix for all groups
	r.HandleFunc("/zabbix-discovery", zabbixDiscovery()).Methods("GET", "HEAD")
	r.HandleFunc("/zabbix-discovery/{group}", zabbixDiscovery()).Methods("GET", "HEAD")
	// строковое значение ошибки для выбранных группы и канала
	r.HandleFunc("/mon/error/{group}/{stream}/{astype:int|str}", monError).Methods("GET", "HEAD")
	// числовое значение ошибки для выбранных группы и канала в диапазоне errlevel from-upto
	r.HandleFunc("/mon/error/{group}/{stream}/{fromerrlevel:[a-z]+}-{uptoerrlevel:[a-z]+}", monErrorLevel).Methods("GET")

	/* Reports for humans
	 */
	// Вывод описания ошибки из анализатора
	r.HandleFunc("/rpt", ReportIndex).Methods("GET")
	r.HandleFunc("/rpt/", ReportIndex).Methods("GET")
	r.HandleFunc("/rpt/{rptid:[0-9]+}", ReportStreamErrors).Methods("GET")

	// Obsoleted reports with old API:
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

	/* Misc static data
	 */
	r.Handle("/css/{{name}}.css", http.FileServer(http.Dir("bootstrap"))).Methods("GET", "HEAD")
	r.Handle("/js/{{name}}.js", http.FileServer(http.Dir("bootstrap"))).Methods("GET", "HEAD")
	r.Handle("/{{name}}.png", http.FileServer(http.Dir("pics"))).Methods("GET", "HEAD")
	r.Handle("/favicon.ico", http.FileServer(http.Dir("pics"))).Methods("GET", "HEAD")
	fmt.Printf("Listen for API connections at %s\n", cfg.ListenHTTP)
	srv := &http.Server{
		Addr:        cfg.ListenHTTP,
		Handler:     r,
		ReadTimeout: 30 * time.Second,
	}
	srv.ListenAndServe()
}

func rootAPI(res http.ResponseWriter, req *http.Request) {
	data := make(map[string]interface{})
	data["title"] = cfg.Stubs.Name
	data["monState"] = StatsGlobals.MonitoringState
	data["totalMonPoints"] = StatsGlobals.TotalMonitoringPoints
	data["totalHLSMonPoints"] = StatsGlobals.TotalHLSMonitoringPoints
	data["totalHDSMonPoints"] = StatsGlobals.TotalHDSMonitoringPoints
	data["totalHTTPMonPoints"] = StatsGlobals.TotalHTTPMonitoringPoints
	Page.ExecuteTemplate(res, "index", data)
}

// Webhandler. Возвращает text/plain значение ошибки для выбранных группы и канала.
func monError(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Server", SERVER)
	res.Header().Set("Content-Type", "text/plain")

	vars := mux.Vars(req)
	if vars["group"] != "" && vars["stream"] != "" {
		if !StatsGlobals.MonitoringState {
			switch vars["astype"] { // пока мониторинг остановлен, считаем, что всё ок
			case "str":
				res.Write([]byte("success"))
			case "int":
				res.Write([]byte("0"))
			}
			return
		}
		if result, err := LoadLastResult(Key{vars["group"], vars["stream"]}); err == nil {
			switch vars["astype"] {
			case "str":
				res.Write([]byte(StreamErr2String(result.ErrType)))
			case "int":
				res.Write([]byte(strconv.Itoa(int(result.ErrType))))
			}
		} else { // пока проверки не проводились, считаем, что всё ок. Чего зря беспокоиться?
			switch vars["astype"] {
			case "str":
				res.Write([]byte("success"))
			case "int":
				res.Write([]byte("0"))
			}
		}
	} else {
		http.Error(res, "Bad parameters in query.", http.StatusBadRequest)
	}
}

// Webhandler. Возвращает text/plain значение ошибки для выбранных группы и канала.
// Если ошибка ниже заданного мин.уровня, то выдаётся 0=OK, если в границах указанных уровней, то 1=PROBLEM,
// если выше макс.уровня, то 2=FATAL
func monErrorLevel(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Server", SERVER)
	res.Header().Set("Content-Type", "text/plain")

	vars := mux.Vars(req)
	if vars["group"] != "" && vars["stream"] != "" && vars["fromerrlevel"] != "" && vars["uptoerrlevel"] != "" {
		if !StatsGlobals.MonitoringState {
			res.Write([]byte("0")) // пока мониторинг остановлен, считаем, что всё ок
			return
		}
		if result, err := LoadLastResult(Key{vars["group"], vars["stream"]}); err == nil {
			cur := result.ErrType
			switch {
			case cur <= String2StreamErr(vars["fromerrlevel"]):
				res.Write([]byte("0")) // OK
			case cur >= String2StreamErr(vars["uptoerrlevel"]):
				res.Write([]byte("2")) // FATAL
			default:
				res.Write([]byte("1")) // PROBLEM
			}
		} else {
			res.Write([]byte("0")) // пока проверки не проводились, считаем, что всё ок. Чего зря беспокоиться?
			//http.Error(res, "Result not found. Probably stream not tested yet.", http.StatusNotFound)
		}
	} else {
		http.Error(res, "Bad parameters in query.", http.StatusBadRequest)
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
func zabbixDiscovery() func(http.ResponseWriter, *http.Request) {
	return func(res http.ResponseWriter, req *http.Request) {
		res.Header().Set("Server", SERVER)
		res.Write(ZabbixDiscoveryWeb(mux.Vars(req)))
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

func setupHTTP(w *http.ResponseWriter, r *http.Request) map[string]string {
	(*w).Header().Set("Server", SURFER)

	return mux.Vars(r)
}
