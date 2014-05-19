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
					startpoint := time.Now().Add(-12 * time.Hour)
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
func analyzeHLS(key Key, hist []KeepedResult, lastCheck *CheckPoint) (reports []Report) {
	var (
		isProblem           bool                    // problem detected in the stream
		isCheckOK, isTaskOK bool       = true, true // statuses for current check and task
		start, stop         time.Time               // start and stop timestamps of error period
		fromTid, toTid, tid int64                   // task id
		errlevel            ErrType                 // error level
		errorRanges         []ErrRange              // ranges with error states of the stream
		forSave             *ErrRange               // continious range of failed tasks
	)

	for _, hitem := range hist {
		if hitem.Started.Before(lastCheck.Occured) {
			continue
		}

		/*
			We check for an each check in an each task in results set. Task interpreted as problem if error occured in
			the any single check. Then we aggregate problem tasks into error ranges ([]ErrRange). Report builder then
			analyze error ranges and make reports on them.
		*/
		if hitem.ErrType > ERROR_LEVEL {
			isCheckOK = false
			if errlevel < hitem.ErrType {
				errlevel = hitem.ErrType
			}

			if !isProblem { // range started
				fromTid = hitem.Tid
				toTid = hitem.Tid
				start = hitem.Started
				stop = hitem.Started.Add(hitem.Elapsed)
				isProblem = true
			} else { // range continued
				toTid = hitem.Tid
				stop = hitem.Started.Add(hitem.Elapsed)
			}

			if isProblem {
				forSave = &ErrRange{fromTid, toTid, start, stop, errlevel}
			}
		}

		if tid != 0 && tid == hitem.Tid { // это следующая проверка в той же задаче
			if !isCheckOK {
				isTaskOK = false
			}
		}

		if tid != 0 && tid != hitem.Tid { // обнаружена смена задачи
			if isTaskOK {
				isProblem = false
				isCheckOK = true
				if forSave != nil {
					errorRanges = append(errorRanges, *forSave)
				}
			}
			isTaskOK = true
		}

		tid = hitem.Tid

		// // TODO учитывать дырки в мониторинге, между проверками должно быть не более 10 мин., иначе период закрывается
		// if hitem.ErrType > ERROR_LEVEL {
		// 	if isProblem { // ошибки продолжаются, продление периода ошибок
		// 		stop = hitem.Started.Add(hitem.Elapsed)
		// 		toTid = hitem.Tid
		// 	} else { // начало периода с ошибками
		// 		start = hitem.Started
		// 		fromTid = hitem.Tid
		// 		toTid = fromTid
		// 		stop = hitem.Started.Add(hitem.Elapsed)
		// 		isProblem = true
		// 	}
		// 	if hitem.ErrType > errlevel { // записываем макс. уровень ошибок
		// 		errlevel = hitem.ErrType
		// 	}
		// } else {
		// 	if isProblem { // фиксация окончания периода ошибок
		// 		stop = hitem.Started.Add(hitem.Elapsed)
		// 		if markForSave != nil {
		// 			// 	if hitem.Tid != markForSave.ToTid { // последняя задача была с ошибками
		// 			// 		goodTask = true
		// 			// 	}
		// 			// if hitem.Tid != toTid && errlevel > 0 { // в задаче достаточно одной ошибочной проверки, для выставления статуса isProblem
		// 			// 	markForSave = &ErrRange{fromTid, hitem.Tid, start, stop, errlevel}
		// 			// }
		// 			// 	if errlevel > 0 {
		// 			// 		errorRanges = append(errorRanges, ErrRange{fromTid, hitem.Tid, start, stop, errlevel})
		// 			// 	}
		// 			// 	errlevel = 0 // reset range
		// 			// 	isProblem = false
		// 		}
		// 	}
		// }
		// if fromTid == 0 {
		// 	fromTid = hitem.Tid
		// }
		// toTid = hitem.Tid
	}
	lastCheck = &CheckPoint{toTid, stop}
	if isProblem && errlevel > 0 { // период остался незакрыт
		errorRanges = append(errorRanges, ErrRange{fromTid, toTid, start, stop, errlevel})
	}
	if len(errorRanges) > 0 {
		fmt.Printf("err range for %s %v\n", key.String(), errorRanges)
	}
	// Permanent errors report. Error is permanent if it continued more than 10 minute.
	for _, val := range errorRanges {
		if val.Discontinued.Sub(val.Occured) > 10*time.Minute {
			reports = append(reports, generatePermanentErrorsReport(key, errorRanges, isProblem))
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
