// Load configuration
package main

import (
	"bufio"
	"errors"
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
	StreamsHDS  []Stream
	StreamsHTTP []Stream
	GroupsHLS   map[string]string // map[group]group
	GroupsHDS   map[string]string // map[group]group
	GroupsHTTP  map[string]string // map[group]group
	Samples     []string
	Params      Params
	GroupParams map[string]Params
}

// Internal config structure parsed from YAML
type config struct {
	StreamsHLS     map[string][]string `yaml:"hls-streams,omitempty"`      // HLS parsing
	StreamsHDS     map[string][]string `yaml:"hds-streams,omitempty"`      // HDS parsing
	StreamsHTTP    map[string][]string `yaml:"http-streams,omitempty"`     // plain HTTP checks
	GetStreamsHLS  []string            `yaml:"get-hls-streams,omitempty"`  // load remote HLS-checks configuration
	GetStreamsHTTP []string            `yaml:"get-http-streams,omitempty"` // load remote HTTP-checks configuration
	Samples        []string            `yaml:"samples"`
	Params         Params              `yaml:"params"`
	GroupParams    map[string]Params   `yaml:"group-params,omitempty"` // parameters per group
}

type Params struct {
	ProbersHTTP            uint          `yaml:"http-probers,omitempty"`              // num of
	ProbersHLS             uint          `yaml:"hls-probers,omitempty"`               // num of
	ProbersHDS             uint          `yaml:"hds-probers,omitempty"`               // num of
	MediaProbers           uint          `yaml:"media-probers,omitempty"`             // num of
	CheckRepeatTime        uint          `yaml:"check-repeat-time"`                   // ms
	CheckBrokenTime        uint          `yaml:"check-broken-time"`                   // ms
	ConnectTimeout         time.Duration `yaml:"connect-timeout,omitempty"`           // sec
	RWTimeout              time.Duration `yaml:"rw-timeout,omitempty"`                // sec
	SlowWarningTimeout     time.Duration `yaml:"slow-warning-timeout,omitempty"`      // sec
	VerySlowWarningTimeout time.Duration `yaml:"very-slow-warning-timeout,omitempty"` // sec
	TimeBetweenTasks       time.Duration `yaml:"time-between-tasks,omitempty"`        // ms
	TryOneSegment          bool          `yaml:"one-segment,omitempty"`
	MethodHTTP             string        `yaml:"http-method,omitempty"` // GET, HEAD
	ListenHTTP             string        `yaml:"http-api-listen,omitempty"`
	ErrorLog               string        `yaml:"error-log,omitempty"`
	Zabbix                 Zabbix        `yaml:"zabbix,omitempty"`
	User                   string        `yaml:"user,omitempty"`
	Pass                   string        `yaml:"pass,omitempty"`
}

type Zabbix struct {
	DiscoveryPath   string   `yaml:"discovery-path,omitempty"`
	DiscoveryGroups []string `yaml:"discovery-groups,omitempty"`
	StreamTemplate  string   `yaml:"stream-template,omitempty"`
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
		Cfg.GroupsHLS = make(map[string]string)
		Cfg.GroupsHTTP = make(map[string]string)
		Cfg.Params = cfg.Params
		Cfg.Samples = cfg.Samples
		Cfg.GroupParams = map[string]Params{}
		Cfg.Params.MethodHTTP = strings.ToUpper(cfg.Params.MethodHTTP)
		for groupName, streamList := range cfg.StreamsHLS {
			addLocalConfig(&Cfg.StreamsHLS, HLS, groupName, streamList)
			Cfg.GroupsHLS[groupName] = groupName
		}
		for groupName, streamList := range cfg.StreamsHDS {
			addLocalConfig(&Cfg.StreamsHDS, HDS, groupName, streamList)
			Cfg.GroupsHDS[groupName] = groupName
		}
		for groupName, streamList := range cfg.StreamsHTTP {
			addLocalConfig(&Cfg.StreamsHTTP, HTTP, groupName, streamList)
			Cfg.GroupsHTTP[groupName] = groupName
		}
		if cfg.GetStreamsHLS != nil {
			for _, source := range cfg.GetStreamsHLS {
				groupURI, groupName := splitName(source)
				remoteUser := ""
				remotePass := ""
				if _, exists := cfg.GroupParams[groupName]; exists {
					remoteUser = cfg.GroupParams[groupName].User
					remotePass = cfg.GroupParams[groupName].Pass
				}
				err := addRemoteConfig(&Cfg.StreamsHLS, HLS, groupName, groupURI, remoteUser, remotePass)
				if err != nil {
					fmt.Printf("Load remote config for group \"%s\" (HLS) failed.\n", groupName)
				} else {
					Cfg.GroupsHLS[groupName] = groupName
				}
			}
		}
		if cfg.GetStreamsHTTP != nil {
			for _, source := range cfg.GetStreamsHTTP {
				groupURI, groupName := splitName(source)
				remoteUser := ""
				remotePass := ""
				if _, exists := cfg.GroupParams[groupName]; exists {
					remoteUser = "root"  //cfg.GroupParams[groupName].User
					remotePass = "zveri" //cfg.GroupParams[groupName].Pass
				}
				err := addRemoteConfig(&Cfg.StreamsHTTP, HTTP, groupName, groupURI, remoteUser, remotePass)
				if err != nil {
					fmt.Printf("Load remote config for group \"%s\" (HTTP) failed.\n", groupName)
				} else {
					Cfg.GroupsHTTP[groupName] = groupName
				}
			}
		}
		for group, groupParams := range cfg.GroupParams {
			if groupParams.ProbersHLS == 0 {
				groupParams.ProbersHLS = cfg.Params.ProbersHLS
			}
			if groupParams.ProbersHTTP == 0 {
				groupParams.ProbersHTTP = cfg.Params.ProbersHTTP
			}
			if groupParams.MediaProbers == 0 {
				groupParams.MediaProbers = cfg.Params.MediaProbers
			}
			if groupParams.CheckRepeatTime == 0 {
				groupParams.CheckRepeatTime = cfg.Params.CheckRepeatTime
			}
			if groupParams.CheckBrokenTime == 0 {
				groupParams.CheckBrokenTime = cfg.Params.CheckBrokenTime
			}
			if groupParams.ConnectTimeout == 0 {
				groupParams.ConnectTimeout = cfg.Params.ConnectTimeout
			}
			if groupParams.RWTimeout == 0 {
				groupParams.RWTimeout = cfg.Params.RWTimeout
			}
			if groupParams.SlowWarningTimeout == 0 {
				groupParams.SlowWarningTimeout = cfg.Params.SlowWarningTimeout
			}
			if groupParams.VerySlowWarningTimeout == 0 {
				groupParams.VerySlowWarningTimeout = cfg.Params.VerySlowWarningTimeout
			}
			if groupParams.TimeBetweenTasks == 0 {
				groupParams.TimeBetweenTasks = cfg.Params.TimeBetweenTasks
			}
			if groupParams.TryOneSegment {
				groupParams.TryOneSegment = cfg.Params.TryOneSegment
			}
			if groupParams.ListenHTTP == "" {
				groupParams.ListenHTTP = cfg.Params.ListenHTTP
			}
			if groupParams.ErrorLog == "" {
				groupParams.ErrorLog = cfg.Params.ErrorLog
			}
			Cfg.GroupParams[group] = groupParams
		}
	} else {
		print("Config file not found. Hardcoded defaults used.\n")
	}
	//	fmt.Printf("HLS: %+v\n\n", Cfg.StreamsHLS)
	//fmt.Printf("HTTP: %+v\n\n", Cfg.GroupsHTTP)

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
func addRemoteConfig(dest *[]Stream, streamType StreamType, group string, uri, remoteUser, remotePass string) error {
	defer func() error {
		if r := recover(); r != nil {
			return errors.New(fmt.Sprintf("Can't get remote config for (%s) %s %s", streamType, group, uri))
		}
		return nil
	}()

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
