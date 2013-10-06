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
const DefaultSettingsBase64 string = "AAAABAAAAGQAAAAHAAD__w"

func init() {
	log.SetFlags(log.Lshortfile)
}

type Client struct {
	url  *URL
	bw   *bufio.Writer
	br   *bufio.Reader
	conn *Conn
}

func (client *Client) Connect(url string) {
	client.url, _ = NewURL(url) // err

	var conn net.Conn
	if client.url.Scheme == "http" {
		conn, _ = net.Dial("tcp", client.url.Host+":"+client.url.Port) // err
	} else {
		log.Fatal("not support yet")
	}

	client.bw = bufio.NewWriter(conn)
	client.br = bufio.NewReader(conn)
	client.conn = NewConn(conn)
}

func (client *Client) Upgrade() {
	upgrade := "" +
		"GET " + client.url.Path + " HTTP/1.1\r\n" +
		"Host: " + client.url.Host + "\r\n" +
		"Connection: Upgrade, HTTP2-Settings\r\n" +
		"Upgrade: " + Version + "\r\n" +
		"HTTP2-Settings: " + DefaultSettingsBase64 + "\r\n" +
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
	client.conn.WriteFrame(frame) // err
}

func (client *Client) Recv() Frame {
	frame := client.conn.ReadFrame() // err
	fmt.Println(Green("recv"), frame)
	return frame
}

func NewHeader(host, path string) http.Header {
	header := http.Header{}
	header.Add("host", host)
	header.Add("method", "GET")
	header.Add("path", path)
	header.Add("scheme", "http")
	header.Add("accept", "*/*")
	header.Add("x-http2-version", Version)
	return header
}

func CreateWindowUpdateFrame(size, streamId uint32) *WindowUpdateFrame {
	fh := &FrameHeader{
		Length:   4,
		Type:     WindowUpdateFrameType,
		StreamId: streamId,
	}
	frame := &WindowUpdateFrame{}
	frame.FrameHeader = fh
	frame.WindowSizeIncrement = size
	return frame
}

func GetHeadersFrame(header http.Header) *HeadersFrame {
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

func Get(url string, upgrade bool) string {
	client := &Client{}
	client.Connect(url)

	if upgrade {
		client.Upgrade()
		client.SendMagic()
		settings := map[SettingsId]uint32{
			SETTINGS_MAX_CONCURRENT_STREAMS: 100,
			SETTINGS_INITIAL_WINDOW_SIZE:    65535,
		}
		client.Send(NewSettingsFrame(settings, 0)) // err
	} else {
		client.SendMagic()
		settings := map[SettingsId]uint32{
			SETTINGS_MAX_CONCURRENT_STREAMS: 100,
			SETTINGS_INITIAL_WINDOW_SIZE:    65535,
			SETTINGS_FLOW_CONTROL_OPTIONS:   1,
		}
		client.Send(NewSettingsFrame(settings, 0)) // err
		header := NewHeader(client.url.Host, client.url.Path)
		client.Send(GetHeadersFrame(header)) // err
	}

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
