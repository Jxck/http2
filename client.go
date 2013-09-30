package http2

import (
	"bufio"
	"flag"
	"fmt"
	. "github.com/jxck/color"
	. "github.com/jxck/debug"
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

var Version string = "HTTP-draft-06/2.0"
var MagicString string = "PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n"

func Get(Host string) {
	defaultSetting := DefaultSettingsFrame()

	conn, _ := net.Dial("tcp", Host) // err

	bw := bufio.NewWriter(conn)
	br := bufio.NewReader(conn)

	bw.WriteString("GET / HTTP/1.1\r\n")                                            // err
	bw.WriteString("Host: " + Host + "\r\n")                                        // err
	bw.WriteString("Connection: Upgrade, HTTP2-Settings\r\n")                       // err
	bw.WriteString("Upgrade: " + Version + "\r\n")                                  // err
	bw.WriteString("HTTP2-Settings: " + defaultSetting.PayloadBase64URL() + "\r\n") // err
	bw.WriteString("Accept: */*\r\n")                                               // err
	bw.WriteString("\r\n\r\n")                                                      // err
	bw.Flush()                                                                      // err

	res, _ := http.ReadResponse(br, &http.Request{Method: "GET"}) // err

	fmt.Println(Green(ResponseString(res)))
	Debug(Red("Upgrade Success :)"))

	bw.WriteString(MagicString) // err
	bw.Flush()                  // err

	framer := &Framer{
		RW: conn,
	}

	framer.WriteFrame(defaultSetting) // err

	fmt.Println(framer.ReadFrame()) // setting
	fmt.Println(framer.ReadFrame()) // window update
	fmt.Println(framer.ReadFrame()) // headers

	// data
	frame := framer.ReadFrame()
	data := frame.(*DataFrame)
	fmt.Println(data)

	html := string(data.Data)
	for data.FrameHeader.Flags != 1 {
		frame = framer.ReadFrame() // data
		data = frame.(*DataFrame)
		fmt.Println(data)
		html += string(data.Data)
	}

	if !nullout {
		fmt.Println(html)
	}

	// TODO: Send GOAWAY
}
