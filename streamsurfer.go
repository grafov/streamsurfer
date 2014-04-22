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

/* TODO

1. Для ассетов выдавать общий список одной страницей (плюс по группам). Ассет, график статусов за последнее время.
2. Страница инфы по отдельному ассету.

*/

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
)

const (
	SURFER  = "Stream Surfer"
	VERSION = "0.3-dev"
)

var build_date string
var Stubs = &configStub{}

func main() {
	var confname = flag.String("config", "", "alternative configuration file")
	var verbose = flag.Bool("verbose", true, "verbose output of logs")

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Stream Surfer trace dumped:", r)
			if err := ioutil.WriteFile(FullPath("~/streamsurfer.trace"), r.([]byte), 0644); err != nil {
				fmt.Println("Can't write trace file!")
			}
		}
	}()

	fmt.Printf("Stream Surfer vers. %s (build %s)\n", VERSION, build_date)
	flag.Parse()

	InitConfig(*confname)

	go ConfigKeeper()
	go LogKeeper(*verbose) // collect program logs and write them to file
	go StatKeeper()        // collect probe statistics for report builders
	go StreamMonitor()     // probe logic
	go HttpAPI()           // control API
	go ProblemAnalyzer()   // analyze problems related to groups of channels
	go ProblemReporter()   // report problems to email

	terminate := make(chan os.Signal)
	signal.Notify(terminate, os.Interrupt)
	<-terminate
	fmt.Println("...probe service interrupted.")
}
