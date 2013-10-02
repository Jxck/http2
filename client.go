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
	urllib "net/url"
	"strings"
)

const Version string = "HTTP-draft-06/2.0"
const MagicString string = "PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n"

var nullout bool
var defaultSetting *SettingsFrame

func init() {
	log.SetFlags(log.Lshortfile)
	defaultSetting = DefaultSettingsFrame()
	flag.BoolVar(&nullout, "n", false, "null output")
	flag.Parse()
}

func URLParse(url string) (scheme, host, path, port string) {
	u, _ := urllib.Parse(url) // err
	scheme = u.Scheme
	path = u.Path
	tmp := strings.Split(u.Host, ":")
	if len(tmp) > 1 {
		host, port = tmp[0], tmp[1]
	} else {
		// TODO: fixme about default port
		// from scheme
		host, port = tmp[0], "80"
	}
	return
}

func Get(url string) {
	scheme, host, path, port := URLParse(url)

	var conn net.Conn
	if scheme == "http" {
		conn, _ = net.Dial("tcp", host+":"+port) // err
	} else {
		log.Fatal("not support yet")
	}

	bw := bufio.NewWriter(conn)
	br := bufio.NewReader(conn)

	upgrade := "" +
		"GET " + path + " HTTP/1.1\r\n" +
		"Host: " + host + "\r\n" +
		"Connection: Upgrade, HTTP2-Settings\r\n" +
		"Upgrade: " + Version + "\r\n" +
		"HTTP2-Settings: " + defaultSetting.PayloadBase64URL() + "\r\n" +
		"Accept: */*\r\n" +
		"\r\n\r\n"
	bw.WriteString(upgrade) // err
	bw.Flush()              // err
	fmt.Println(Green(upgrade))

	res, _ := http.ReadResponse(br, &http.Request{Method: "GET"}) // err

	fmt.Println(Green(ResponseString(res)))
	Debug(Red("Upgrade Success :)"))

	framer := &Framer{
		RW: conn,
	}

	frame := framer.ReadFrame()
	fmt.Println(frame)

	//	bw.WriteString(MagicString) // err
	//	bw.Flush()                  // err

	framer.WriteFrame(defaultSetting) // err

	c := 0
	html := ""
	for {
		frame := framer.ReadFrame()
		fmt.Println(frame)
		frameHeader := frame.Header()
		if frameHeader.Type == DataFrameType {
			dataFrame := frame.(*DataFrame)
			html += string(dataFrame.Data)
		}
		if frameHeader.Flags == 0x1 {
			break
		}
		if c > 10 {
			break
		}
		c++
	}

	if !nullout {
		fmt.Println(html)
	}

	// TODO: Send GOAWAY
}
