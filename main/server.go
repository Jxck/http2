package main

import (
	"flag"
	"fmt"
	"github.com/jxck/http2"
	"github.com/jxck/logger"
	"log"
	"net/http"
	"os"
)

var verbose, no_tls bool
var loglevel int

func init() {
	log.SetFlags(log.Lshortfile)
	f := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	f.IntVar(&loglevel, "l", 0, "log level (1 ERR, 2 WARNING, 3 INFO, 4 DEBUG)")
	f.BoolVar(&no_tls, "no-tls", false, "disable TLS and user upgrade")
	f.Parse(os.Args[1:])
	for 0 < f.NArg() {
		f.Parse(f.Args()[1:])
	}
	logger.LogLevel(loglevel)
}

type Hello struct{}

func (h *Hello) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Hello HTTP2.0!")
}

func main() {
	var handler http.Handler = &Hello{}
	// handler = http.FileServer(http.Dir("."))

	addr := ":" + os.Args[1]

	if no_tls {
		http2.ListenAndServe(addr, handler)
	} else {
		cert := "keys/cert.pem"
		key := "keys/key.pem"

		http2.ListenAndServeTLS(addr, cert, key, handler)
	}
}
