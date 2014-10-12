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
	loglevel int
	dir      string
	key      string
	cert     string
)

func init() {
	f := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	f.IntVar(&loglevel, "l", 0, logger.Help())
	f.StringVar(&dir, "d", ".", "document root")
	f.StringVar(&key, "key", "keys/key.pem", "ssl key")
	f.StringVar(&cert, "cert", "keys/cert.pem", "ssl cert")
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
	port := ":" + os.Args[1]

	var handler http.Handler = http.FileServer(http.Dir(dir))

	// setup TLS config
	config := &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{http2.VERSION},
	}

	// setup Server
	server := &http.Server{
		Addr:           port,
		Handler:        handler,
		MaxHeaderBytes: http.DefaultMaxHeaderBytes,
		TLSConfig:      config,
		TLSNextProto:   http2.TLSNextProto,
	}

	fmt.Println("server starts at localhost", port)
	fmt.Println(server.ListenAndServeTLS(cert, key))
}
