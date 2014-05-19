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

/*
 * Public API
 */

// Анализирует проблемы связей между отдельными потоками и группы потоков на серверах.
func ProblemAnalyzer() {
	var (
		lastAnalyzed map[Key]CheckPoint = make(map[Key]CheckPoint)
		key          Key
		conclusion   []Report
	)

	for {
		time.Sleep(10 * time.Second)
		for gname, gdata := range cfg.GroupParams {
			for _, stream := range *cfg.GroupStreams[gname] {
				key = Key{gname, stream.Name}
				if _, ok := lastAnalyzed[key]; !ok {
					startpoint := time.Now().Add(-2 * time.Hour)
					lastAnalyzed[key] = CheckPoint{0, startpoint}
				}
				checkPoint := lastAnalyzed[key]
				hist, err := LoadHistoryResults(key) // TODO загружать только результаты после времени lastAnalyzed
				if err != nil {
					continue
				}
				switch gdata.Type {
				case HLS:
					conclusion = analyzeHLS(key, hist, &checkPoint)
				case HDS:
					conclusion = analyzeHDS(hist)
				case HTTP:
					conclusion = analyzeHTTP(hist)
				}
				lastAnalyzed[key] = checkPoint
				// TODO save new checkpoint to DB
			}
			if len(conclusion) > 0 {
				fmt.Printf("conclusion for %s %v\n", key.String(), conclusion)
			}
			// TODO дополнительно делать анализ для всей группы
		}
		// TODO RemoveExpiredReports(cfg.ExpireDurationDB)
	}
}

// Report found problems from problem analyzer
// Maybe several methods to report problems of different prirority
func ProblemReporter() {

}

func LoadReports() []Report {
	return nil
}

/*
 * Private realization
 */

// Analyze HLS and form report with error states.
// We check for an each check in an each task in results set. Task interpreted as problem if error occured in
// the any single check. Then we aggregate problem tasks into error ranges ([]ErrRange). Report builder then
// analyze error ranges and make reports on them.
func analyzeHLS(key Key, hist []KeepedResult, lastCheck *CheckPoint) (reports []Report) {
	var (
		isRangeOpened           bool              // problem under analyzator cursor
		isTaskOK                bool       = true // statuses for current check and task
		start, stop             time.Time         // start and stop timestamps of error period
		prevTid, fromTid, toTid int64             // task id
		errlevel                ErrType           // error level
		errorRanges             []ErrRange        // ranges with error states of the stream
		forSave                 *ErrRange         // continious range of failed tasks
	)

	for _, hitem := range hist {
		if hitem.Started.Before(lastCheck.Occured) {
			continue
		}

		if key.Name == "sd_2014_game_of_thrones_04_02_film" {
			fmt.Printf("+++ %d %d %d %d %s %d %+v\n", prevTid, hitem.Tid, fromTid, toTid, isTaskOK, errlevel, forSave)
		}

		if prevTid > 0 && prevTid != hitem.Tid { // переход задач
			if isTaskOK && forSave != nil { // период ошибок кончился
				errorRanges = append(errorRanges, *forSave)
				forSave = nil
				isRangeOpened = false
			} else {
				isTaskOK = true
			}
			prevTid = hitem.Tid
		}

		if prevTid == 0 {
			prevTid = hitem.Tid
		}

		if hitem.ErrType > ERROR_LEVEL {
			isTaskOK = false
			if hitem.ErrType > errlevel {
				errlevel = hitem.ErrType
			}
			if !isRangeOpened {
				isRangeOpened = true
				fromTid = hitem.Tid
				start = hitem.Started
				toTid = fromTid
				stop = start
			} else {
				toTid = hitem.Tid
				stop = hitem.Started.Add(hitem.Elapsed)
			}
			forSave = &ErrRange{fromTid, hitem.Tid, start, stop, errlevel}
		}
	}

	lastCheck = &CheckPoint{toTid, stop}
	if isRangeOpened && errlevel > 0 { // период остался незакрыт
		errorRanges = append(errorRanges, ErrRange{fromTid, toTid, start, stop, errlevel})
	}
	if len(errorRanges) > 0 {
		fmt.Printf("err range for %s %v\n", key.String(), errorRanges)
	}
	// Permanent errors report. Error is permanent if it continued more than 10 minute.
	for _, val := range errorRanges {
		if val.Discontinued.Sub(val.Occured) > 10*time.Minute {
			reports = append(reports, generatePermanentErrorsReport(key, errorRanges, isRangeOpened))
		}
	}
	return reports
}

func analyzeHDS(hist []KeepedResult) []Report {
	return []Report{}
}

func analyzeHTTP(hist []KeepedResult) []Report {
	return []Report{}
}

/*
 * Report generators
 */

// Permanent errors report generator
func generatePermanentErrorsReport(key Key, ranges []ErrRange, errorPersists bool) Report {
	return Report{Title: "Sample report"}
}
