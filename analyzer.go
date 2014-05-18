package main

import (
	"fmt"
	"time"
)

/* TODO

хранить lastAnalyzed в виде [time, taskId] в базе
при старте problemanalyzer забирает карту lastanalyzed из бд
при проверке канала, если данных lastanalyzed нет, то ставится [now-6h, nil]
в анализаторе проходить по задачам, выбирать макс. errtype
если ошибка, то выставлять флаг ошибки и записывать время старта
идти по проверкам задачи записывая макс.флаг ошибки
при переходе к след. задаче смотреть, если ошибка устранилась, то записывать время окончания
сбрасывать флаг ошибки, записывать диапазон в список
затем проходить по диапазону, выбирая ошибки > минДлительностьДляОтчета → формировать отчет
меньшие периоды, если их > N за М времени рапортовать как сводный отчёт (warning level)
если <N за M времени, то не рапортовать
сохранять последнее проверенное время и tid в базу

*/

// Анализирует проблемы связей между отдельными потоками и группы потоков на серверах.
func ProblemAnalyzer() {
	var (
		lastAnalyzed map[Key]CheckPoint = make(map[Key]CheckPoint)
		conclusion   Report
	)

	for {
		time.Sleep(30 * time.Second)
		for gname, gdata := range cfg.GroupParams {
			for _, stream := range *cfg.GroupStreams[gname] {
				key := Key{gname, stream.Name}
				if _, ok := lastAnalyzed[key]; !ok {
					lastAnalyzed[key] = CheckPoint{0, time.Now().Add(-12 * time.Hour)}
				}
				hist, err := LoadHistoryResults(key) // TODO загружать только результаты после времени lastAnalyzed
				if err != nil {
					continue
				}
				switch gdata.Type {
				case HLS:
					conclusion, lastAnalyzed[key] = AnalyzeHLS(hist, lastAnalyzed[key])
				case HDS:
					conclusion = AnalyzeHDS(hist)
				case HTTP:
					conclusion = AnalyzeHTTP(hist)
				}
				// TODO save new checkpoint
			}
			fmt.Printf("%v\n", conclusion)
			// TODO дополнительно делать анализ для всей группы
		}
		// TODO RemoveExpiredReports(cfg.ExpireDurationDB)
	}
}

func AnalyzeHLS(hist []KeepedResult, lastCheck CheckPoint) (Report, CheckPoint) {
	var (
		report    Report
		isProblem bool
	)

	for _, hitem := range hist {
		if hitem.Started.Before(lastCheck.Occured) {
			continue
		}
		if hitem.ErrType > ERROR_LEVEL {
			if isProblem { // начало периода с ошибками

			} else {
				isProblem = true
			}
		}
	}
	return report, lastCheck
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

func LoadReports() []Report {
	return nil
}
