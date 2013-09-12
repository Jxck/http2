package main

import (
	"log"
	"net"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

func main() {
	conn, _ := net.Dial("tcp", "jxck.io:8080")

	n, e := conn.Write([]byte("GET / HTTP/1.1\r\n"))
	log.Println(n, e)
	n, e = conn.Write([]byte("Host: jxck.io\r\n"))
	log.Println(n, e)
	n, e = conn.Write([]byte("Connection: Upgrade, HTTP2-Settings\r\n"))
	log.Println(n, e)
	n, e = conn.Write([]byte("upgrade: http/2.0\r\n"))
	log.Println(n, e)
	n, e = conn.Write([]byte("HTTP2-Settings: AAgEAAAAAAAAAAAEAAAAxA==\r\n"))
	log.Println(n, e)
	n, e = conn.Write([]byte("\r\n"))
	log.Println(n, e)

	buf := make([]byte, 16, 16)
	n, err := conn.Read(buf)

	log.Printf("%v %v %v", n, err, buf)
}
