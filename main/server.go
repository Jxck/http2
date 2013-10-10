package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

type Hello struct{}

func (h *Hello) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Hello!")
}

var httpHandler http.Handler = &Hello{}

// var httpHandler http.Handler = http.FileServer(http.Dir("."))

func main() {
	flag.Parse()
	port := ":" + flag.Args()[0]

	err := http.ListenAndServe(port, httpHandler)
	if err != nil {
		log.Fatal(err)
	}
}
