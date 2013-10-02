package http2

import (
	"bufio"
	"fmt"
	. "github.com/jxck/color"
	"log"
	"net"
	"net/http"
	urllib "net/url"
	"strings"
)

const Version string = "HTTP-draft-06/2.0"
const MagicString string = "PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n"

var defaultSetting *SettingsFrame

func init() {
	log.SetFlags(log.Lshortfile)
	defaultSetting = DefaultSettingsFrame()
}

func URLParse(url string) (scheme, host, port, path string) {
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

type Client struct {
	scheme string
	host   string
	port   string
	path   string
	conn   net.Conn
	bw     *bufio.Writer
	br     *bufio.Reader
	framer *Framer
}

func NewClient(url string) *Client {
	client := &Client{}

	client.scheme, client.host, client.port, client.path = URLParse(url)

	if client.scheme == "http" {
		client.conn, _ = net.Dial("tcp", client.host+":"+client.port) // err
	} else {
		log.Fatal("not support yet")
	}

	client.bw = bufio.NewWriter(client.conn)
	client.br = bufio.NewReader(client.conn)
	client.framer = &Framer{
		RW: client.conn,
	}

	return client
}

func (client *Client) Upgrade() {
	upgrade := "" +
		"GET " + client.path + " HTTP/1.1\r\n" +
		"Host: " + client.host + "\r\n" +
		"Connection: Upgrade, HTTP2-Settings\r\n" +
		"Upgrade: " + Version + "\r\n" +
		"HTTP2-Settings: " + defaultSetting.PayloadBase64URL() + "\r\n" +
		"Accept: */*\r\n" +
		"\r\n"
	client.bw.WriteString(upgrade) // err
	client.bw.Flush()              // err
	fmt.Println(Blue(upgrade))

	res, _ := http.ReadResponse(client.br, &http.Request{Method: "GET"}) // err

	fmt.Println(Blue(ResponseString(res)))
	fmt.Println(Yellow("HTTP Upgrade Success :)"))
}

func (client *Client) SendMagic() {
	client.bw.WriteString(MagicString) // err
	client.bw.Flush()                  // err
}

// TODO: *Frame
func (client *Client) Send(frame Frame) {
	client.framer.WriteFrame(frame) // err
}

// TODO: *Frame
func (client *Client) Recv() Frame {
	return client.framer.ReadFrame()
}

func Get(url string) string {
	client := NewClient(url)
	client.Upgrade()

	client.Send(NoFlowSettingsFrame()) // err
	client.Recv()

	c := 0
	html := ""
	for {
		frame := client.Recv()
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

	return html

	// TODO: Send GOAWAY
}
