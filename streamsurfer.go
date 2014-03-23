/*
 Stream Surfer is prober and monitor for HTTP video streaming.
 This file defines main entry points of the program.

 Copyleft 2013-2014  Alexander I.Grafov aka Axel <grafov@gmail.com>

 This program is free software: you can redistribute it and/or modify
 it under the terms of the GNU General Public License as published by
 the Free Software Foundation, either version 3 of the License, or
 (at your option) any later version.

 This program is distributed in the hope that it will be useful,
 but WITHOUT ANY WARRANTY; without even the implied warranty of
 MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 GNU General Public License for more details.

 You should have received a copy of the GNU General Public License
 along with this program.  If not, see <http://www.gnu.org/licenses/>.

 ॐ तारे तुत्तारे तुरे स्व
*/

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
)

const (
	SURFER  = "Stream Surfer v0.1-dev"
	VERSION = "0.X"
)

var build_date string
var Stubs = &StubValues{}

func main() {
	var confname = flag.String("config", "", "alternative configuration file")
	var verbose = flag.Bool("verbose", true, "verbose output of logs")

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Stream Surfer trace dumped:", r)
			if err := ioutil.WriteFile(FullPath("~/hlsprobe2.trace"), r.([]byte), 0644); err != nil {
				fmt.Println("Can't write trace file!")
			}
		}
	}()

	fmt.Printf("Stream Surfer vers. %s (build %s)\n", VERSION, build_date)
	flag.Parse()

	//cfgq := make(chan ConfigQuery, 12)
	//go SourceLoader(*config, cfgq)
	cfg := ReadConfig(*confname)

	go LogKeeper(cfg, *verbose) // collect program logs and write them to file
	go StatKeeper(cfg)          // collect probe statistics and may be queried by report builders

	go StreamMonitor(cfg)       // probe logic
	go ZabbixDiscoveryFile(cfg) // maintain discovery file for Zabbix
	go HttpAPI(cfg)             // control API

	terminate := make(chan os.Signal)
	signal.Notify(terminate, os.Interrupt)
	<-terminate
	fmt.Println("...probe service interrupted.")
}

// Top level problem analyzer
// It accept errors from streams and stream groups
func ProblemAnalyzer() {

}

// Report found problems from problem analyzer
// Maybe several methods to report problems of different prirority
func ProblemReporter() {
}
