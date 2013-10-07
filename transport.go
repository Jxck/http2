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
	LastStreamId uint32
	url          *URL
	bw           *bufio.Writer
	br           *bufio.Reader
	conn         *Conn
	Upgrade      bool
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

func (transport *Transport) SendUpgrade() *Stream {
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
	stream := &Stream{
		Id:   1,
		Conn: transport.conn,
	}
	return stream
}

func (transport *Transport) SendMagic() {
	transport.bw.WriteString(MagicString) // err
	transport.bw.Flush()                  // err
	fmt.Println(Yellow("Send MagicOctet"))
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

type Stream struct {
	Id   uint32
	Conn *Conn
	req  *http.Request
}

func (stream *Stream) Send(frame Frame) {
	stream.Conn.WriteFrame(frame) // err
	fmt.Println(Red("send"), frame)
}

func (stream *Stream) Recv() Frame {
	frame := stream.Conn.ReadFrame() // err
	fmt.Println(Green("recv"), frame)
	return frame
}

func (stream *Stream) SendRequest(req *http.Request) {
	stream.req = req
	headerBlock := stream.Conn.EncodeHeader(req.Header)
	frame := NewHeadersFrame(req.Header, headerBlock, 0x05, stream.Id)
	stream.Send(frame) // err
}

func (stream *Stream) ReadResponse() *http.Response {
	c := 0
	header := http.Header{}
	resBody := bytes.NewBuffer([]byte{})

	for {
		frame := stream.Recv()
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

	stream.Send(NewGoAwayFrame(0, NO_ERROR, 0)) // err
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
		Request:          stream.req,
	}
	return res
}

func (transport *Transport) NewStream() *Stream {
	stream := &Stream{
		Id:   transport.LastStreamId, // TODO: transport.GetNextID()
		Conn: transport.conn,
	}
	transport.LastStreamId += 2
	return stream
}

func (transport *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	transport.url, _ = NewURL(req.URL.String())
	transport.Connect(req.URL.String())

	var stream *Stream
	if transport.Upgrade {
		stream = transport.SendUpgrade()
		transport.SendMagic()
		settings := map[SettingsId]uint32{
			SETTINGS_MAX_CONCURRENT_STREAMS: 100,
			SETTINGS_INITIAL_WINDOW_SIZE:    65535,
		}
		stream.Send(NewSettingsFrame(settings, 0)) // err
	} else {
		transport.SendMagic()
		stream = transport.NewStream()
		settings := map[SettingsId]uint32{
			SETTINGS_MAX_CONCURRENT_STREAMS: 100,
			SETTINGS_INITIAL_WINDOW_SIZE:    65535,
			SETTINGS_FLOW_CONTROL_OPTIONS:   1,
		}
		stream.Send(NewSettingsFrame(settings, 0)) // err
		stream.SendRequest(req)
	}

	res := stream.ReadResponse()
	return res, nil
}
