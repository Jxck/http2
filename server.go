package http2

import (
	. "github.com/jxck/color"
	"log"
	"net"
	"net/http"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

func ListenAndServe(addr string, handler http.Handler) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	for c := 0; c < 50; c++ {
		conn, err := listener.Accept()
		log.Printf(Cyan("New connection from %s\n"), conn.RemoteAddr())
		if err != nil {
			return err
		}
		go HandleConnection(conn)
	}

	return nil
}

func HandleConnection(conn net.Conn) {
	defer conn.Close()
	Conn := NewConn(conn)
	Conn.ReadRequest()

	for {
		frame := Conn.ReadFrame()
		log.Println(frame)
	}
}
