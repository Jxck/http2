package http2

import (
	. "github.com/Jxck/http2/frame"
)

const (
	OVER_TLS           string = "h2"
	OVER_TCP                  = "h2c"
	VERSION                   = OVER_TLS
	CONNECTION_PREFACE        = "PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n"
)

var DefaultSettings = map[SettingsID]int32{
	SETTINGS_HEADER_TABLE_SIZE: DEFAULT_HEADER_TABLE_SIZE,
	// SETTINGS_ENABLE_PUSH:            DEFAULT_ENABLE_PUSH, // server dosen't send this
	SETTINGS_MAX_CONCURRENT_STREAMS: DEFAULT_MAX_CONCURRENT_STREAMS,
	SETTINGS_INITIAL_WINDOW_SIZE:    DEFAULT_INITIAL_WINDOW_SIZE,
	SETTINGS_MAX_FRAME_SIZE:         DEFAULT_MAX_FRAME_SIZE,
	SETTINGS_MAX_HEADER_LIST_SIZE:   DEFAULT_MAX_HEADER_LIST_SIZE,
}

var NilSettings = make(map[SettingsID]int32, 0)
