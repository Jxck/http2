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

	for c := 0; c < 10; c++ {
		conn, err := listener.Accept()
		if err != nil {
			log.Println(err)
			return err
		}
		log.Printf(Cyan("New connection from %s\n"), conn.RemoteAddr())
		go HandleConnection(conn)
	}

	return nil
}

func HandleConnection(conn net.Conn) {
	log.Println("Handle Connection")
	defer conn.Close()
	Conn := NewConn(conn)
	req := Conn.ReadRequest()
	log.Println(req)
	//	for {
	//		frame := Conn.ReadFrame()
	//		log.Println(frame)
	//	}
	return
}
