package main

import (
	"bufio"
	"flag"
	"fmt"
	. "github.com/jxck/color"
	. "github.com/jxck/debug"
	"github.com/jxck/http2"
	"log"
	"net"
	"net/http"
)

var nullout bool

func init() {
	log.SetFlags(log.Lshortfile)
	flag.BoolVar(&nullout, "n", false, "null output")
	flag.Parse()
}

func main() {
	settingsFrame := http2.DefaultSettingsFrame()

	conn, _ := net.Dial("tcp", "106.186.112.116:80") // err

	conn.Write([]byte("GET / HTTP/1.1\r\n"))                                           // err
	conn.Write([]byte("Host: 106.186.112.116:80\r\n"))                                 // err
	conn.Write([]byte("Connection: Upgrade, HTTP2-Settings\r\n"))                      // err
	conn.Write([]byte("Upgrade: HTTP-draft-06/2.0\r\n"))                               // err
	conn.Write([]byte("HTTP2-Settings: " + settingsFrame.PayloadBase64URL() + "\r\n")) // err
	conn.Write([]byte("Accept: */*\r\n"))                                              // err
	conn.Write([]byte("\r\n"))                                                         // err

	br := bufio.NewReader(conn)
	res, _ := http.ReadResponse(br, &http.Request{Method: "GET"}) // err

	fmt.Println(Green(http2.ResponseString(res)))
	Debug(Red("Upgrade Success :)"))

	conn.Write([]byte("PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n")) // err

	framer := &http2.Framer{
		RW: conn,
	}

	framer.WriteFrame(settingsFrame) // err

	fmt.Println(framer.ReadFrame()) // setting
	fmt.Println(framer.ReadFrame()) // window update
	fmt.Println(framer.ReadFrame()) // headers

	// data
	frame := framer.ReadFrame()
	data := frame.(*http2.DataFrame)
	fmt.Println(data)

	html := string(data.Data)
	for data.FrameHeader.Flags != 1 {
		frame = framer.ReadFrame() // data
		data = frame.(*http2.DataFrame)
		fmt.Println(data)
		html += string(data.Data)
	}

	if !nullout {
		fmt.Println(html)
	}

	// TODO: Send GOAWAY
}
