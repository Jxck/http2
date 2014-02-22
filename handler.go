package http2

import (
	//"github.com/jxck/hpack"
	. "github.com/jxck/http2/frame"
	. "github.com/jxck/logger"
	"log"
	"net"
	"net/http"
	//"strconv"
	"time"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

var DefaultSettings = map[SettingsId]uint32{
	SETTINGS_MAX_CONCURRENT_STREAMS: 100,
	SETTINGS_INITIAL_WINDOW_SIZE:    DEFAULT_WINDOW_SIZE,
}

func HandleTLSConnection(conn net.Conn, h http.Handler) {
	Info("Handle TLS Connection")
	defer conn.Close() // err

	Conn := NewConn(conn) // convert to http2.Conn
	Conn.Handler = h
	_ = Conn

	time.Sleep(time.Minute)
	return
}
