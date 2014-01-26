package http2

import (
	"github.com/jxck/hpack"
	. "github.com/jxck/http2/frame"
	. "github.com/jxck/logger"
	"log"
	"net"
	"net/http"
	neturl "net/url"
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

func HandleTLSConnection(conn net.Conn, h http.Handler) {
	Info("Handle TLS Connection")
	defer conn.Close() // err

	handler := &Handler{
		Conn:    NewConn(conn), // convert to http2.Conn
		Handler: h,
	}

	// Read Magic
	handler.Conn.ReadMagic()

	// Send SETTINGS
	handler.Conn.SendSettings(defaultSettings)

	// Recv SETTINGS
	handler.Conn.ReadFrame(hpack.REQUEST)
	// TODO: Send SETTINGS ACK

	// Recv HEADERS
	headers := handler.Conn.ReadFrame(hpack.REQUEST).(*HeadersFrame)
	header := headers.Headers
	log.Println(header)

	handler.Conn.ReadFrame(hpack.REQUEST)

	// TODO: fill in other params
	url := &neturl.URL{
		Scheme: header.Get("scheme"),
		Host:   header.Get("authority"),
		Path:   header.Get("path"),
	}

	req := &http.Request{
		Method:     header.Get("method"),
		URL:        url,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 0,
		Header:     header,
		// Body io.ReadCloser
		ContentLength: 0,
		// TransferEncoding []string
		Close: false,
		Host:  header.Get("Authority"),
	}

	handler.Handler.ServeHTTP(res, req)

	//frame := NewHeadersFrame(END_HEADERS, 1)
	//frame.Headers = header

	//headerSet := hpack.ToHeaderSet(header)
	//frame.HeaderBlock = stream.Conn.ResponseContext.Encode(headerSet)
	//frame.Length = uint16(len(frame.HeaderBlock))
	//stream.Send(frame) // err

	//// Send DATA
	//data := NewDataFrame(0, 1)
	//data.Data = res.Body.Bytes()
	//data.Length = uint16(len(data.Data))
	//stream.Send(data)

	//data = NewDataFrame(END_STREAM, stream.Id)
	//stream.Send(data)

	//fin := make(chan bool)
	//for i := 0; i < 30; i++ {
	//	frame := handler.Conn.ReadFrame(hpack.REQUEST)
	//	_, ok := frame.(*GoAwayFrame)
	//	if ok {
	//		fin <- true
	//		break
	//	}
	//}
	//<-fin
	return
}
