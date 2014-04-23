// misc helpers
package main

import (
	"fmt"
	"os"
	"strings"
)

func FullPath(name string) string {
	return os.ExpandEnv(strings.Replace(name, "~", os.Getenv("HOME"), 1))
}

// Return text representation for StreamType constants
func StreamType2String(t StreamType) string {
	switch t {
	case SAMPLE:
		return "sample"
	case HLS:
		return "hls"
	case HDS:
		return "hds"
	case WV:
		return "wv"
	case HTTP:
		return "http"
	default:
		return "unknown"
	}
}

//
func String2StreamType(s string) StreamType {
	switch strings.ToLower(s) {
	case "sample":
		return SAMPLE
	case "hls":
		return HLS
	case "hds":
		return HDS
	case "wv":
		return WV
	case "http":
		return HTTP
	default:
		return UNKSTREAM
	}
}

// Text representation of stream errors
func StreamErr2String(err ErrType) string {
	switch err {
	case SUCCESS:
		return "success"
	case HLSPARSER:
		return "HLS parser" // debug
	case BADREQUEST:
		return "invalid request" // debug
	case SLOW:
		return "slow response"
	case VERYSLOW:
		return "very slow response"
	case BADSTATUS:
		return "bad status"
	case BADURI:
		return "bad URI"
	case LISTEMPTY: // HLS specific
		return "list empty"
	case BADFORMAT: // HLS specific
		return "bad format"
	case TTLEXPIRED:
		return "TTL expired"
	case RTIMEOUT:
		return "timeout on read"
	case CTIMEOUT:
		return "connection timeout"
	case BADLENGTH:
		return "bad content length value"
	case BODYREAD:
		return "response body error"
	case REFUSED:
		return "connection refused"
	default:
		return "unknown"
	}
}

//
func String2StreamErr(s string) ErrType {
	switch strings.ToLower(s) {
	case "success":
		return SUCCESS
	case "debug":
		return DEBUG_LEVEL
	case "hlsparser":
		return HLSPARSER
	case "badrequest":
		return BADREQUEST
	case "warning":
		return WARNING_LEVEL
	case "slow":
		return SLOW
	case "veryslow":
		return VERYSLOW
	case "badstatus":
		return BADSTATUS
	case "baduri":
		return BADURI
	case "listempty": // HLS specific
		return LISTEMPTY
	case "badformat": // HLS specific
		return BADFORMAT
	case "ttlexpired":
		return TTLEXPIRED
	case "rtimeout":
		return RTIMEOUT
	case "error":
		return ERROR_LEVEL
	case "ctimeout":
		return CTIMEOUT
	case "badlength":
		return BADLENGTH
	case "bodyread":
		return BODYREAD
	case "critical":
		return CRITICAL_LEVEL
	case "refused":
		return REFUSED
	default:
		return UNKERR
	}
}

// Helper to make a href.
// First arg must be URL, second is text.
// Optional args is title (3d) and class (4d).
func href(url, text string, opts ...string) string {
	switch len(opts) {
	case 1:
		return fmt.Sprintf("<a title=\"%s\" href=\"%s\">%s</a>", opts[0], url, text)
	case 2:
		return fmt.Sprintf("<a title=\"%s\" class=\"%s\" href=\"%s\">%s</a>", opts[0], opts[1], url, text)
	default:
		return fmt.Sprintf("<a href=\"%s\">%s</a>", url, text)
	}
}

//
func span(text, class string) string {
	return fmt.Sprintf("<span class=\"%s\">%s</span>", class, text)
}
