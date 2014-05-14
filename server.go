package http2

import (
	"crypto/tls"
	. "github.com/jxck/color"
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

var TLSNextProto = map[string]func(*http.Server, *tls.Conn, http.Handler){
	VERSION: TLSNextProtoHandler,
}

var TLSNextProtoHandler = func(server *http.Server, conn *tls.Conn, handler http.Handler) {
	Notice(Yellow("New Connection from %s"), conn.RemoteAddr())
	HandleTLSConnection(conn, handler)
}

func HandleTLSConnection(conn net.Conn, handler http.Handler) {
	Info("Handle TLS Connection")
	// defer conn.Close() TODO: not Close() only retun function

	Conn := NewConn(conn) // convert to http2.Conn
	Conn.CallBack = HandlerCallBack(handler)

	err := Conn.ReadMagic()
	if err != nil {
		log.Fatal(err)
	}

	go Conn.WriteLoop()

	// stream id 0
	zeroStream := Conn.NewStream(0)
	Conn.Streams[0] = zeroStream

	// send default settings to id 0
	settingsFrame := NewSettingsFrame(UNSET, DefaultSettings, 0)
	zeroStream.Write(settingsFrame)

	Conn.ReadLoop()
	Info("return TLSNextProto means close connection")
}

func HandlerCallBack(handler http.Handler) CallBack {
	return func(stream *Stream) {
		headerFrame := stream.Bucket.Headers[0]
		header := headerFrame.Headers

		url := &neturl.URL{
			Scheme: header.Get("scheme"),
			Host:   header.Get("authority"),
			Path:   header.Get("path"),
		}

		body := &Body{}
		if len(stream.Bucket.Data) != 0 {
			for _, data := range stream.Bucket.Data {
				_, err := body.Write(data.Data)
				if err != nil {
					log.Fatal(err)
				}
			}
		}

		req := &http.Request{
			Method:        header.Get("method"),
			URL:           url,
			Proto:         "HTTP/1.1",
			ProtoMajor:    1,
			ProtoMinor:    1,
			Header:        header,
			Body:          body,
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
		// each DataFrame has data in window size
		data := res.body.Bytes()
		length := len(data)
		window := 4096
		for i := 0; ; i++ {
			start := i * window
			end := start + window
			if end > length {
				dataFrame := NewDataFrame(UNSET, stream.Id)
				dataFrame.Data = data[start:]
				dataFrame.Length = uint16(len(dataFrame.Data))
				stream.Write(dataFrame)
				break
			}
			dataFrame := NewDataFrame(UNSET, stream.Id)
			dataFrame.Data = data[start:end]
			dataFrame.Length = uint16(len(dataFrame.Data))
			stream.Write(dataFrame)
		}

		// End Stream in empty DATA Frame
		endDataFrame := NewDataFrame(END_STREAM, stream.Id)
		stream.Write(endDataFrame)
	}
}
