// Load configuration
package main

import (
	"bufio"
	//	"fmt"
	"io/ioutil"
	"launchpad.net/goyaml"
	"net/http"
	"strings"
	"time"
)

// Exported config structure
type Config struct {
	StreamsHLS  []Stream
	StreamsHTTP []Stream
	Samples     []string
	Params      Params
	Options     Options
}

// Internal config structure parsed from YAML
type config struct {
	StreamsHLS     map[string][]string `yaml:"hls-streams"`      // потоки для проверки HLS
	StreamsHTTP    map[string][]string `yaml:"http-streams"`     // потоки для проверки только HTTP, без парсинга HLS
	GetStreamsHLS  []string            `yaml:"get-hls-streams"`  // ссылка на внешний список потоков HLS
	GetStreamsHTTP []string            `yaml:"get-http-streams"` // ссылка на внешний список потоков HTTP
	Samples        []string            `yaml:"samples"`
	Params         Params              `yaml:"params"`
	Options        Options             `yaml:"options"`
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

func ReadConfig(confile string) (Cfg *Config) {
	var cfg = &config{}

	Cfg = &Config{}
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
		Cfg.Params = cfg.Params
		Cfg.Options = cfg.Options
		for groupName, streamList := range cfg.StreamsHLS {
			addLocalConfig(&Cfg.StreamsHLS, HLS, groupName, streamList)
		}
		for groupName, streamList := range cfg.StreamsHTTP {
			addLocalConfig(&Cfg.StreamsHTTP, HTTP, groupName, streamList)
		}
		if cfg.GetStreamsHLS != nil {
			for _, source := range cfg.GetStreamsHLS {
				groupURI, groupName := splitName(source)
				addRemoteConfig(&Cfg.StreamsHLS, HLS, groupName, groupURI)
			}
		}
		if cfg.GetStreamsHTTP != nil {
			for _, source := range cfg.GetStreamsHTTP {
				groupURI, groupName := splitName(source)
				addRemoteConfig(&Cfg.StreamsHTTP, HTTP, groupName, groupURI)
			}
		}
	} else {
		print("Config file not found. Hardcoded defaults used.\n")
	}
	return
}

// Helper. Split stream link to URI and Name parts.
func splitName(source string) (uri string, name string) {
	splitted := strings.SplitN(source, " ", 2)
	uri = splitted[0]
	if len(splitted) > 1 && splitted[1] != "" {
		name = splitted[1]
	} else {
		name = strings.SplitN(splitted[0], "://", 2)[1]
	}
	return
}

// Helper. Parse config of
func addLocalConfig(dest *[]Stream, streamType StreamType, group string, sources []string) {
	for _, source := range sources {
		uri, name := splitName(source)
		*dest = append(*dest, Stream{URI: uri, Type: streamType, Name: name, Group: group})
	}
}

// Helper. Get remote list of streams.
func addRemoteConfig(dest *[]Stream, streamType StreamType, group, uri string) {
	client := NewTimeoutClient(5*time.Second, 5*time.Second)
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return
	}
	result, err := client.Do(req)
	if err == nil {
		body := bufio.NewReader(result.Body)
		for {
			line, err := body.ReadString('\n')
			if err != nil {
				break
			}
			uri, name := splitName(line)
			*dest = append(*dest, Stream{URI: uri, Type: streamType, Name: name, Group: group})
		}
	}
}
