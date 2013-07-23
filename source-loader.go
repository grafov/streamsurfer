package main

import (
	"io/ioutil"
	"launchpad.net/goyaml"
	"os"
	"strings"
)

// Parse config and run stream monitors
func SourceLoader() {
	ReadConfig()
}

func ReadConfig() (cfg map[string]interface{}) {
	/*	cfg = config{
			streams:  stream{name: "localhost:1234"},
			workers:  workers{1, 1},
			timeouts: timeouts{12, 12},
			options:  options{tryOneSegment: true},
		}
	*/
	cfg = make(map[string]interface{})
	data, e := ioutil.ReadFile(os.ExpandEnv(strings.Replace("~/.hlsproberc", "~", os.Getenv("HOME"), 1)))
	if e == nil {
		e = goyaml.Unmarshal(data, &cfg)
	} else {
		print("Config file not found or parsing failed. Hardcoded defaults used.\n")
	}
	return
}

/* Datatypes */

type config struct {
	streams  stream
	workers  workers
	timeouts timeouts
	options  options
}

type stream struct {
	name    string  // name of stream group or stream URI
	members *stream // link to stream group members (nil if the item is a stream not a group)
}

type workers struct {
	streamProbers uint
	mediaProbers  uint
}

type timeouts struct {
	playlistRead int
	mediaRead    int
}

type options struct {
	tryOneSegment bool
}
