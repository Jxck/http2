package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/jxck/http2"
	"github.com/jxck/logger"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

var nullout, upgrade, flowctl, verbose bool
var post string
var loglevel int

func init() {
	log.SetFlags(log.Lshortfile)
	f := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	f.BoolVar(&nullout, "n", false, "null output")
	f.BoolVar(&upgrade, "u", false, "upgrade")
	f.BoolVar(&flowctl, "f", false, "flow control")
	f.BoolVar(&verbose, "v", false, "verbose out")
	f.StringVar(&post, "d", "", "send post data")
	f.IntVar(&loglevel, "l", 0, "log level (1 ERR, 2 WARNING, 3 INFO, 4 DEBUG)")
	f.Parse(os.Args[1:])
	for 0 < f.NArg() {
		f.Parse(f.Args()[1:])
	}
	logger.LogLevel(loglevel)
	logger.Verbose(verbose)
}

func main() {
	url := os.Args[1]

	transport := &http2.Transport{
		Upgrade: upgrade,
		FlowCtl: flowctl,
	}
	client := &http.Client{
		Transport: transport,
	}

	var res *http.Response
	var err error
	if post == "" {
		// GET
		res, err = client.Get(url)
		if err != nil {
			log.Println(err)
		}
	} else {
		// POST
		buf := bytes.NewBufferString(post)
		res, err = client.Post(url, "text/plain", buf)
		if err != nil {
			log.Println(err)
		}
	}

	defer res.Body.Close()
	if !nullout {
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			log.Println(err)
		}
		fmt.Println(string(body))
	}
}
