package main

import (
	"github.com/jxck/http2"
	"log"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

func main() {
	http2.Get("http://106.186.112.116:80/")
}
