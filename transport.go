package http2

import (
	"bytes"
	"crypto/tls"
	. "github.com/Jxck/color"
	. "github.com/Jxck/http2/frame"
	. "github.com/Jxck/logger"
	"log"
	"net/http"
	"strconv"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

// Transport implements http.RoundTriper
// with RoundTrip(request) response
type Transport struct {
	URL      *URL
	Conn     *Conn
	CertPath string
	KeyPath  string
}

// connect tcp connection with host
func (transport *Transport) Connect() {
	address := transport.URL.Host + ":" + transport.URL.Port

	// loading key pair
	cert, err := tls.LoadX509KeyPair(transport.CertPath, transport.KeyPath)
	if err != nil {
		log.Fatal(err)
	}

	// setting TLS config
	config := tls.Config{
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true,
		NextProtos:         []string{VERSION},
	}
	conn, err := tls.Dial("tcp", address, &config)
	if err != nil {
		log.Fatal(err)
	}

	// check connection state
	state := conn.ConnectionState()
	Info("%v %v", Yellow("handshake"), state.HandshakeComplete)
	Info("%v %v", Yellow("protocol"), state.NegotiatedProtocol)

	Conn := NewConn(conn)

	// send Magic Octet
	err = Conn.WriteMagic()
	if err != nil {
		log.Fatal(err)
	}

	go Conn.ReadLoop()
	go Conn.WriteLoop()

	// stream id 0
	zeroStream := Conn.NewStream(0)
	Conn.Streams[0] = zeroStream

	// send default settings to id 0
	settingsFrame := NewSettingsFrame(UNSET, DefaultSettings, 0)
	zeroStream.Write(settingsFrame)

	transport.Conn = Conn
}

// http.RoundTriper implementation
func (transport *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	transport.URL, _ = NewURL(req.URL.String()) // err

	// establish tcp connection and handshake
	transport.Connect()

	callback, response := TransportCallBack(req)
	transport.Conn.CallBack = callback

	// create stream
	stream := transport.Conn.NewStream(<-NextClientStreamId)
	req = util.UpdateRequest(req, transport.URL)

	// send request header via HEADERS Frame
	var flags uint8 = END_STREAM + END_HEADERS
	frame := NewHeadersFrame(flags, stream.Id)
	frame.Headers = req.Header
	frame.HeaderBlock = stream.EncodeHeader(frame.Headers)
	frame.Length = uint16(len(frame.HeaderBlock))
	stream.Write(frame) // err

	// send GOAWAY
	// stream.Write(NewGoAwayFrame(0, NO_ERROR, 0))

	//return res, nil
	res := <-response

	return res, nil
}

func TransportCallBack(req *http.Request) (CallBack, chan *http.Response) {
	response := make(chan *http.Response)
	return func(stream *Stream) {
		headerFrame := stream.Bucket.Headers[0]

		var data bytes.Buffer
		for _, dataFrame := range stream.Bucket.Data {
			data.Write(dataFrame.Data)
		}

		body := &Body{data}

		headers := headerFrame.Headers

		status, _ := strconv.Atoi(headers.Get("status")) // err
		res := &http.Response{
			Status:        http.StatusText(status),
			StatusCode:    status,
			Proto:         "HTTP/1.1",
			ProtoMajor:    1,
			ProtoMinor:    1,
			Header:        headers,
			Body:          body,
			ContentLength: int64(body.Len()),
			// TransferEncoding []string
			// Close bool
			// Trailer Header
			Request: req,
		}

		response <- res

	}, response
}
