package main

import (
	. "github.com/jxck/http2"
	"log"
	"net"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

func main() {
	conn, err := net.Dial("tcp", "jxck.io:8080")
	log.Println(err)

	n, e := conn.Write([]byte("GET / HTTP/1.1\r\n"))
	log.Println(n, e)
	n, e = conn.Write([]byte("Connection: Upgrade, HTTP2-Settings\r\n"))
	log.Println(n, e)
	n, e = conn.Write([]byte("Upgrade: HTTP-draft-06/2.0\r\n"))
	log.Println(n, e)
	// n, e = conn.Write([]byte("HTTP2-Settings: AAgEAAAAAAAAAAAEAAAAxA==\r\n"))
	n, e = conn.Write([]byte("HTTP2-Settings: AAAABAAAAGQAAAAHAAD//w==\r\n"))
	log.Println(n, e)
	n, e = conn.Write([]byte("\r\n"))
	log.Println(n, e)

	fh := &FrameHeader{}
	fh.Decode(conn)
}
