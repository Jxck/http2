package main

import (
	"bytes"
	"flag"
	"github.com/Jxck/http2"
	"github.com/Jxck/logger"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

var (
	nullout  bool
	post     string
	loglevel int
)

func init() {
	log.SetFlags(log.Lshortfile)
	f := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	f.BoolVar(&nullout, "n", false, "null output")
	f.StringVar(&post, "d", "", "send post data")
	f.IntVar(&loglevel, "l", 0, "log level (1 ERR, 2 WARNING, 3 INFO, 4 DEBUG)")
	f.Parse(os.Args[1:])
	for 0 < f.NArg() {
		f.Parse(f.Args()[1:])
	}
	logger.LogLevel(loglevel)
}

func main() {
	defer func() {
		err := recover()
		if err != nil {
			log.Println(`
# usage
$ go run main/client.go http://localhost:3000
`
		}
	}()
	url := os.Args[1]

	transport := &http2.Transport{
		CertPath: "keys/cert.pem",
		KeyPath:  "keys/key.pem",
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
		log.Println(string(body))
	}
}
