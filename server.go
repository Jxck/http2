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

	log.Println(req.Header.Get("Connection"))
	log.Println(req.Header.Get("Upgrade"))
	log.Println(req.Header.Get("Http2-Settings"))

	upgrade := `HTTP/1.1 101 Switching Protocols
Connection: Upgrade
Upgrade: HTTP-draft-06/2.0

`
	log.Printf("%q", upgrade)
	Conn.WriteString(upgrade)

	settings := map[SettingsId]uint32{
		SETTINGS_MAX_CONCURRENT_STREAMS: 100,
		SETTINGS_INITIAL_WINDOW_SIZE:    DEFAULT_WINDOW_SIZE,
	}
	Conn.SendSettings(settings)
	for c := 0; c < 4; c++ {
		Conn.ReadFrame()
	}

	return
}
