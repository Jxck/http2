package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/Jxck/http2"
	"github.com/Jxck/logger"
	"io/ioutil"
	"net/http"
	"os"
)

var (
	nullout  bool
	post     string
	loglevel int
)

func init() {
	f := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	f.BoolVar(&nullout, "n", false, "null output")
	f.StringVar(&post, "d", "", "send post data")
	f.IntVar(&loglevel, "l", 0, logger.Help())
	f.Parse(os.Args[1:])
	for 0 < f.NArg() {
		f.Parse(f.Args()[1:])
	}
	logger.Level(loglevel)
}

func main() {
	defer func() {
		err := recover()
		if err != nil {
			fmt.Println(`
# usage
$ go run main/client.go http://localhost:3000 -l 4 -d "data to send" -n
`)
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
			logger.Error("%v", err)
		}
	} else {
		// POST
		buf := bytes.NewBufferString(post)
		res, err = client.Post(url, "text/plain", buf)
		if err != nil {
			logger.Error("%v", err)
		}
	}

	defer res.Body.Close()
	if !nullout {
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			logger.Error("%v", err)
		}
		fmt.Println(string(body))
	}
}
