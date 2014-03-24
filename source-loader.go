// Load configuration file.
package main

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"launchpad.net/goyaml"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Exported config structure
type Config struct {
	StreamsHLS  []Stream
	StreamsHDS  []Stream
	StreamsHTTP []Stream
	StreamsWV   []Stream
	GroupsHLS   map[string]string // map[group]group
	GroupsHDS   map[string]string // map[group]group
	GroupsHTTP  map[string]string // map[group]group
	GroupsWV    map[string]string // map[group]group
	Groups      map[Group][]string
	Samples     []string
	Params      Params
	GroupParams map[string]Params
	m           sync.Mutex
}

// Internal config structure parsed from YAML
type configYAML struct {
	StreamsHLS     map[string][]string `yaml:"hls-streams,omitempty"`      // HLS parsing
	StreamsHDS     map[string][]string `yaml:"hds-streams,omitempty"`      // HDS parsing
	StreamsHTTP    map[string][]string `yaml:"http-streams,omitempty"`     // plain HTTP checks
	StreamsWV      map[string][]string `yaml:"wv-streams,omitempty"`       // HTTP checks with additional WV VOD checks
	GetStreamsHLS  []string            `yaml:"get-hls-streams,omitempty"`  // load remote HLS checks configuration
	GetStreamsHDS  []string            `yaml:"get-hds-streams,omitempty"`  // load remote HLS checks configuration
	GetStreamsHTTP []string            `yaml:"get-http-streams,omitempty"` // load remote HTTP checks configuration
	GetStreamsWV   []string            `yaml:"get-wv-streams,omitempty"`   // load remote WV VOD checks configuration
	Samples        []string            `yaml:"samples"`
	Params         Params              `yaml:"params"`
	GroupParams    map[string]Params   `yaml:"group-params,omitempty"` // parameters per group
	Stubs          StubValues          `yaml:"stubs"`
}

type Params struct {
	ProbersHTTP            uint          `yaml:"http-probers,omitempty"`              // num of
	ProbersHLS             uint          `yaml:"hls-probers,omitempty"`               // num of
	ProbersHDS             uint          `yaml:"hds-probers,omitempty"`               // num of
	ProbersWV              uint          `yaml:"wv-probers,omitempty"`                // num of
	MediaProbers           uint          `yaml:"media-probers,omitempty"`             // num of
	CheckBrokenTime        uint          `yaml:"check-broken-time"`                   // ms
	ConnectTimeout         time.Duration `yaml:"connect-timeout,omitempty"`           // sec
	RWTimeout              time.Duration `yaml:"rw-timeout,omitempty"`                // sec
	SlowWarningTimeout     time.Duration `yaml:"slow-warning-timeout,omitempty"`      // sec
	VerySlowWarningTimeout time.Duration `yaml:"very-slow-warning-timeout,omitempty"` // sec
	TimeBetweenTasks       time.Duration `yaml:"time-between-tasks,omitempty"`        // sec
	TaskTTL                time.Duration `yaml:"task-ttl,omitempty"`                  // sec
	TryOneSegment          bool          `yaml:"one-segment,omitempty"`
	MethodHTTP             string        `yaml:"http-method,omitempty"` // GET, HEAD
	ListenHTTP             string        `yaml:"http-api-listen,omitempty"`
	ErrorLog               string        `yaml:"error-log,omitempty"`
	Zabbix                 Zabbix        `yaml:"zabbix,omitempty"`
	ParseName              string        `yaml:"parse-name,omitempty"` // regexp for alternative method of title/name parsing from the URL
	User                   string        `yaml:"user,omitempty"`
	Pass                   string        `yaml:"pass,omitempty"`
}

type Zabbix struct {
	DiscoveryPath   string   `yaml:"discovery-path,omitempty"`
	DiscoveryGroups []string `yaml:"discovery-groups,omitempty"`
	StreamTemplate  string   `yaml:"stream-template,omitempty"`
}

// custom values for HTML-templates and reports
type StubValues struct {
	Name string `yaml:"name,omitempty"`
}

// глобальный конфиг
var cfg *Config

// XXX выпилить
//var NameParseMode string // regexp for parse name/title from the stream URL, by default string splitted by space and http:// part becomes URL other part becomes stream name

func ReadConfig(confile string) *Config {
	var cfg = &configYAML{}

	// Hardcoded defaults:
	cfg.Stubs = StubValues{Name: "Stream Surfer"}
	// Final config:
	Cfg := new(Config)
	if confile == "" {
		confile = "/etc/streamsurfer/default.yaml"
	}
	data, e := ioutil.ReadFile(FullPath(confile))
	if e == nil {
		e = goyaml.Unmarshal(data, &cfg)
		if e != nil {
			print("Config file parsing failed. Hardcoded defaults used.\n")
		}
		Cfg.Groups = make(map[Group][]string) // TODO свести использование к Cfg.Groups, убрать GroupsHLS, GroupsHDS, GroupsHTTP
		Cfg.GroupsHLS = make(map[string]string)
		Cfg.GroupsHDS = make(map[string]string)
		Cfg.GroupsHTTP = make(map[string]string)
		Cfg.GroupsWV = make(map[string]string)
		Cfg.Params = cfg.Params
		Cfg.Samples = cfg.Samples
		Cfg.Params.MethodHTTP = strings.ToUpper(cfg.Params.MethodHTTP)
		Stubs = &cfg.Stubs

		Cfg.GroupParams = make(map[string]Params)
		Cfg.GroupParams = cfg.GroupParams

		// for group, params := range cfg.GroupParams {
		// 	if
		// 	Cfg.GroupParams[group] = cfg.Params
		// 	fmt.Printf("%s %v\n", group, params)
		// 	// if params.ProbersHTTP != 0 {
		// 	// 	Cfg.GroupParams[group].ProbersHTTP = params.ProbersHTTP
		// 	// }
		// 	// if params.ProbersHLS != 0 {
		// 	// 	Cfg.GroupParams[group].ProbersHLS = params.ProbersHLS
		// 	// }
		// 	// if cfg.GroupParams[group].ProbersHDS != 0 {
		// 	// 	Cfg.GroupParams[group].ProbersHDS = cfg.GroupParams[group].ProbersHDS
		// 	// }
		// 	// if cfg.GroupParams[group].ProbersWV != 0 {
		// 	// 	Cfg.GroupParams[group].ProbersWV = cfg.GroupParams[group].ProbersWV
		// 	// }
		// 	if params.ParseName != "" {
		// 		Cfg.GroupParams[group].ParseName = params.ParseName
		// 	}
		// 	// if cfg.GroupParams[group].User != "" {
		// 	// 	Cfg.GroupParams[group].User = cfg.GroupParams[group].User
		// 	// 	Cfg.GroupParams[group].Pass = cfg.GroupParams[group].Pass
		// 	// }
		// }

		for groupName, streamList := range cfg.StreamsHLS {
			nameList := addLocalConfig(&Cfg.StreamsHLS, HLS, groupName, streamList)
			Cfg.GroupsHLS[groupName] = groupName
			Cfg.Groups[Group{HLS, groupName}] = nameList
		}
		for groupName, streamList := range cfg.StreamsHDS {
			nameList := addLocalConfig(&Cfg.StreamsHDS, HDS, groupName, streamList)
			Cfg.GroupsHDS[groupName] = groupName
			Cfg.Groups[Group{HDS, groupName}] = nameList
		}
		for groupName, streamList := range cfg.StreamsHTTP {
			nameList := addLocalConfig(&Cfg.StreamsHTTP, HTTP, groupName, streamList)
			Cfg.GroupsHTTP[groupName] = groupName
			Cfg.Groups[Group{HTTP, groupName}] = nameList
		}

		for groupName, streamList := range cfg.StreamsWV {
			nameList := addLocalConfig(&Cfg.StreamsWV, WV, groupName, streamList)
			Cfg.GroupsWV[groupName] = groupName
			Cfg.Groups[Group{WV, groupName}] = nameList
		}

		if cfg.GetStreamsHLS != nil {
			for _, source := range cfg.GetStreamsHLS {
				groupURI, groupName := splitName("", source)
				remoteUser := ""
				remotePass := ""
				if _, exists := cfg.GroupParams[groupName]; exists {
					remoteUser = cfg.GroupParams[groupName].User
					remotePass = cfg.GroupParams[groupName].Pass
				}
				nameList, err := addRemoteConfig(&Cfg.StreamsHLS, HLS, groupName, groupURI, remoteUser, remotePass)
				if err != nil {
					fmt.Printf("Load remote config for group \"%s\" (HLS) failed.\n", groupName)
				} else {
					Cfg.GroupsHLS[groupName] = groupName
					Cfg.Groups[Group{HLS, groupName}] = nameList
				}
			}
		}
		fmt.Printf("%+v\n", Cfg.GroupParams)
		if cfg.GetStreamsHDS != nil {
			for _, source := range cfg.GetStreamsHDS {
				groupURI, groupName := splitName("", source)
				remoteUser := ""
				remotePass := ""
				if _, exists := cfg.GroupParams[groupName]; exists {
					remoteUser = cfg.GroupParams[groupName].User
					remotePass = cfg.GroupParams[groupName].Pass
				}
				nameList, err := addRemoteConfig(&Cfg.StreamsHDS, HDS, groupName, groupURI, remoteUser, remotePass)
				if err != nil {
					fmt.Printf("Load remote config for group \"%s\" (HDS) failed.\n", groupName)
				} else {
					Cfg.GroupsHDS[groupName] = groupName
					Cfg.Groups[Group{HDS, groupName}] = nameList
				}
			}
		}
		if cfg.GetStreamsHTTP != nil {
			for _, source := range cfg.GetStreamsHTTP {
				groupURI, groupName := splitName("", source)
				remoteUser := ""
				remotePass := ""
				if _, exists := cfg.GroupParams[groupName]; exists {
					remoteUser = cfg.GroupParams[groupName].User
					remotePass = cfg.GroupParams[groupName].Pass
				}
				nameList, err := addRemoteConfig(&Cfg.StreamsHTTP, HTTP, groupName, groupURI, remoteUser, remotePass)
				if err != nil {
					fmt.Printf("Load remote config for group \"%s\" (HTTP) failed.\n", groupName)
				} else {
					Cfg.GroupsHTTP[groupName] = groupName
					Cfg.Groups[Group{HTTP, groupName}] = nameList
				}
			}
		}
		if cfg.GetStreamsWV != nil {
			for _, source := range cfg.GetStreamsWV {
				groupURI, groupName := splitName("", source)
				remoteUser := ""
				remotePass := ""
				if _, exists := cfg.GroupParams[groupName]; exists {
					remoteUser = cfg.GroupParams[groupName].User
					remotePass = cfg.GroupParams[groupName].Pass
				}
				nameList, err := addRemoteConfig(&Cfg.StreamsWV, WV, groupName, groupURI, remoteUser, remotePass)
				if err != nil {
					fmt.Printf("Load remote config for group \"%s\" (WV) failed.\n", groupName)
				} else {
					Cfg.GroupsWV[groupName] = groupName
					Cfg.Groups[Group{WV, groupName}] = nameList
				}
			}
		}

		for group, groupParams := range cfg.GroupParams {
			if groupParams.ProbersHLS == 0 {
				groupParams.ProbersHLS = cfg.Params.ProbersHLS
			}
			if groupParams.ProbersHDS == 0 {
				groupParams.ProbersHDS = cfg.Params.ProbersHDS
			}
			if groupParams.ProbersHTTP == 0 {
				groupParams.ProbersHTTP = cfg.Params.ProbersHTTP
			}
			if groupParams.ProbersWV == 0 {
				groupParams.ProbersWV = cfg.Params.ProbersWV
			}
			if groupParams.MediaProbers == 0 {
				groupParams.MediaProbers = cfg.Params.MediaProbers
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
			if groupParams.TaskTTL == 0 {
				groupParams.TaskTTL = cfg.Params.TaskTTL
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

	return Cfg
}

// Helper. Split stream link to URI and Name parts.
// Supported both cases: title<space>uri and uri<space>title
// URI must be prepended by http:// or https://
func splitName(group, source string) (uri string, name string) {
	source = strings.TrimSpace(source)
	sep := regexp.MustCompile("htt(p|ps)://")
	loc := sep.FindStringIndex(source)
	if loc != nil {
		if loc[0] == 0 { // uri title
			splitted := strings.SplitN(source, " ", 2)
			if len(splitted) > 1 {
				name = strings.TrimSpace(splitted[1])
			}
			uri = strings.TrimSpace(splitted[0])
		} else { // title uri
			name = strings.TrimSpace(source[0:loc[0]])
			uri = source[loc[0]:]
		}
		if name == "" {
			name = uri
		}
	}
	if group == "" {
		return // для парсинга имён групп в YAML-конфиге не применяется парсинг по регвырам
	}
	if params, err := cfg.Params4(group); err == nil && params.ParseName != "" {
		re := regexp.MustCompile(params.ParseName)
		vals := re.FindStringSubmatch(uri)
		if len(vals) > 1 {
			name = vals[1]
		}
	}
	return
}

// Helper. Parse config of
func addLocalConfig(dest *[]Stream, streamType StreamType, group string, sources []string) []string {
	var nameList []string

	for _, source := range sources {
		uri, name := splitName(group, source)
		*dest = append(*dest, Stream{URI: uri, Type: streamType, Name: name, Group: group})
		nameList = append(nameList, name)
	}

	return nameList
}

// Helper. Get remote list of streams.
func addRemoteConfig(dest *[]Stream, streamType StreamType, group string, uri, remoteUser, remotePass string) ([]string, error) {
	var nameList []string

	defer func() error {
		if r := recover(); r != nil {
			return errors.New(fmt.Sprintf("Can't get remote config for (%s) %s %s", streamType, group, uri))
		}
		return nil
	}()

	client := NewTimeoutClient(10*time.Second, 10*time.Second)
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return nil, err
	}
	result, err := client.Do(req)
	if err == nil {
		body := bufio.NewReader(result.Body)
		for {
			line, err := body.ReadString('\n')
			if err != nil {
				break
			}
			uri, name := splitName(group, line)
			nameList = append(nameList, name)
			*dest = append(*dest, Stream{URI: uri, Type: streamType, Name: name, Group: group})
		}
	}
	return nameList, err
}

// TODO Dynamic configuration without program restart.
// Elder.
func ConfigKeeper(confile string) {
	cfg = new(Config)
	cfg = ReadConfig(confile)
	select {} // TODO reload config by query
}

func (cfg *Config) Params4(group string) (*Params, error) {
	cfg.m.Lock()
	defer cfg.m.Unlock()
	if val, ok := cfg.GroupParams[group]; !ok {
		return &cfg.Params, nil
	} else {
		return &val, nil
	}
}
