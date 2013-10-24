// Integration with Zabbix monitoring tool
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"text/template"
)

type ZabbixDiscoveryData struct {
	Data []map[string]string `json:"data"`
}

// Data for stream template
type streamTemplateData struct {
	Stream
	Check string
}

func ZabbixDiscoveryWeb(cfg *Config, vars map[string]string) []byte {
	var page []byte
	var tmpl *template.Template
	var buf *bytes.Buffer = new(bytes.Buffer)
	var err error

	sort.Strings(cfg.Params.Zabbix.DiscoveryGroups)
	data := new(ZabbixDiscoveryData)

	if cfg.Params.Zabbix.StreamTemplate == "" {
		cfg.Params.Zabbix.StreamTemplate = "{{.Check}}-{{.Group}}-{{.Name}}"
	}
	tmpl, err = template.New("stream").Parse(cfg.Params.Zabbix.StreamTemplate)
	if err != nil {
		tmpl, _ = template.New("stream").Parse("{{.Check}}-{{.Group}}-{{.Name}}")

	}

	for _, stream := range cfg.StreamsHLS {
		buf.Reset()
		_ = tmpl.Execute(buf, streamTemplateData{Stream: stream, Check: StreamTypeText(stream.Type)})
		if _, exists := vars["group"]; exists { // report for selected group
			if stream.Group == vars["group"] {
				data.Data = append(data.Data, map[string]string{"{#STREAM}": buf.String()})
			}
		} else if len(cfg.Params.Zabbix.DiscoveryGroups) > 0 {
			if sort.SearchStrings(cfg.Params.Zabbix.DiscoveryGroups, stream.Group) == 0 {
				data.Data = append(data.Data, map[string]string{"{#STREAM}": buf.String()})
			}
		} else {
			data.Data = append(data.Data, map[string]string{"{#STREAM}": buf.String()})
		}
	}

	for _, stream := range cfg.StreamsHTTP {
		buf.Reset()
		_ = tmpl.Execute(buf, streamTemplateData{Stream: stream, Check: StreamTypeText(stream.Type)})
		if _, exists := vars["group"]; exists { // report for selected group
			if stream.Group == vars["group"] {
				data.Data = append(data.Data, map[string]string{"{#STREAM}": buf.String()})
			}
		} else if len(cfg.Params.Zabbix.DiscoveryGroups) > 0 && sort.SearchStrings(cfg.Params.Zabbix.DiscoveryGroups, stream.Group) == 0 {
			data.Data = append(data.Data, map[string]string{"{#STREAM}": buf.String()})
		}
	}
	page, _ = json.Marshal(data)

	return page
}

// Discover names of streams as CHECKTYPE-GROUP-NAME.
// Path to discovery file must be defined in config file in `zabbix-discovery` parameter.
// Accordingly with spec https://www.zabbix.com/documentation/2.2/manual/discovery/low_level_discovery
func ZabbixDiscoveryFile(cfg *Config) {
	var discoveryw *bufio.Writer
	var tmpl *template.Template
	var buf *bytes.Buffer = new(bytes.Buffer)
	var err error

	if cfg.Params.Zabbix.DiscoveryPath == "" {
		return
	} else {
		file, err := os.Create(cfg.Params.Zabbix.DiscoveryPath)
		if err != nil {
			return
		}
		discoveryw = bufio.NewWriter(file)
		defer discoveryw.Flush()
		fmt.Printf("Zabbix discovery file prepared: %s\n", cfg.Params.Zabbix.DiscoveryPath)
	}
	sort.Strings(cfg.Params.Zabbix.DiscoveryGroups)
	data := new(ZabbixDiscoveryData)
	if cfg.Params.Zabbix.StreamTemplate == "" {
		cfg.Params.Zabbix.StreamTemplate = "{{.Check}}-{{.Group}}-{{.Name}}"
	}
	tmpl, err = template.New("stream").Parse(cfg.Params.Zabbix.StreamTemplate)
	if err != nil {
		tmpl, _ = template.New("stream").Parse("{{.Check}}-{{.Group}}-{{.Name}}")

	}

	for _, stream := range cfg.StreamsHLS {
		buf.Reset()
		_ = tmpl.Execute(buf, streamTemplateData{Stream: stream, Check: StreamTypeText(stream.Type)})
		if len(cfg.Params.Zabbix.DiscoveryGroups) > 0 {
			if sort.SearchStrings(cfg.Params.Zabbix.DiscoveryGroups, stream.Group) == 0 {
				data.Data = append(data.Data, map[string]string{"{#STREAM}": buf.String()})
			}
		} else {
			data.Data = append(data.Data, map[string]string{"{#STREAM}": buf.String()})
		}
	}

	for _, stream := range cfg.StreamsHTTP {
		buf.Reset()
		_ = tmpl.Execute(buf, streamTemplateData{Stream: stream, Check: StreamTypeText(stream.Type)})
		if len(cfg.Params.Zabbix.DiscoveryGroups) > 0 && sort.SearchStrings(cfg.Params.Zabbix.DiscoveryGroups, stream.Group) == 0 {
			data.Data = append(data.Data, map[string]string{"{#STREAM}": buf.String()})
		}
	}
	output, _ := json.Marshal(data)
	discoveryw.Write(output)
	discoveryw.WriteRune('\n')
}

// Log problem for Zabbix agent (duplicates error log but in another format)
func ZabbixStatus(vars map[string]string) []byte {
	var page bytes.Buffer

	ReportedStreams.RLock()
	defer ReportedStreams.RUnlock()

	if _, exists := vars["group"]; exists { // report for selected group
		for _, value := range ReportedStreams.data[vars["group"]] {
			page.WriteString(strconv.Quote(fmt.Sprintf("%s-%s-%s", StreamTypeText(value.Stream.Type), value.Stream.Group, value.Stream.Name)))
			page.WriteRune(',')
			page.WriteString(strconv.Quote(StreamTypeText(value.Stream.Type)))
			page.WriteRune(',')
			page.WriteString(strconv.Quote(value.Stream.Group))
			page.WriteRune(',')
			page.WriteString(strconv.Quote(value.Stream.Name))
			page.WriteRune(',')
			errnum, _ := fatalityLevel(value.Last.ErrType)
			page.WriteString(strconv.Itoa(errnum))
			page.WriteRune('\n')
		}
	} else { // report for all groups
		for _, group := range ReportedStreams.data {
			for _, value := range group {
				page.WriteString(strconv.Quote(fmt.Sprintf("%s-%s-%s", StreamTypeText(value.Stream.Type), value.Stream.Group, value.Stream.Name)))
				page.WriteRune(',')
				page.WriteString(strconv.Quote(StreamTypeText(value.Stream.Type)))
				page.WriteRune(',')
				page.WriteString(strconv.Quote(value.Stream.Group))
				page.WriteRune(',')
				page.WriteString(strconv.Quote(value.Stream.Name))
				page.WriteRune(',')
				errnum, _ := fatalityLevel(value.Last.ErrType)
				page.WriteString(strconv.Itoa(errnum))
				page.WriteRune('\n')
			}
		}
	}

	return page.Bytes()
}

func fatalityLevel(err ErrType) (int, string) {
	switch {
	case err == SUCCESS:
		return 0, "info"
	case err > SUCCESS && err < BADSTATUS:
		return 1, "warning"
	case err >= BADSTATUS:
		return 2, "error"
	}
	return -1, "unknown"
}
