package main

import (
	"flag"
	"github.com/jxck/http2"
	"log"
	"net/http"
	"os"
)

var nullout, upgrade bool

func init() {
	log.SetFlags(log.Lshortfile)
	f := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	f.BoolVar(&nullout, "n", false, "null output")
	f.BoolVar(&upgrade, "u", false, "upgrade")
	f.Parse(os.Args[1:])
	for 0 < f.NArg() {
		f.Parse(f.Args()[1:])
	}
}

func main() {
	url := os.Args[1]

	transport := &http2.Client{
		Upgrade: upgrade,
	}
	client := &http.Client{
		Transport: transport,
	}

	res, err := client.Get(url)
	if err != nil {
		log.Println(err)
	}
	if !nullout {
		log.Println(res.Body)
	}
}
