package http2

import (
	"bufio"
	"bytes"
	"fmt"
	. "github.com/jxck/color"
	"io/ioutil"
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
	url     *URL
	bw      *bufio.Writer
	br      *bufio.Reader
	conn    *Conn
	Upgrade bool
}

func (client *Client) Connect(url string) {
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

func (client *Client) SendUpgrade() {
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

func (client *Client) RoundTrip(req *http.Request) (*http.Response, error) {
	client.url, _ = NewURL(req.URL.String())
	client.Connect(req.URL.String())

	if client.Upgrade {
		client.SendUpgrade()
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
		headerBlock := client.conn.EncodeHeader(header)
		frame := NewHeadersFrame(header, headerBlock, 0x05, 1)
		client.Send(frame) // err
	}

	c := 0
	header := http.Header{}
	resBody := bytes.NewBuffer([]byte{})

	for {
		frame := client.Recv()
		frameHeader := frame.Header()

		if frameHeader.Type == HeadersFrameType {
			headersFrame := frame.(*HeadersFrame)
			header = headersFrame.Headers
		}

		if frameHeader.Type == DataFrameType {
			dataFrame := frame.(*DataFrame)
			resBody.Write(dataFrame.Data)
		}

		if frameHeader.Flags == 0x1 {
			break
		}
		if c > 50 {
			break
		}
		c++
	}

	client.Send(NewGoAwayFrame(0, NO_ERROR, 0)) // err

	res := &http.Response{ // TODO
		Status:           "200 OK",
		StatusCode:       200,
		Proto:            Version,
		ProtoMajor:       2,
		ProtoMinor:       0,
		Header:           header,
		Body:             ioutil.NopCloser(resBody),
		ContentLength:    int64(resBody.Len()),
		TransferEncoding: nil,
		Close:            false,
		Trailer:          nil,
		Request:          req,
	}

	return res, nil
}
