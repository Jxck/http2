package main

import (
	"flag"
	"fmt"
	_ "github.com/jxck/http2"
	"log"
	"net/http"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

type Hello struct{}

func (h *Hello) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Hello HTTP2.0!")
}

var handler http.Handler = &Hello{}

// var httpHandler http.Handler = http.FileServer(http.Dir("."))

func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) == 0 {
		log.Fatal("no port number specified")
	}

	addr := ":" + args[0]

	log.Printf("server starts on port %s\n", addr)
	err := http.ListenAndServe(addr, handler)
	if err != nil {
		log.Fatal(err)
	}
}
