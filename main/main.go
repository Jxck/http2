package main

import (
	"github.com/jxck/http2"
	"log"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

func main() {
	//http2.Get("http://106.186.112.116:80/")
	//http2.Get("http://106.186.112.116:80/package_README.html")
	//http2.Get("http://106.186.112.116/apiref.html")
	http2.Get("http://jxck.io:8080/")
}
