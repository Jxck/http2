package http2

import (
	//"github.com/jxck/hpack"
	. "github.com/jxck/http2/frame"
	. "github.com/jxck/logger"
	"log"
	"net"
	"net/http"
	//neturl "net/url"
	//"strconv"
	"time"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

var DefaultSettings = map[SettingsId]uint32{
	SETTINGS_MAX_CONCURRENT_STREAMS: 100,
	SETTINGS_INITIAL_WINDOW_SIZE:    DEFAULT_WINDOW_SIZE,
}

func HandleTLSConnection(conn net.Conn, h http.Handler) {
	Info("Handle TLS Connection")
	defer conn.Close() // err

	Conn := NewConn(conn) // convert to http2.Conn
	_ = Conn

	time.Sleep(time.Second)
	/*
		handler.Conn.ReadFrame()

		// TODO: fill in other params
		url := &neturl.URL{
			Scheme: header.Get("scheme"),
			Host:   header.Get("authority"),
			Path:   header.Get("path"),
		}

		req := &http.Request{
			Method:        header.Get("method"),
			URL:           url,
			Proto:         "HTTP/1.1",
			ProtoMajor:    1,
			ProtoMinor:    1,
			Header:        header,
			Body:          nil,
			ContentLength: 0,
			// TransferEncoding []string
			Close: false,
			Host:  header.Get("Authority"),
		}

		// log.Println(RequestString(req))

		// Handle HTTP
		res := NewResponseWriter()
		handler.Handler.ServeHTTP(res, req)
		responseHeader := res.Header()
		responseHeader.Add(":status", strconv.Itoa(res.status))

		// Send HEADERS
		frame := NewHeadersFrame(END_HEADERS, 1)
		frame.Headers = responseHeader

		headerSet := hpack.ToHeaderSet(responseHeader)
		frame.HeaderBlock = handler.Conn.HpackContext.Encode(headerSet)
		frame.Length = uint16(len(frame.HeaderBlock))
		handler.Conn.WriteFrame(frame)

		// Send DATA
		data := NewDataFrame(0, 1)
		data.Data = res.body.Bytes()
		data.Length = uint16(len(data.Data))
		handler.Conn.WriteFrame(data)

		data = NewDataFrame(END_STREAM, 1)
		handler.Conn.WriteFrame(data)

		for i := 0; i < 30; i++ {
			frame := handler.Conn.ReadFrame()
			_, ok := frame.(*GoAwayFrame)
			if ok {
				break
			}
		}
	*/
	return
}
