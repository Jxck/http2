package http2

import (
	"bytes"
	"crypto/tls"
	. "github.com/Jxck/color"
	. "github.com/Jxck/http2/frame"
	. "github.com/Jxck/logger"
	"net/http"
	"strconv"
)

// Transport implements http.RoundTriper
// with RoundTrip(request) response
type Transport struct {
	Conn     *Conn
	CertPath string
	KeyPath  string
}

// connect tcp connection with host
func (transport *Transport) Connect(url *URL) (err error) {
	address := url.Host + ":" + url.Port

	// loading key pair
	cert, err := tls.LoadX509KeyPair(transport.CertPath, transport.KeyPath)
	if err != nil {
		return err
	}

	// setting TLS config
	config := tls.Config{
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true,
		NextProtos:         []string{VERSION},
	}
	conn, err := tls.Dial("tcp", address, &config)
	if err != nil {
		return err
	}

	// check connection state
	state := conn.ConnectionState()
	Info("%v %v", Yellow("handshake"), state.HandshakeComplete)
	Info("%v %v", Yellow("protocol"), state.NegotiatedProtocol)

	Conn := NewConn(conn)

	// send Magic Octet
	err = Conn.WriteMagic()
	if err != nil {
		return err
	}

	go Conn.ReadLoop()
	go Conn.WriteLoop()

	// stream id 0
	zeroStream := Conn.NewStream(0)
	Conn.Streams[0] = zeroStream

	// send default settings to id 0
	settingsFrame := NewSettingsFrame(UNSET, 0, DefaultSettings)
	zeroStream.Write(settingsFrame)

	transport.Conn = Conn
	return
}

// http.RoundTriper implementation
func (transport *Transport) RoundTrip(req *http.Request) (res *http.Response, err error) {
	url, err := NewURL(req.URL.String()) // err
	if err != nil {
		Error("%v", err)
		return nil, err
	}
	req = util.UpdateRequest(req, url)

	// establish tcp connection and handshake
	err = transport.Connect(url)
	if err != nil {
		Error("%v", err)
		return nil, err
	}

	callback, response := TransportCallBack(req)
	transport.Conn.CallBack = callback

	// create stream
	stream := transport.Conn.NewStream(<-NextClientStreamId)

	// send request header via HEADERS Frame
	var flags Flag = END_STREAM + END_HEADERS
	headerBlock := stream.EncodeHeader(req.Header)
	Trace("encoded header block %v", headerBlock)
	frame := NewHeadersFrame(flags, stream.Id, nil, headerBlock, nil)
	frame.Headers = req.Header
	stream.Write(frame) // TODO: err

	res = <-response

	// TODO: send GOAWAY
	// stream.Write(NewGoAwayFrame(0, stream.Id, NO_ERROR, nil))

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
