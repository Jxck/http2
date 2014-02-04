package http2

import (
	"crypto/tls"
	. "github.com/jxck/color"
	. "github.com/jxck/logger"
	"log"
	"net/http"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

var TLSNextProto = map[string]func(*http.Server, *tls.Conn, http.Handler){
	Version: TLSNextProtoHandler,
}

var TLSNextProtoHandler = func(server *http.Server, conn *tls.Conn, handler http.Handler) {
	Info(Yellow("New Connection from %s\n"), conn.RemoteAddr())
	HandleTLSConnection(conn, handler)
}
