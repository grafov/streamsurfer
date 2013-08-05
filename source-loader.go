// Load configuration
package main

import (
	"io/ioutil"
	"launchpad.net/goyaml"
	"time"
)

/* Datatypes */

type Config struct {
	StreamsHLS  []string `yaml:"hls-streams"`  // потоки для проверки HLS
	StreamsHTTP []string `yaml:"http-streams"` // потоки для проверки только HTTP, без парсинга HLS
	Samples     []string `yaml:"samples"`
	Params      Params   `yaml:"params"`
	Options     Options  `yaml:"options"`
}

//type stream map[string]interface{}
type stream struct {
	URI   string
	Title string // optional title of stream or mandatory title of a group
	//members *stream // link to stream group members (nil if the item is a stream not a group)
}

type Params struct {
	ProbersHTTP      uint          `yaml:"http-probers"`
	ProbersHLS       uint          `yaml:"hls-probers"`
	MediaProbers     uint          `yaml:"media-probers"`
	ConnectTimeout   time.Duration `yaml:"connect-timeout"`
	RWTimeout        time.Duration `yaml:"rw-timeout"`         // sec
	WarningTimeout   time.Duration `yaml:"warning-timeout"`    // sec
	TimeBetweenTasks time.Duration `yaml:"time-between-tasks"` // ms
	ListenHTTP       string        `yaml:"http-api-listen"`
	ErrorLog         string        `yaml:"error-log"`
}

type Options struct {
	TryOneSegment bool `yaml:"one-segment"`
}

// Parse config and run stream monitors
// func SourceLoader(confile string) {
//	ReadConfig(confile)
//}

func ReadConfig(confile string) (cfg *Config) {
	//cfg = &Config{StreamsHLS: make(map[string]interface{})}
	/*	cfg = config{
			Streams:  stream{name: "localhost:1234"},
			workers:  workers{1, 1},
			timeouts: timeouts{12, 12},
			Options:  Options{tryOneSegment: true},
		}
	*/
	cfg = &Config{}
	// TODO второй конфиг из /etc/hlsproberc
	if confile == "" {
		confile = "~/.hlsproberc"
	}
	data, e := ioutil.ReadFile(FullPath(confile))
	if e == nil {
		e = goyaml.Unmarshal(data, &cfg)
		if e != nil {
			print("Config file parsing failed. Hardcoded defaults used.\n")
		}
	} else {
		print("Config file not found. Hardcoded defaults used.\n")
	}
	return
}
