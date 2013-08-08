// The code keeps streams statistics and program internal statistics.
// Statistics output to files and to JSON HTTP API.
package main

import (
	"time"
)

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

// Elder
func StatKeeper(cfg *Config) {
	statq := make(chan Stats, 1024)
	<-statq
}
