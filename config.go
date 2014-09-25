package http2

import (
	. "github.com/Jxck/http2/frame"
)

const (
	OVER_TLS           string = "h2"
	OVER_TCP                  = "h2c"
	VERSION                   = OVER_TLS + "-14"
	CONNECTION_PREFACE        = "PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n"
)

var DefaultSettings = []Setting{
	{SETTINGS_MAX_CONCURRENT_STREAMS, 100},
}

var NilSettings = make([]Setting, 0)
