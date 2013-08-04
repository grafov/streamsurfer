// Load configuration
package main

import (
	"io/ioutil"
	"launchpad.net/goyaml"
	"os"
	"strings"
)

// Parse config and run stream monitors
// func SourceLoader(confile string) {
//	ReadConfig(confile)
//}

func ReadConfig(confile string) (cfg *Config) {
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
	data, e := ioutil.ReadFile(os.ExpandEnv(strings.Replace(confile, "~", os.Getenv("HOME"), 1)))
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

/* Datatypes */

type Config struct {
	StreamsHTTP []string `yaml:"hls-streams"`  // потоки для проверки HLS
	StreamsHLS  []string `yaml:"http-streams"` // потоки для проверки только HTTP, без парсинга HLS
	Samples     []string `yaml:"samples"`
	Workers     Workers  `yaml:"workers"`
	Options     Options  `yaml:"options"`
}

//type stream map[string]interface{}
type stream struct {
	URI   string
	Title string // optional title of stream or mandatory title of a group
	//members *stream // link to stream group members (nil if the item is a stream not a group)
}

type Workers struct {
	StreamProbers uint `yaml:"stream-probers,omitempty"`
	MediaProbers  uint `yaml:"media-probers"`
}

type Options struct {
	TryOneSegment bool `yaml:"one-segment"`
}
