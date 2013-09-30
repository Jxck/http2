package main

import (
	"github.com/jxck/http2"
	"log"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

func main() {
	Host := "106.186.112.116:80"
	http2.Get(Host)
}
