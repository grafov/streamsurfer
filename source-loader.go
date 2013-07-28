package main

import (
	"fmt"
	"io/ioutil"
	"launchpad.net/goyaml"
	"os"
	"strings"
)

// Parse config and run stream monitors
func SourceLoader(confile string) {
	ReadConfig(confile)
}

func ReadConfig(confile string) (cfg config) {
	/*	cfg = config{
			streams:  stream{name: "localhost:1234"},
			workers:  workers{1, 1},
			timeouts: timeouts{12, 12},
			options:  options{tryOneSegment: true},
		}
	*/
	//cfg = make(map[string]interface{})
	// TODO второй конфиг из /etc/hlsproberc
	if confile == "" {
		confile = "~/.hlsproberc"
	}
	data, e := ioutil.ReadFile(os.ExpandEnv(strings.Replace(confile, "~", os.Getenv("HOME"), 1)))
	if e == nil {
		e = goyaml.Unmarshal(data, &cfg)
		fmt.Printf("%+v", cfg)
	} else {
		print("Config file not found or parsing failed. Hardcoded defaults used.\n")
	}
	/*	for k, v := range cfg {
			switch vv := v.(type) {
			case string:
				fmt.Println(k, "is string", vv)
			case int:
				fmt.Println(k, "is int", vv)
			case map[string]interface{}:
				fmt.Println(k, "is an config:")
				for i, u := range vv {
					fmt.Println(i, u)
				}
			case map[string]interface{}:
				fmt.Println(k, "is an dict:")
				for i, u := range vv {
					fmt.Println(i, u)
				}
			case []interface{}:
				fmt.Println(k, "is an array:")
				for i, u := range vv {
					fmt.Println(i, u)
				}
			case map[interface{}]interface{}:
				fmt.Println(k, "is an map:")
				for i, u := range vv {
					fmt.Println(i, u)
				}
			default:
				fmt.Printf("%s is of a type %s I don't know how to handle\n", k, v)
			}
		}
	*/
	return
}

/* Datatypes */

type config struct {
	streams  []string
	workers  workers `yaml:"workers"`
	timeouts timeouts
	options  options
}

//type stream map[string]interface{}
type stream struct {
	URI   string
	Title string // optional title of stream or mandatory title of a group
	//members *stream // link to stream group members (nil if the item is a stream not a group)
}

type workers struct {
	streamProbers uint `yaml:"stream-probers,omitempty"`
	mediaProbers  uint `yaml:"media-probers"`
}

type timeouts struct {
	playlistRead int
	mediaRead    int
}

type options struct {
	tryOneSegment bool
}
