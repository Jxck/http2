package http2

import (
	"bufio"
	"fmt"
	. "github.com/jxck/color"
	"github.com/jxck/hpack"
	"log"
	"net"
	"net/http"
)

const Version string = "HTTP-draft-06/2.0"
const MagicString string = "PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n"

var defaultSetting *SettingsFrame

func init() {
	log.SetFlags(log.Lshortfile)
	defaultSetting = DefaultSettingsFrame()
}

type Client struct {
	url    *URL
	conn   net.Conn
	bw     *bufio.Writer
	br     *bufio.Reader
	framer *Framer
}

func NewClient(url string) *Client {
	client := &Client{}

	client.url, _ = NewURL(url) // err

	if client.url.Scheme == "http" {
		client.conn, _ = net.Dial("tcp", client.url.Host+":"+client.url.Port) // err
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
		"GET " + client.url.Path + " HTTP/1.1\r\n" +
		"Host: " + client.url.Host + "\r\n" +
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

func (client *Client) Send(frame Frame) {
	fmt.Println(Red("send"), frame)
	client.framer.WriteFrame(frame) // err
}

func (client *Client) Recv() Frame {
	frame := client.framer.ReadFrame() // err
	fmt.Println(Green("recv"), frame)
	return frame
}

func GetHeadersFrame(host, path string) *HeadersFrame {
	header := http.Header{}
	header.Add("host", host)
	header.Add("method", "GET")
	header.Add("path", path)
	header.Add("scheme", "http")
	header.Add("accept", "*/*")
	header.Add("x-http2-version", Version)

	req := hpack.NewRequestContext()
	headerBlock := req.Encode(header)

	fh := &FrameHeader{
		Length:   uint16(len(headerBlock)),
		Type:     HeadersFrameType,
		Flags:    0x05,
		StreamId: 1,
	}

	headersFrame := &HeadersFrame{
		FrameHeader: fh,
		HeaderBlock: headerBlock,
		Headers:     header,
	}

	return headersFrame
}

func NoEerrorGoAwayFrame() *GoAwayFrame {
	fh := &FrameHeader{
		Length:   8,
		Type:     GoAwayFrameType,
		Flags:    0x00,
		StreamId: 0,
	}

	goAwayFrame := &GoAwayFrame{
		FrameHeader:  fh,
		LastStreamID: 0,
		ErrorCode:    NO_ERROR,
	}

	return goAwayFrame
}

func Get(url string) string {
	client := NewClient(url)
	// client.Upgrade()

	client.SendMagic()

	client.Send(NoFlowSettingsFrame()) // err

	client.Send(GetHeadersFrame(client.url.Host, client.url.Path)) // err

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
		if c > 50 {
			break
		}
		c++
	}

	client.Send(NoEerrorGoAwayFrame()) // err

	return html
}
