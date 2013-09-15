package main

import (
	"encoding/base64"
	. "github.com/jxck/hpack"
	. "github.com/jxck/http2"
	"log"
	"net"
	"net/http"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

func setting() string {
	// [0 8 4 0, 0 0 0 0, 0 0 0 4, 127 255 255 255]
	buf := []byte{
		0, 8, 4, 0,
		// 00000000 00001000,00000100,00000000
		0, 0, 0, 0,
		// 00000000 00000000 00000000 00000000
		0, 0, 0, 4,
		// 00000000 00000000 00000000 00000100
		0, 0, 0, 0xc4,
		// 00000000 00000000 00000000 11000100
	}
	str := base64.StdEncoding.EncodeToString(buf)

	// log.Printf("%v", str) // AAgEAAAAAAAAAAAEAAAAxA==

	return str
}

func main() {
	conn, _ := net.Dial("tcp", "jxck.io:8080") // err

	conn.Write([]byte("GET / HTTP/1.1\r\n"))                      // err
	conn.Write([]byte("Connection: Upgrade, HTTP2-Settings\r\n")) // err
	conn.Write([]byte("Upgrade: HTTP-draft-06/2.0\r\n"))          // err
	conn.Write([]byte("HTTP2-Settings: " + setting() + "\r\n"))   // err
	// n, e = conn.Write([]byte("HTTP2-Settings: AAAABAAAAGQAAAAHAAD//w==\r\n")) // err
	conn.Write([]byte("\r\n")) // err

	fh := &FrameHeader{}
	fh.Decode(conn)

	headers := http.Header{
		"Scheme":     []string{"https"},
		"Host":       []string{"jxck.io:8080"},
		"Path":       []string{"/"},
		"Method":     []string{"GET"},
		"User-Agent": []string{"http2cat"},
		"Cookie":     []string{"xxxxxxx2"},
		"Accept":     []string{"*/*"},
	}

	client := NewContext()
	wire := client.Encode(headers)

	fh = &FrameHeader{}
	fh.Length = uint16(len(wire))
	fh.Type = 0x1
	fh.Flags = 0x1 // END_STREAM
	fh.StreamId = 0

	hf := &HeadersFrame{*fh, 0, wire}
	log.Println(hf.Encode().Bytes())

	n, e := conn.Write(hf.Encode().Bytes())
	log.Println(n, e)

	b := make([]byte, 100)
	n, err := conn.Read(b)
	log.Println(n, err, b)

}
