package http2

import (
	"bytes"
	"fmt"
	"github.com/jxck/hpack"
	. "github.com/jxck/http2/frame"
	. "github.com/jxck/logger"
	"log"
	"net"
	"net/http"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

var defaultSettings = map[SettingsId]uint32{
	SETTINGS_MAX_CONCURRENT_STREAMS: 100,
	SETTINGS_INITIAL_WINDOW_SIZE:    DEFAULT_WINDOW_SIZE,
}

type Handler struct {
	Conn    *Conn
	Handler http.Handler
}

func (handler *Handler) HandShakeSSL() {
	// Read Magic
	handler.Conn.ReadMagic()

	// Send SETTINGS
	handler.Conn.SendSettings(defaultSettings)

	handler.Conn.ReadFrame(hpack.REQUEST)
	handler.Conn.ReadFrame(hpack.REQUEST)
}

func (handler *Handler) HandShake() (req *http.Request) {
	req = handler.Conn.ReadRequest()

	// TODO: parse/check settings
	Debug("%s", req.Header.Get("Http2-Settings"))

	upgrade := fmt.Sprintf(""+
		"HTTP/1.1 101 Switching Protocols\r\n"+
		"Connection: Upgrade\r\n"+
		"Upgrade: %v\r\n"+
		"\r\n",
		Version)

	handler.Conn.WriteString(upgrade)

	// Send SETTINGS
	handler.Conn.SendSettings(defaultSettings)

	// Read Magic
	handler.Conn.ReadMagic()

	return req
}

func (handler *Handler) ServeHTTP(req *http.Request) {
	// Send HEADERS
	stream := handler.Conn.NewStream()
	header := http.Header{}
	header.Add("status", "200")
	header.Add("content-type", "text/plain")

	res := &Response{
		Headers: header,
		Body:    bytes.NewBuffer(make([]byte, 0)),
	}

	handler.Handler.ServeHTTP(res, req)

	frame := NewHeadersFrame(END_HEADERS, 1)
	frame.Headers = header

	headerSet := hpack.ToHeaderSet(header)
	frame.HeaderBlock = stream.Conn.ResponseContext.Encode(headerSet)
	frame.Length = uint16(len(frame.HeaderBlock))
	stream.Send(frame) // err

	// Send DATA
	data := NewDataFrame(0, 1)
	data.Data = res.Body.Bytes()
	data.Length = uint16(len(data.Data))
	stream.Send(data)

	data = NewDataFrame(END_STREAM, stream.Id)
	stream.Send(data)
}

func (handler *Handler) RecvLoop() {
	fin := make(chan bool)
	for {
		frame := handler.Conn.ReadFrame(hpack.REQUEST)
		_, ok := frame.(*GoAwayFrame)
		if ok {
			fin <- true
			break
		}
	}
	<-fin
	return
}

func HandleConnection(conn net.Conn, h http.Handler) {
	Info("Handle Connection")
	defer conn.Close() // err

	handler := &Handler{
		// convert to http2.Conn
		Conn:    NewConn(conn),
		Handler: h,
	}

	req := handler.HandShake()

	// Send Routine
	go handler.ServeHTTP(req)

	// Recv Routine
	handler.RecvLoop()

	return
}

func HandleTLSConnection(conn net.Conn, h http.Handler) {
	Info("Handle TLS Connection")
	defer conn.Close() // err

	handler := &Handler{
		// convert to http2.Conn
		Conn:    NewConn(conn),
		Handler: h,
	}

	req := &http.Request{}
	handler.HandShakeSSL()

	// Send Routine
	go handler.ServeHTTP(req)

	// Recv Routine
	handler.RecvLoop()

	return
}
