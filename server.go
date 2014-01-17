package http2

import (
	"crypto/tls"
	. "github.com/jxck/color"
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
