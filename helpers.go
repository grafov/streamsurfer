// misc helpers
package main

import (
	"os"
	"strings"
)

func FullPath(name string) string {
	return os.ExpandEnv(strings.Replace(name, "~", os.Getenv("HOME"), 1))
}

// Return text representation for StreamType constants
func StreamTypeText(t StreamType) string {
	switch t {
	case SAMPLE:
		return "sample"
	case HLS:
		return "hls"
	case HTTP:
		return "http"
	default:
		return "unknown"
	}
}

// Text representation of stream errors
func StreamErrText(err ErrType) string {
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
	case RTIMEOUT:
		return "timeout on read"
	case CTIMEOUT:
		return "connection timeout"
	case REFUSED:
		return "connection refused"
	default:
		return "unknown"
	}
}
