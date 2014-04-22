// Data structures related to whole program
package main

import (
	"bytes"
	"github.com/grafov/m3u8"
	"net/http"
	"time"
)

const (
	SERVER = "Stream Surfer"
)

// Kinds of streams
// Must be in consistence with StreamType2String() and String2StreamType()
const (
	UNKSTREAM StreamType = iota // хрень какая-то
	SAMPLE                      // internet resources for monitor self checks
	HTTP                        // "plain" HTTP
	HLS                         // Apple HTTP Live Streaming
	HDS                         // Adobe HTTP Dynamic Streaming
	WV                          // Widevine VOD
)

// Error codes (ordered by errors importance).
// If several errors detected then only one with the heaviest weight reported.
// Must be in consistence with String2StreamErr() and StreamErr2String()
const (
	SUCCESS        ErrType = iota
	DEBUG_LEVEL            // Internal debug messages follow below:
	TTLEXPIRED             // Task was not executed because task TTL expired. StreamSurfer too busy.
	HLSPARSER              // HLS parser error (debug)
	BADREQUEST             // Request failed (internal client error)
	WARNING_LEVEL          // Warnings follow below:
	SLOW                   // SlowWarning threshold on reading server response
	VERYSLOW               // VerySlowWarning threshold on reading server response
	ERROR_LEVEL            // Errors follow below:
	CTIMEOUT               // Timeout on connect
	RTIMEOUT               // Timeout on read
	BADLENGTH              // ContentLength value not equal real content length
	BODYREAD               // Response body read error
	CRITICAL_LEVEL         // Permanent errors level
	REFUSED                // Connection refused
	BADSTATUS              // HTTP Status >= 400
	BADURI                 // Incorret URI format
	LISTEMPTY              // HLS specific (by m3u8 lib)
	BADFORMAT              // HLS specific (by m3u8 lib)
	UNKERR                 // хрень какая-то
)

// Commands for probers.
const (
	STOP_MON Command = iota
	START_MON
	RELOAD_CONFIG // TODO made dynamic stream loading and unloading
	LOAD_GROUP
	LOAD_STREAM
	DROP_GROUP
	DROP_STREAM
)

type StreamType uint // Type of checked streams
type ErrType uint
type Command uint

type Group struct {
	Type StreamType
	Name string
}

type Stream struct {
	URI   string
	Type  StreamType
	Name  string
	Title string // опциональный заголовок = name по умолчанию
	Group string
}

// Stream checking task
type Task struct {
	Stream
	ReadBody bool
	ReplyTo  chan Result
	TTL      time.Time // valid until the time
}

type VariantTask struct {
	Task
}

type ChunkTask struct {
	Task
}

// Stream group
type GroupTask struct {
	Type    StreamType
	Name    string
	Tasks   *Task
	ReplyTo chan Result
}

// Result of task of check streaming
type Result struct {
	Task              *Task
	ErrType           ErrType
	HTTPCode          int    // HTTP status code
	HTTPStatus        string // HTTP status string
	ContentLength     int64
	RealContentLength int64
	Headers           http.Header
	Body              bytes.Buffer
	Started           time.Time     // начало исполнения проверки
	Elapsed           time.Duration // понадобилось времени на задачу
	TotalErrs         uint
	Meta              interface{} // Reference to metainformation about result data (playlist type etc.)
	SubResults        []Result    // Результаты вложенных проверок (i.e. media playlists for different bitrate of master playlists)
}

type MetaHLS struct {
	ListType  m3u8.ListType // type of analyzed playlist
	DeepLinks []string      // sublists for analysis
}

type MetaHDS struct {
	ListType  m3u8.ListType // XXX type of analyzed playlist
	DeepLinks []string      // sublists for analysis
}

// ключ для статистики
type Key struct {
	Group string
	Name  string
}

// запросы на сохранение статистики
type StatInQuery struct {
	Stream Stream
	Last   Result
}

// запросы на получение статистики
type StatOutQuery struct {
	Key     Key
	ReplyTo chan []Result
}

type ErrHistoryKey struct {
	Curhour string
	ErrType ErrType
	Group   string
	Name    string
	URI     string
}

type ErrTotalHistoryKey struct {
	Curhour string
	Group   string
	Name    string
}

// Values for the webpage
type PageValues struct {
	Title string
	Data  interface{}
}
