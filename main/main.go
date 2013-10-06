package main

import (
	"flag"
	"github.com/jxck/http2"
	"io/ioutil"
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

	transport := &http2.Transport{
		LastStreamId: 1,
		Upgrade:      upgrade,
	}
	client := &http.Client{
		Transport: transport,
	}

	res, err := client.Get(url)
	if err != nil {
		log.Println(err)
	}
	defer res.Body.Close()
	if !nullout {
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			log.Println(err)
		}
		log.Println(string(body))
	}
}
