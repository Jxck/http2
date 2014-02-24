package http2

import (
	. "github.com/jxck/http2/frame"
)

const (
	Version     = "h2-10"
	MagicString = "PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n"
)

var DefaultSettings = map[SettingsId]uint32{
	SETTINGS_MAX_CONCURRENT_STREAMS: 100,
	SETTINGS_INITIAL_WINDOW_SIZE:    DEFAULT_WINDOW_SIZE,
}
