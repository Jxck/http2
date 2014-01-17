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

func HandleConnection(conn net.Conn, handler http.Handler) {
	Info("Handle Connection")
	defer conn.Close() // err

	// convert to http2.Conn
	Conn := NewConn(conn)

	var req *http.Request
	if SSL {
		// Read Magic
		Conn.ReadMagic()

		// Send SETTINGS
		Conn.SendSettings(defaultSettings)

		Conn.ReadFrame(hpack.REQUEST)
		Conn.ReadFrame(hpack.REQUEST)

	} else {
		req = Conn.ReadRequest()

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
		Conn.SendSettings(defaultSettings)

		// Read Magic
		Conn.ReadMagic()
	}

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
			frame := Conn.ReadFrame(hpack.REQUEST)
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
