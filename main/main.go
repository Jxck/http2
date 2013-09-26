package main

import (
	"flag"
	"fmt"
	"github.com/jxck/http2"
	"log"
	"net"
)

var nullout bool

func init() {
	log.SetFlags(log.Lshortfile)
	flag.BoolVar(&nullout, "n", false, "null output")
	flag.Parse()
}

func main() {
	conn, _ := net.Dial("tcp", "106.186.112.116:80") // err

	conn.Write([]byte("GET / HTTP/1.1\r\n"))                         // err
	conn.Write([]byte("Host: 106.186.112.116:80\r\n"))               // err
	conn.Write([]byte("Connection: Upgrade, HTTP2-Settings\r\n"))    // err
	conn.Write([]byte("Upgrade: HTTP-draft-06/2.0\r\n"))             // err
	conn.Write([]byte("HTTP2-Settings: AAAABAAAAGQAAAAHAAD__w\r\n")) // err
	conn.Write([]byte("Accept: */*\r\n"))                            // err
	conn.Write([]byte("\r\n"))                                       // err

	b := make([]byte, 76)
	conn.Read(b)
	// Upgrade Success :)

	conn.Write([]byte("PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n")) // err

	settingsFrame := http2.DefaultSettingsFrame()
	conn.Write(settingsFrame.Encode().Bytes()) // err

	fh := &http2.FrameHeader{}
	log.Println(fh.Decode(conn)) // setting
	log.Println(fh.Decode(conn)) // window update

	// headers
	frame := fh.Decode(conn)
	headersFrame := frame.(*http2.HeadersFrame)
	log.Println(headersFrame.Header)

	// data
	frame = fh.Decode(conn)
	data := frame.(*http2.DataFrame)

	html := string(data.Data)
	for data.FrameHeader.Flags != 1 {
		frame = fh.Decode(conn) // data
		data = frame.(*http2.DataFrame)
		html += string(data.Data)
	}

	if !nullout {
		fmt.Println(html)
	}

	// TODO: Send GOAWAY
}
