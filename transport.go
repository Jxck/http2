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
	"strconv"
)

const Version string = "HTTP-draft-06/2.0"
const MagicString string = "PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n"
const DefaultSettingsBase64 string = "AAAABAAAAGQAAAAHAAD__w"

func init() {
	log.SetFlags(log.Lshortfile)
}

type Transport struct {
	url     *URL
	bw      *bufio.Writer
	br      *bufio.Reader
	conn    *Conn
	Upgrade bool
}

func (transport *Transport) Connect(url string) {
	var conn net.Conn
	if transport.url.Scheme == "http" {
		conn, _ = net.Dial("tcp", transport.url.Host+":"+transport.url.Port) // err
	} else {
		log.Fatal("not support yet")
	}

	transport.bw = bufio.NewWriter(conn)
	transport.br = bufio.NewReader(conn)
	transport.conn = NewConn(conn)
}

func (transport *Transport) SendUpgrade() {
	upgrade := "" +
		"GET " + transport.url.Path + " HTTP/1.1\r\n" +
		"Host: " + transport.url.Host + "\r\n" +
		"Connection: Upgrade, HTTP2-Settings\r\n" +
		"Upgrade: " + Version + "\r\n" +
		"HTTP2-Settings: " + DefaultSettingsBase64 + "\r\n" +
		"Accept: */*\r\n" +
		"\r\n"

	transport.bw.WriteString(upgrade) // err
	transport.bw.Flush()              // err
	fmt.Println(Blue(upgrade))

	res, _ := http.ReadResponse(transport.br, &http.Request{Method: "GET"}) // err

	fmt.Println(Blue(ResponseString(res)))
	fmt.Println(Yellow("HTTP Upgrade Success :)"))
}

func (transport *Transport) SendMagic() {
	transport.bw.WriteString(MagicString) // err
	transport.bw.Flush()                  // err
}

func (transport *Transport) Send(frame Frame) {
	fmt.Println(Red("send"), frame)
	transport.conn.WriteFrame(frame) // err
}

func (transport *Transport) Recv() Frame {
	frame := transport.conn.ReadFrame() // err
	fmt.Println(Green("recv"), frame)
	return frame
}

func (transport *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	transport.url, _ = NewURL(req.URL.String())
	transport.Connect(req.URL.String())

	if transport.Upgrade {
		transport.SendUpgrade()
		transport.SendMagic()
		settings := map[SettingsId]uint32{
			SETTINGS_MAX_CONCURRENT_STREAMS: 100,
			SETTINGS_INITIAL_WINDOW_SIZE:    65535,
		}
		transport.Send(NewSettingsFrame(settings, 0)) // err
	} else {
		transport.SendMagic()
		settings := map[SettingsId]uint32{
			SETTINGS_MAX_CONCURRENT_STREAMS: 100,
			SETTINGS_INITIAL_WINDOW_SIZE:    65535,
			SETTINGS_FLOW_CONTROL_OPTIONS:   1,
		}
		transport.Send(NewSettingsFrame(settings, 0)) // err
		header := NewHeader(transport.url.Host, transport.url.Path)
		headerBlock := transport.conn.EncodeHeader(header)
		frame := NewHeadersFrame(header, headerBlock, 0x05, 1)
		transport.Send(frame) // err
	}

	c := 0
	header := http.Header{}
	resBody := bytes.NewBuffer([]byte{})

	for {
		frame := transport.Recv()
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

	transport.Send(NewGoAwayFrame(0, NO_ERROR, 0)) // err
	status := header.Get("Status")
	statuscode, _ := strconv.Atoi(status) // err
	res := &http.Response{                // TODO
		Status:           status + http.StatusText(statuscode),
		StatusCode:       statuscode,
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
