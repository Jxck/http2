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

	// TODO: parse/check settings
	log.Println(req.Header.Get("Connection"))
	log.Println(req.Header.Get("Upgrade"))
	log.Println(req.Header.Get("Http2-Settings"))

	upgrade := "HTTP/1.1 101 Switching Protocols\r\n" +
		"Connection: Upgrade\r\n" +
		"Upgrade: HTTP-draft-06/2.0\r\n" +
		"\r\n"

	Conn.WriteString(upgrade)

	// SEND SETTINGS
	settings := map[SettingsId]uint32{
		SETTINGS_MAX_CONCURRENT_STREAMS: 100,
		SETTINGS_INITIAL_WINDOW_SIZE:    DEFAULT_WINDOW_SIZE,
	}
	Conn.SendSettings(settings)

	Conn.ReadString()

	// SEND HEADERS
	stream := Conn.NewStream()
	header := http.Header{}
	header.Add("status", "200")
	header.Add("content-type", "text/plain")

	frame := NewHeadersFrame(END_HEADERS, 1)
	frame.Headers = header
	frame.HeaderBlock = stream.Conn.ResponseContext.Encode(header)
	frame.Length = uint16(len(frame.HeaderBlock))
	stream.Send(frame) // err

	// SEND DATA
	data := NewDataFrame(0, 1)
	data.Data = []byte("hello world")
	data.Length = uint16(len(data.Data))
	stream.Send(data)

	data = NewDataFrame(END_STREAM, stream.Id)
	stream.Send(data)

	for c := 0; c < 4; c++ {
		frame := Conn.ReadFrame()
		_, ok := frame.(*GoAwayFrame)
		if ok {
			break
		}
	}
	return
}
