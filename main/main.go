package main

import (
	"flag"
	"fmt"
	"github.com/jxck/http2"
	"log"
	"os"
)

var nullout bool

func init() {
	log.SetFlags(log.Lshortfile)
	f := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	f.BoolVar(&nullout, "n", false, "null output")
	f.Parse(os.Args[1:])
	for 0 < f.NArg() {
		f.Parse(f.Args()[1:])
	}
}

func main() {
	url := os.Args[1]
	html := http2.Get(url)
	if !nullout {
		fmt.Println(html)
	}
}
