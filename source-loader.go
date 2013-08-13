// Load configuration
package main

import (
	"bufio"
	"fmt"
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
}

// Internal config structure parsed from YAML
type config struct {
	StreamsHLS     map[string][]string `yaml:"hls-stream,omitempty"`       // HLS parsing
	StreamsHTTP    map[string][]string `yaml:"http-streams,omitempty"`     // plain HTTP checks
	GetStreamsHLS  []string            `yaml:"get-hls-streams,omitempty"`  // load remote HLS-checks configuration
	GetStreamsHTTP []string            `yaml:"get-http-streams,omitempty"` // load remote HTTP-checks configuration
	Samples        []string            `yaml:"samples"`
	Params         Params              `yaml:"params"`
	//	GroupParams    Params              `yaml:"group-params,omitempty"` // parameters per group
}

type Params struct {
	ProbersHTTP            uint          `yaml:"http-probers"`              // num of
	ProbersHLS             uint          `yaml:"hls-probers"`               // num of
	MediaProbers           uint          `yaml:"media-probers"`             // num of
	ConnectTimeout         time.Duration `yaml:"connect-timeout"`           // sec
	RWTimeout              time.Duration `yaml:"rw-timeout"`                // sec
	SlowWarningTimeout     time.Duration `yaml:"slow-warning-timeout"`      // sec
	VerySlowWarningTimeout time.Duration `yaml:"very-slow-warning-timeout"` // sec
	TimeBetweenTasks       time.Duration `yaml:"time-between-tasks"`        // ms
	TryOneSegment          bool          `yaml:"one-segment"`
	ListenHTTP             string        `yaml:"http-api-listen"`
	ErrorLog               string        `yaml:"error-log"`
	ZabbixDiscoveryPath    string        `yaml:"zabbix-discovery-path,omitempty"`
	ZabbixDiscoveryGroups  []string      `yaml:"zabbix-discovery-groups,omitempty"`
	//	User                   string        `yaml:"user,omitempty"`
	//	Pass                   string        `yaml:"pass,omitempty"`
}

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
		for groupName, streamList := range cfg.StreamsHLS {
			addLocalConfig(&Cfg.StreamsHLS, HLS, groupName, streamList)
		}
		for groupName, streamList := range cfg.StreamsHTTP {
			addLocalConfig(&Cfg.StreamsHTTP, HTTP, groupName, streamList)
		}
		if cfg.GetStreamsHLS != nil {
			for _, source := range cfg.GetStreamsHLS {
				groupURI, groupName := splitName(source)
				err := addRemoteConfig(&Cfg.StreamsHLS, HLS, groupName, groupURI)
				if err != nil {
					fmt.Printf("Load remote config for group %s (HLS streams) failed.\n", groupName)
				}
			}
		}
		if cfg.GetStreamsHTTP != nil {
			for _, source := range cfg.GetStreamsHTTP {
				groupURI, groupName := splitName(source)
				err := addRemoteConfig(&Cfg.StreamsHTTP, HTTP, groupName, groupURI)
				if err != nil {
					fmt.Printf("Load remote config for group %s (HTTP streams) failed.\n", groupName)
				}
			}
		}
	} else {
		print("Config file not found. Hardcoded defaults used.\n")
	}
	return
}

// Helper. Split stream link to URI and Name parts.
func splitName(source string) (uri string, name string) {
	splitted := strings.SplitN(strings.TrimSpace(source), " ", 2)
	uri = splitted[0]
	if len(splitted) > 1 && splitted[1] != "" {
		name = strings.TrimSpace(splitted[1])
	} else {
		name = strings.SplitN(strings.TrimSpace(splitted[0]), "://", 2)[1]
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
func addRemoteConfig(dest *[]Stream, streamType StreamType, group, uri string) error {
	client := NewTimeoutClient(20*time.Second, 20*time.Second)
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return err
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
	return err
}
