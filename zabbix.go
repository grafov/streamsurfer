// Integration with Zabbix monitoring tool
package main

import (
	"bytes"
	"encoding/json"
	"sort"
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

func ZabbixDiscoveryWeb(vars map[string]string) []byte {
	var page []byte
	var tmplName, tmplTitle *template.Template
	var bufn *bytes.Buffer = new(bytes.Buffer)
	var buft *bytes.Buffer = new(bytes.Buffer)
	var err error

	sort.Strings(cfg.Zabbix.DiscoveryGroups)
	data := new(ZabbixDiscoveryData)

	if cfg.Zabbix.NameTemplate != "" {
		tmplName, err = template.New("name").Parse(cfg.Zabbix.NameTemplate)
	}
	if err != nil || cfg.Zabbix.NameTemplate == "" {
		tmplName, _ = template.New("name").Parse("{{.Group}}-{{.Name}}")

	}
	if cfg.Zabbix.TitleTemplate != "" {
		tmplTitle, err = template.New("title").Parse(cfg.Zabbix.TitleTemplate)
	}
	if err != nil || cfg.Zabbix.TitleTemplate == "" {
		tmplTitle, _ = template.New("title").Parse("{{.Title}}")

	}

	for _, streams := range cfg.GroupStreams {
		for _, stream := range *streams {
			bufn.Reset()
			buft.Reset()
			_ = tmplName.Execute(bufn, streamTemplateData{Stream: stream, Check: StreamType2String(stream.Type)})
			_ = tmplTitle.Execute(buft, streamTemplateData{Stream: stream, Check: StreamType2String(stream.Type)})
			if _, exists := vars["group"]; exists { // report for selected group
				if stream.Group == vars["group"] {
					data.Data = append(data.Data, map[string]string{"{#STREAM}": bufn.String(), "{#TITLE}": buft.String()})
				}
			} else if len(cfg.Zabbix.DiscoveryGroups) > 0 {
				if sort.SearchStrings(cfg.Zabbix.DiscoveryGroups, stream.Group) == 0 {
					data.Data = append(data.Data, map[string]string{"{#STREAM}": bufn.String(), "{#TITLE}": buft.String()})
				}
			} else {
				data.Data = append(data.Data, map[string]string{"{#STREAM}": bufn.String(), "{#TITLE}": buft.String()})
			}
		}
	}

	page, _ = json.Marshal(data)

	return page
}

// // OBSOLETED by web handler
// // Discover names of streams as CHECKTYPE-GROUP-NAME.
// // Path to discovery file must be defined in config file in `zabbix-discovery` parameter.
// // Accordingly with spec https://www.zabbix.com/documentation/2.2/manual/discovery/low_level_discovery
// func ZabbixDiscoveryFile() {
// 	var discoveryw *bufio.Writer
// 	var tmpl *template.Template
// 	var buf *bytes.Buffer = new(bytes.Buffer)
// 	var err error

// 	if cfg.Zabbix.DiscoveryPath == "" {
// 		return
// 	} else {
// 		file, err := os.Create(cfg.Zabbix.DiscoveryPath)
// 		if err != nil {
// 			return
// 		}
// 		discoveryw = bufio.NewWriter(file)
// 		defer discoveryw.Flush()
// 		fmt.Printf("Zabbix discovery file prepared: %s\n", cfg.Zabbix.DiscoveryPath)
// 	}
// 	sort.Strings(cfg.Zabbix.DiscoveryGroups)
// 	data := new(ZabbixDiscoveryData)
// 	if cfg.Zabbix.StreamTemplate == "" {
// 		cfg.Zabbix.StreamTemplate = "{{.Check}}-{{.Group}}-{{.Name}}"
// 	}
// 	tmpl, err = template.New("stream").Parse(cfg.Zabbix.StreamTemplate)
// 	if err != nil {
// 		tmpl, _ = template.New("stream").Parse("{{.Check}}-{{.Group}}-{{.Name}}")

// 	}

// 	for _, streams := range cfg.GroupStreams {
// 		for _, stream := range *streams {
// 			buf.Reset()
// 			_ = tmpl.Execute(buf, streamTemplateData{Stream: stream, Check: StreamType2String(stream.Type)})
// 			if len(cfg.Zabbix.DiscoveryGroups) > 0 {
// 				if sort.SearchStrings(cfg.Zabbix.DiscoveryGroups, stream.Group) == 0 {
// 					data.Data = append(data.Data, map[string]string{"{#STREAM}": buf.String()})
// 				}
// 			} else {
// 				data.Data = append(data.Data, map[string]string{"{#STREAM}": buf.String()})
// 			}
// 		}
// 	}

// 	output, _ := json.Marshal(data)
// 	discoveryw.Write(output)
// 	discoveryw.WriteRune('\n')
// }

// Log problem for Zabbix agent (duplicates error log but in another format)
// func ZabbixStatus(vars map[string]string) []byte {
// 	var page bytes.Buffer

// 	ReportedStreams.RLock()
// 	defer ReportedStreams.RUnlock()

// 	if _, exists := vars["group"]; exists { // report for selected group
// 		for _, value := range ReportedStreams.data[vars["group"]] {
// 			page.WriteString(strconv.Quote(fmt.Sprintf("%s-%s-%s", StreamTypeText(value.Stream.Type), value.Stream.Group, value.Stream.Name)))
// 			page.WriteRune(',')
// 			page.WriteString(strconv.Quote(StreamTypeText(value.Stream.Type)))
// 			page.WriteRune(',')
// 			page.WriteString(strconv.Quote(value.Stream.Group))
// 			page.WriteRune(',')
// 			page.WriteString(strconv.Quote(value.Stream.Name))
// 			page.WriteRune(',')
// 			errnum, _ := fatalityLevel(value.Last.ErrType)
// 			page.WriteString(strconv.Itoa(errnum))
// 			page.WriteRune('\n')
// 		}
// 	} else { // report for all groups
// 		for _, group := range ReportedStreams.data {
// 			for _, value := range group {
// 				page.WriteString(strconv.Quote(fmt.Sprintf("%s-%s-%s", StreamTypeText(value.Stream.Type), value.Stream.Group, value.Stream.Name)))
// 				page.WriteRune(',')
// 				page.WriteString(strconv.Quote(StreamTypeText(value.Stream.Type)))
// 				page.WriteRune(',')
// 				page.WriteString(strconv.Quote(value.Stream.Group))
// 				page.WriteRune(',')
// 				page.WriteString(strconv.Quote(value.Stream.Name))
// 				page.WriteRune(',')
// 				errnum, _ := fatalityLevel(value.Last.ErrType)
// 				page.WriteString(strconv.Itoa(errnum))
// 				page.WriteRune('\n')
// 			}
// 		}
// 	}

// 	return page.Bytes()
// }

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
