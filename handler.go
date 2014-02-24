package http2

import (
	"github.com/jxck/hpack"
	. "github.com/jxck/http2/frame"
	. "github.com/jxck/logger"
	"log"
	"net"
	"net/http"
	neturl "net/url"
	"strconv"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

var DefaultSettings = map[SettingsId]uint32{
	SETTINGS_MAX_CONCURRENT_STREAMS: 100,
	SETTINGS_INITIAL_WINDOW_SIZE:    DEFAULT_WINDOW_SIZE,
}

func HandleTLSConnection(conn net.Conn, handler http.Handler) {
	Info("Handle TLS Connection")
	defer conn.Close() // err

	Conn := NewConn(conn) // convert to http2.Conn
	Conn.CallBack = HandleBucket(handler)

	Conn.ReadMagic()

	go Conn.WriteLoop()

	// stream id 0
	zeroStream := Conn.NewStream(0)
	Conn.Streams[0] = zeroStream

	// send default settings to id 0
	settingsFrame := NewSettingsFrame(UNSET, DefaultSettings, 0)
	zeroStream.Write(settingsFrame)

	Conn.ReadLoop()
}

func HandleBucket(handler http.Handler) CallBack {
	return func(stream *Stream) {
		headerFrame := stream.Bucket.Headers[0]
		header := headerFrame.Headers

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

		Notice("%s", util.Indent(util.RequestString(req)))

		// Handle HTTP
		res := NewResponseWriter()
		handler.ServeHTTP(res, req)
		responseHeader := res.Header()
		responseHeader.Add(":status", strconv.Itoa(res.status))

		// Send HEADERS
		headersFrame := NewHeadersFrame(END_HEADERS, stream.Id)
		headersFrame.Headers = responseHeader

		headerSet := hpack.ToHeaderSet(responseHeader)
		headersFrame.HeaderBlock = stream.HpackContext.Encode(headerSet)
		headersFrame.Length = uint16(len(headersFrame.HeaderBlock))
		stream.Write(headersFrame)

		// Send DATA
		dataFrame := NewDataFrame(UNSET, stream.Id)
		dataFrame.Data = res.body.Bytes()
		dataFrame.Length = uint16(len(dataFrame.Data))
		stream.Write(dataFrame)

		// End Stream
		endDataFrame := NewDataFrame(END_STREAM, stream.Id)
		stream.Write(endDataFrame)

		stream.ChangeState(CLOSED)
	}
}
