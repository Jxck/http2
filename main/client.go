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

var nullout, upgrade, flowctl bool
var post string
var loglevel int

func init() {
	log.SetFlags(log.Lshortfile)
	f := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	f.BoolVar(&nullout, "n", false, "null output")
	f.BoolVar(&upgrade, "u", false, "upgrade")
	f.BoolVar(&flowctl, "f", false, "flow control")
	f.StringVar(&post, "d", "", "send post data")
	f.IntVar(&loglevel, "l", 0, "log level (1 ERR, 2 WARNING, 3 INFO, 4 DEBUG)")
	f.Parse(os.Args[1:])
	for 0 < f.NArg() {
		f.Parse(f.Args()[1:])
	}
	logger.LogLevel(loglevel)
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

// NPN Dial
// func DialNPN(address, certpath, keypath string) *tls.Conn {
// 	// 証明書の設定
// 	cert, err := tls.LoadX509KeyPair(certpath, keypath)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
//
// 	// TLS の設定(証明書検証無)
// 	config := tls.Config{
// 		Certificates:       []tls.Certificate{cert},
// 		InsecureSkipVerify: true,
// 		NextProtos:         []string{"http2.0/draft-09"},
// 	}
// 	conn, err := tls.Dial("tcp", address, &config)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
//
// 	// 接続確認
// 	state := conn.ConnectionState()
// 	log.Println("handshake: ", state.HandshakeComplete)
// 	log.Println("protocol: ", state.NegotiatedProtocol)
//
// 	return conn
// }
//
// func main() {
// 	// セットアップ
// 	address := "jxck.io:8443"
// 	certpath, keypath := "keys/cert.pem", "keys/key.pem"
//
// 	// Dial NPN
// 	conn := DialNPN(address, certpath, keypath)
// 	defer conn.Close() // err
//
// 	// データの送受信
// 	message := MagicString
//
// 	n, err := io.WriteString(conn, message)
// 	if err != nil {
// 		log.Fatalf("write: %s", err)
// 	}
// 	log.Printf("wrote %q (%d bytes)", message, n)
//
// 	c := NewConn(conn)
//
// 	settings := map[SettingsId]uint32{
// 		SETTINGS_MAX_CONCURRENT_STREAMS: 100,
// 		SETTINGS_INITIAL_WINDOW_SIZE:    DEFAULT_WINDOW_SIZE,
// 	}
//
// 	c.WriteFrame(NewSettingsFrame(settings, 0)) // err
//
//
// 	reply := make([]byte, 256)
// 	n, err = conn.Read(reply)
// 	log.Printf("read %q (%d bytes)", string(reply[:n]), n)
// 	log.Print("exiting")
// }
