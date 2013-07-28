package main

import (
	"flag"
	"github.com/grafov/m3u8"
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
	var config = flag.String("config", "", "alternative configuration file")

	print("HLS Probe vers. ")
	print(VERSION)
	print("\n")
	flag.Parse()

	go SourceLoader(*config)
	go HttpAPI()

	terminate := make(chan os.Signal)
	signal.Notify(terminate, os.Interrupt)
	<-terminate
	print("...probe service interrupted.\n")
}

// Control monitoring of a single stream
func StreamMonitor() {

}

// Parse and probe M3U8 playlists (multi- and single bitrate)
// and report time statistics and errors
func CupertinoProbe() {
	m3u8.NewMasterPlaylist()

}

// Parse and probe media chunk
// and report time statistics and errors
func MediaProbe() {

}

// Top level problem analyzer
// It accept errors from streams and stream groups
func ProblemAnalyzer() {

}

// Report found problems from problem analyzer
// Maybe several methods to report problems of different prirority
func ProblemReporter() {
}

// Offers HTTP REST API to control probe service and view statistics
func HttpAPI() {
}
