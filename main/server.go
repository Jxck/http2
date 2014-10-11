package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"github.com/Jxck/http2"
	"github.com/Jxck/logger"
	"net/http"
	"os"
)

var (
	verbose  bool
	loglevel int
)

func init() {
	f := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
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
$ go run main/server.go 3000
			`)
		}
	}()
	// params
	addr := ":" + os.Args[1]
	cert := "keys/cert.pem"
	key := "keys/key.pem"

	var handler http.Handler = http.FileServer(http.Dir("/tmp"))

	// setup TLS config
	config := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{http2.VERSION},
	}

	// setup Server
	server := &http.Server{
		Addr:           addr,
		Handler:        handler,
		MaxHeaderBytes: http.DefaultMaxHeaderBytes,
		TLSConfig:      config,
		TLSNextProto:   http2.TLSNextProto,
	}

	fmt.Println("server starts at localhost", addr)
	fmt.Println(server.ListenAndServeTLS(cert, key))
}
