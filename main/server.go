package main

import (
	"crypto/tls"
	"flag"
	"github.com/Jxck/http2"
	"github.com/Jxck/logger"
	"log"
	"net/http"
	"os"
)

var (
	verbose  bool
	loglevel int
)

func init() {
	log.SetFlags(log.Lshortfile)
	f := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
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

	log.Printf("server starts at localhost%v", addr)
	log.Println(server.ListenAndServeTLS(cert, key))
}
