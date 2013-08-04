package main

import (
	"flag"
	"os"
	"os/signal"
)

const (
	VERSION = "2.1"
)

/*
TODO
мониторы в рутинах вешаются на каждый поток
и на каждую группу потоков (и группы групп тоже) — по описанию конфига

мониторы сообщают об ошибках в канал ошибок
канал ошибок потока смотрит в свою группу
группа смотрит в верхнюю группу
группа верхнего уровня смотрит в общий анализатор ошибок (пирамида)

*/

func main() {
	var confname = flag.String("config", "", "alternative configuration file")

	print("HLS Probe vers. ")
	print(VERSION)
	print("\n")
	flag.Parse()

	//cfgq := make(chan ConfigQuery, 12)
	//go SourceLoader(*config, cfgq)
	cfg := ReadConfig(*confname)

	statq := make(chan Stats, 1024)
	go StatsKeeper(cfg, statq)
	go StreamMonitor(cfg)
	go HttpAPI(cfg)

	terminate := make(chan os.Signal)
	signal.Notify(terminate, os.Interrupt)
	<-terminate
	print("...probe service interrupted.\n")
}

// Top level problem analyzer
// It accept errors from streams and stream groups
func ProblemAnalyzer() {

}

// Report found problems from problem analyzer
// Maybe several methods to report problems of different prirority
func ProblemReporter() {
}
