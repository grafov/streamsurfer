package main

import (
	"fmt"
	"time"
)

type Report struct {
	Error          ErrType
	Severity       Severity
	Title          string
	Body           string
	RelatedGroups  []*Group
	RelatedStreams []*Stream
	RelatedTasks   []*Task
	Generated      time.Time
}

// Анализирует проблемы связей между отдельными потоками и группы потоков на серверах.
func ProblemAnalyzer() {
	//var analyzedTimes map[Key]time.Time
	var conclusion Report
	for {
		time.Sleep(30 * time.Second)
		for gname, gdata := range cfg.GroupParams {
			for _, stream := range *cfg.GroupStreams[gname] {
				hist, err := LoadHistoryResults(Key{gname, stream.Name})
				if err != nil {
					continue
				}
				switch gdata.Type {
				case HLS:
					conclusion = AnalyzeHLS(hist)
				case HDS:
					conclusion = AnalyzeHDS(hist)
				case HTTP:
					conclusion = AnalyzeHTTP(hist)
				}
			}
			fmt.Printf("%v\n", conclusion)
			// TODO дополнительно делать анализ для всей группы
		}
	}
}

func AnalyzeHLS(hist []KeepedResult) Report {
	return Report{}
}

func AnalyzeHDS(hist []KeepedResult) Report {
	return Report{}
}

func AnalyzeHTTP(hist []KeepedResult) Report {
	return Report{}
}

// Report found problems from problem analyzer
// Maybe several methods to report problems of different prirority
func ProblemReporter() {
}
