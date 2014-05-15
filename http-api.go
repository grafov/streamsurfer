// HTTP API to control probe service for representing collected data.
package main

import (
	"crypto/sha1"
	"encoding/base64"
	"expvar"
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	"os"
	"strconv"
	"strings"
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
	r.HandleFunc("/debug", HandleHTTP(expvarHandler)).Methods("GET", "HEAD")
	r.HandleFunc("/", HandleHTTP(rootAPI)).Methods("GET", "HEAD")

	/* Monitoring interface (for humans and robots)
	 */
	// Show stream list for all groups
	r.HandleFunc("/act", HandleHTTP(ActivityIndex)).Methods("GET")
	// Show stream list for the group
	r.HandleFunc("/act/{group}", HandleHTTP(ActivityIndex)).Methods("GET")
	// Информация о потоке и сводная статистика
	r.HandleFunc("/act/{group}/{stream}", HandleHTTP(ActivityStreamInfo)).Methods("GET")
	r.HandleFunc("/act/{group}/{stream}/", HandleHTTP(ActivityStreamInfo)).Methods("GET")
	// История ошибок
	r.HandleFunc("/act/{group}/{stream}/{mode:history|errors}", HandleHTTP(ActivityStreamHistory)).Methods("GET")
	// Вывод результата проверки для мастер-плейлиста
	r.HandleFunc("/act/{group}/{stream}/{stamp:[0-9]+}/raw", HandleHTTP(ActivityStreamHistory)).Methods("GET")
	// Вывод результата проверки для вложенных проверок
	r.HandleFunc("/act/{group}/{stream}/{stamp:[0-9]+}/{idx:[0-9]+}/raw", HandleHTTP(ActivityStreamHistory)).Methods("GET")

	/* Zabbix integration
	 */
	// Discovery data for Zabbix for all groups
	r.HandleFunc("/zabbix-discovery", HandleHTTP(zabbixDiscovery())).Methods("GET", "HEAD")
	r.HandleFunc("/zabbix-discovery/{group}", HandleHTTP(zabbixDiscovery())).Methods("GET", "HEAD")
	// строковое значение ошибки для выбранных группы и канала
	r.HandleFunc("/mon/error/{group}/{stream}/{astype:int|str}", HandleHTTP(monError)).Methods("GET", "HEAD")
	// числовое значение ошибки для выбранных группы и канала в диапазоне errlevel from-upto
	r.HandleFunc("/mon/error/{group}/{stream}/{fromerrlevel:[a-z]+}-{uptoerrlevel:[a-z]+}", HandleHTTP(monErrorLevel)).Methods("GET")

	/* Reports for humans
	 */
	// Вывод описания ошибки из анализатора
	r.HandleFunc("/rpt", HandleHTTP(ReportIndex)).Methods("GET")
	r.HandleFunc("/rpt/", HandleHTTP(ReportIndex)).Methods("GET")
	r.HandleFunc("/rpt/{rptid:[0-9]+}", HandleHTTP(ReportStreamErrors)).Methods("GET")

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

	/* Misc static data. Unauthorized access allowed.
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

func rootAPI(res http.ResponseWriter, req *http.Request, vars map[string]string) {
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
func monError(res http.ResponseWriter, req *http.Request, vars map[string]string) {
	res.Header().Set("Server", SERVER)
	res.Header().Set("Content-Type", "text/plain")

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
func monErrorLevel(res http.ResponseWriter, req *http.Request, vars map[string]string) {
	res.Header().Set("Server", SERVER)
	res.Header().Set("Content-Type", "text/plain")

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
func zabbixDiscovery() func(http.ResponseWriter, *http.Request, map[string]string) {
	return func(res http.ResponseWriter, req *http.Request, vars map[string]string) {
		res.Write(ZabbixDiscoveryWeb(vars))
	}
}

func expvarHandler(w http.ResponseWriter, r *http.Request, vars map[string]string) {
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

// Wrapper for all HTTP handlers.
// Does authorization and preparation of headers.
func HandleHTTP(f func(http.ResponseWriter, *http.Request, map[string]string)) func(http.ResponseWriter, *http.Request) {
	var user string

	handler := func(resp http.ResponseWriter, req *http.Request) {
		resp.Header().Set("Server", SURFER)
		if cfg.User != "" && cfg.Pass != "" {
			user = checkAuth(req)
			if user != "" {
				resp.Header().Set("X-Authenticated-Username", user)
			} else {
				requireAuth(resp, req, nil)
				return
			}
		}
		vars := mux.Vars(req)
		f(resp, req, vars)
	}
	return handler
}

// Handler for unauthorized access.
func requireAuth(w http.ResponseWriter, r *http.Request, v map[string]string) {
	w.Header().Set("WWW-Authenticate", `Basic realm="`+SURFER+`"`)
	w.WriteHeader(401)
	w.Write([]byte("401 Unauthorized\n"))
}

/*
 TODO адаптировать вместо использования модуля auth

 Checks the username/password combination from the request. Returns
 either an empty string (authentication failed) or the name of the
 authenticated user.

 Supports MD5 and SHA1 password entries
*/
func checkAuth(r *http.Request) string {
	var passwd string

	s := strings.SplitN(r.Header.Get("Authorization"), " ", 2)
	if len(s) != 2 || s[0] != "Basic" {
		return ""
	}

	b, err := base64.StdEncoding.DecodeString(s[1])
	if err != nil {
		return ""
	}
	pair := strings.SplitN(string(b), ":", 2)
	if len(pair) != 2 {
		return ""
	}
	if pair[0] == cfg.User {
		passwd = cfg.Pass
	} else {
		return ""
	}
	if passwd[:5] == "{SHA}" {
		d := sha1.New()
		d.Write([]byte(pair[1]))
		if passwd[5:] != base64.StdEncoding.EncodeToString(d.Sum(nil)) {
			return ""
		}
	} else {
		e := NewMD5Entry(passwd)
		if e == nil {
			return ""
		}
		if passwd != string(MD5Crypt([]byte(pair[1]), e.Salt, e.Magic)) {
			return ""
		}
	}
	return pair[0]
}
