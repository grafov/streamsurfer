package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
)

const (
	VERSION = "0.X"
)

func main() {
	var confname = flag.String("config", "", "alternative configuration file")

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Stream Surfer trace dumped:", r)
			if err := ioutil.WriteFile(FullPath("~/hlsprobe2.trace"), r.([]byte), 0644); err != nil {
				fmt.Println("Can't write trace file!")
			}
		}
	}()

	fmt.Printf("Stream Surfer vers. %s\n", VERSION)
	flag.Parse()

	//cfgq := make(chan ConfigQuery, 12)
	//go SourceLoader(*config, cfgq)
	cfg := ReadConfig(*confname)

	go LogKeeper(cfg)
	go StatKeeper(cfg)
	go StreamMonitor(cfg)
	go ZabbixDiscoveryFile(cfg)
	go HttpAPI(cfg)

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
