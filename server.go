package http2

import (
	"bytes"
	"crypto/tls"
	"fmt"
	. "github.com/jxck/color"
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

const (
	// TODO: move to arg
	DEFAULT_CERT string = "keys/cert.pem"
	DEFAULT_KEY         = "keys/key.pem"
)

// TODO: move to arg
var SSL bool = true

type Server struct {
	listener net.Listener
	addr     string
}

func (s *Server) Listen() {
	s.listener, _ = net.Listen("tcp", s.addr)
}

func (s *Server) ListenTLS() {
	// loading key pair
	cert, err := tls.LoadX509KeyPair(DEFAULT_CERT, DEFAULT_KEY)
	if err != nil {
		log.Fatal(err)
	}

	// setting TLS config
	config := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true,
		NextProtos:         []string{Version},
	}

	s.listener, _ = tls.Listen("tcp", s.addr, config)
}

func ListenAndServe(addr string, handler http.Handler) (err error) {

	server := &Server{
		addr: addr,
	}
	if SSL {
		server.ListenTLS()
	} else {
		server.Listen()
	}

	Info(Yellow("server starts on port %s"), addr)

	for c := 0; c < 10; c++ {
		conn, err := server.listener.Accept()
		if err != nil {
			return err
		}
		Info(Yellow("New connection from %s\n"), conn.RemoteAddr())
		go HandleConnection(conn, handler)
	}

	return nil
}

func HandleConnection(conn net.Conn, handler http.Handler) {
	Info("Handle Connection")
	defer conn.Close() // err

	// convert to http2.Conn
	Conn := NewConn(conn)

	req := Conn.ReadRequest()

	// TODO: parse/check settings
	Debug("%s", req.Header.Get("Http2-Settings"))

	upgrade := fmt.Sprintf(""+
		"HTTP/1.1 101 Switching Protocols\r\n"+
		"Connection: Upgrade\r\n"+
		"Upgrade: %v\r\n"+
		"\r\n",
		Version)

	Conn.WriteString(upgrade)

	// Send SETTINGS
	settings := map[SettingsId]uint32{
		SETTINGS_MAX_CONCURRENT_STREAMS: 100,
		SETTINGS_INITIAL_WINDOW_SIZE:    DEFAULT_WINDOW_SIZE,
	}
	Conn.SendSettings(settings)

	// Read Magic
	Conn.ReadString()

	fin := make(chan bool)

	// Send Routine
	go func() {
		// Send HEADERS
		stream := Conn.NewStream()
		header := http.Header{}
		header.Add("status", "200")
		header.Add("content-type", "text/plain")

		res := &Response{
			Headers: header,
			Body:    bytes.NewBuffer(make([]byte, 0)),
		}

		handler.ServeHTTP(res, req)

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
	}()

	// Recv Routine
	go func() {
		for c := 0; c < 4; c++ {
			frame := Conn.ReadFrame()
			_, ok := frame.(*GoAwayFrame)
			if ok {
				break
			}
		}
		fin <- true
	}()
	<-fin
	return
}
