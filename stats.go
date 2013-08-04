// The code keeps streams statistics and program internal statistics.
// Statistics output to files and to JSON HTTP API.
package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"time"
)

const layout = "Jan 2, 2006 at 3:04pm (MST)"

var (
	logq  chan LogMessage
	statq chan Stats
)

type Stats struct {
	Source    string
	Operation string
	Started   time.Time
	Elapsed   time.Duration
}

type LogMessage struct {
	Text string
	Task
	TaskResult
}

// Elder
func StatsKeeper(cfg *Config) {
	statq := make(chan Stats, 1024)
	<-statq
}

func LogKeeper(cfg *Config) {
	var skip error

	logq = make(chan LogMessage, 1024)
	logf, skip := os.Create(cfg.Params.ErrorLog)
	if skip != nil {
		fmt.Printf("Can't create file for error log. Error logging to file skiped.")
	}
	logw := bufio.NewWriter(logf)
	timeout := make(chan bool, 1)

	for {
		go func() {
			time.Sleep(1 * time.Second)
			timeout <- true
		}()

		select {
		case msg := <-logq:
			if skip == nil {
				// конвертировать и вывести
				logw.WriteString(msg.Started.Format(layout))
				logw.WriteRune('\t')
				logw.WriteString(strconv.Itoa(msg.HTTPCode))
				logw.WriteRune('\t')
				logw.WriteString(strconv.FormatInt(msg.ContentLength, 10))
				logw.WriteRune('\t')
				logw.WriteString(msg.Elapsed.String())
				logw.WriteRune('\t')
				logw.WriteString(msg.URI)
				logw.WriteRune('\n')
			}
		case <-timeout:
			if skip == nil {
				_ = logw.Flush()
			}
		}
	}
}

func Log(desc string, task Task, taskres TaskResult) {
	logq <- LogMessage{Text: desc, Task: task, TaskResult: taskres}
}
