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
	"time"
)

// parsed and validated config data
type Config struct {
	GroupParams  map[string]*configGroup
	GroupStreams map[string]*[]Stream // map[groupname]stream
	Stubs        configStub
	Zabbix       configZabbix
	Samples      []string
	ListenHTTP   string
	ErrorLog     string
	IsReady      chan bool // config parsed and ready to use
}

// parsed grup config
type configGroup struct {
	Type                   StreamType
	Probers                int
	MediaProbers           int
	CheckBrokenTime        int
	ConnectTimeout         time.Duration
	RWTimeout              time.Duration
	SlowWarningTimeout     time.Duration
	VerySlowWarningTimeout time.Duration
	TimeBetweenTasks       time.Duration
	TaskTTL                time.Duration
	TryOneSegment          bool
	MethodHTTP             string
	ParseMethod            string
	User                   string
	Pass                   string
}

type configZabbix struct {
	DiscoveryPath   string   `yaml:"discovery-path,omitempty"`
	DiscoveryGroups []string `yaml:"discovery-groups,omitempty"`
	NameTemplate    string   `yaml:"name-template,omitempty"`
	TitleTemplate   string   `yaml:"title-template,omitempty"`
}

// custom values for HTML-templates and reports
type configStub struct {
	Name string `yaml:"name,omitempty"`
}

// raw config data
type configYAML struct {
	ListenHTTP string                     `yaml:"http-api-listen,omitempty"`
	Stubs      configStub                 `yaml:"stubs,omitempty"`
	Zabbix     configZabbix               `yaml:"zabbix,omitempty"`
	Samples    []string                   `yaml:"unmortal,omitempty"`
	Defaults   configGroupYAML            `yaml:"defaults,omitempty"`
	Groups     map[string]configGroupYAML `yaml:"groups,omitempty"`
}

// rawconfig group data
type configGroupYAML struct {
	Type                   string        `yaml:"type,omitempty"`
	URI                    string        `yaml:"streams-uri,omitempty"`               // external link list
	Streams                []string      `yaml:"streams,omitempty"`                   // link list
	Probers                int           `yaml:"probers,omitempty"`                   // num of
	MediaProbers           int           `yaml:"media-probers,omitempty"`             // num of
	CheckBrokenTime        int           `yaml:"check-broken-time"`                   // ms
	ConnectTimeout         time.Duration `yaml:"connect-timeout,omitempty"`           // sec
	RWTimeout              time.Duration `yaml:"rw-timeout,omitempty"`                // sec
	SlowWarningTimeout     time.Duration `yaml:"slow-warning-timeout,omitempty"`      // sec
	VerySlowWarningTimeout time.Duration `yaml:"very-slow-warning-timeout,omitempty"` // sec
	TimeBetweenTasks       time.Duration `yaml:"time-between-tasks,omitempty"`        // sec
	TaskTTL                time.Duration `yaml:"task-ttl,omitempty"`                  // sec
	TryOneSegment          bool          `yaml:"one-segment,omitempty"`
	MethodHTTP             string        `yaml:"http-method,omitempty"` // GET, HEAD
	ErrorLog               string        `yaml:"error-log,omitempty"`
	ParseMethod            string        `yaml:"parse-method,omitempty"` // regexp for alternative method of title/name parsing from the URL
	User                   string        `yaml:"user,omitempty"`
	Pass                   string        `yaml:"pass,omitempty"`
}

var cfg *Config

// TODO Dynamic configuration without program restart.
// Elder.
func InitConfig(confile string) {
	rawcfg := rawConfig(confile)
	cfg = new(Config)
	cfg.IsReady = make(chan bool, 1)
	parseOptionsConfig(rawcfg)
	parseGroupsConfig(rawcfg)
	cfg.IsReady <- true
}

// XXX for future
func ConfigKeeper() {
	<-cfg.IsReady
	select {} // TODO reload config by query
}

//
func (cfg *Config) Params(gname string) configGroup {
	if data, ok := cfg.GroupParams[gname]; ok {
		return *data
	} else {
		return configGroup{}
	}
}

//
func (cfg *Config) Streams() {

}

// Read raw config with YAML validation
func rawConfig(confile string) *configYAML {
	cfg := new(configYAML)

	// Hardcoded defaults:
	cfg.Stubs = configStub{Name: "Stream Surfer"}

	if confile == "" {
		confile = "/etc/streamsurfer.yaml"
	}
	data, e := ioutil.ReadFile(FullPath(confile))
	if e == nil {
		e = goyaml.Unmarshal(data, &cfg)
		if e != nil {
			print("Config file parsing failed. Hardcoded defaults used.\n")
		}
	}

	return cfg
}

//
func parseOptionsConfig(rawcfg *configYAML) {
	cfg.ListenHTTP = rawcfg.ListenHTTP
	cfg.Stubs = rawcfg.Stubs
	cfg.Zabbix = rawcfg.Zabbix
	cfg.Samples = rawcfg.Samples
}

//
func parseGroupsConfig(rawcfg *configYAML) {
	cfg.GroupParams = make(map[string]*configGroup)
	cfg.GroupStreams = make(map[string]*[]Stream)

	for gname, gdata := range rawcfg.Groups {
		stype := String2StreamType(gdata.Type)
		cfg.GroupParams[gname] = &configGroup{
			Type:                   stype,
			Probers:                gdata.Probers,
			MediaProbers:           gdata.MediaProbers,
			CheckBrokenTime:        gdata.CheckBrokenTime,
			ParseMethod:            gdata.ParseMethod,
			TimeBetweenTasks:       gdata.TimeBetweenTasks,
			ConnectTimeout:         gdata.ConnectTimeout,
			RWTimeout:              gdata.RWTimeout,
			SlowWarningTimeout:     gdata.SlowWarningTimeout,
			VerySlowWarningTimeout: gdata.VerySlowWarningTimeout,
			TaskTTL:                gdata.TaskTTL,
			TryOneSegment:          gdata.TryOneSegment,
			MethodHTTP:             strings.ToUpper(gdata.MethodHTTP),
			User:                   gdata.User,
			Pass:                   gdata.Pass,
		}

		if gdata.URI != "" {
			cfg.GroupStreams[gname] = new([]Stream)
			addRemoteConfig(cfg.GroupStreams[gname], cfg.GroupParams[gname], gname, gdata.URI, gdata.User, gdata.Pass)
		} else {
			cfg.GroupStreams[gname] = new([]Stream)
			addLocalConfig(cfg.GroupStreams[gname], cfg.GroupParams[gname], gname, gdata.Streams)
		}
	}
	// println("----------------------------------------")
	// for key, data := range cfg.GroupParams {
	// 	fmt.Printf("%s %#v\n", key, data)
	// }
	// println("----------------------------------------")
	// for key, data := range cfg.GroupStreams {
	// 	fmt.Printf("%s %#v\n", key, data)
	// }
}

// Helper. Get remote list of streams.
func addRemoteConfig(dest *[]Stream, params *configGroup, group string, uri, remoteUser, remotePass string) error {
	defer func() error {
		if r := recover(); r != nil {
			return errors.New(fmt.Sprintf("Can't get remote config for (%s) %s %s", params.Type, group, uri))
		}
		return nil
	}()

	client := NewTimeoutClient(10*time.Second, 10*time.Second)
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return err
	}
	if remoteUser != "" {
		req.SetBasicAuth(remoteUser, remotePass)
	}
	result, err := client.Do(req)
	if err == nil {
		body := bufio.NewReader(result.Body)
		for {
			line, err := body.ReadString('\n')
			if err != nil {
				break
			}
			uri, name, title := splitName(params.ParseMethod, line)
			*dest = append(*dest, Stream{URI: uri, Type: params.Type, Name: name, Title: title, Group: group})
		}
	}
	return err
}

// Helper. Parse config of
func addLocalConfig(dest *[]Stream, params *configGroup, group string, sources []string) {
	for _, source := range sources {
		uri, name, title := splitName(params.ParseMethod, source)
		*dest = append(*dest, Stream{URI: uri, Type: params.Type, Name: name, Title: title, Group: group})
	}
}

// Helper. Split stream link to URI and Name parts.
// Supported both cases: title<space>uri and uri<space>title
// If `re` presents then name parsed from uri by regular expression.
// URI must be prepended by http:// or https://
func splitName(re, source string) (uri, name, title string) {
	source = strings.TrimSpace(source)
	sep := regexp.MustCompile("htt(p|ps)://")
	loc := sep.FindStringIndex(source)
	if loc != nil {
		if loc[0] == 0 { // uri title
			splitted := strings.SplitN(source, " ", 2)
			if len(splitted) > 1 {
				title = strings.TrimSpace(splitted[1])
			}
			uri = strings.TrimSpace(splitted[0])
		} else { // title uri
			title = strings.TrimSpace(source[0:loc[0]])
			uri = source[loc[0]:]
		}
		if title == "" {
			title = uri
		}
	}
	if re != "" { // get name by regexp
		compiledRe := regexp.MustCompile(re)
		vals := compiledRe.FindStringSubmatch(uri)
		if len(vals) > 1 {
			name = vals[1]
		} else {
			name = title
		}
	} else {
		name = title
	}
	return
}
