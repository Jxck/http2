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

var verbose bool
var loglevel int

func init() {
	log.SetFlags(log.Lshortfile)
	f := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	f.BoolVar(&verbose, "v", false, "verbose out")
	f.IntVar(&loglevel, "l", 0, "log level (1 ERR, 2 WARNING, 3 INFO, 4 DEBUG)")
	f.Parse(os.Args[1:])
	for 0 < f.NArg() {
		f.Parse(f.Args()[1:])
	}
	logger.LogLevel(loglevel)
	logger.Verbose(verbose)
}

type Hello struct{}

func (h *Hello) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Hello HTTP2.0!")
}

var handler http.Handler = &Hello{}

// var httpHandler http.Handler = http.FileServer(http.Dir("."))

func main() {
	addr := ":" + os.Args[1]

	err := http2.ListenAndServe(addr, handler)
	if err != nil {
		log.Fatal(err)
	}
}
