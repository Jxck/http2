package http2

import (
	"crypto/tls"
	. "github.com/Jxck/color"
	"github.com/Jxck/hpack"
	. "github.com/Jxck/http2/frame"
	. "github.com/Jxck/logger"
	"net"
	"net/http"
	neturl "net/url"
	"strconv"
)

var TLSNextProto = map[string]func(*http.Server, *tls.Conn, http.Handler){
	VERSION: TLSNextProtoHandler,
}

var TLSNextProtoHandler = func(server *http.Server, conn *tls.Conn, handler http.Handler) {
	Notice(Yellow("New Connection from %s"), conn.RemoteAddr())
	HandleTLSConnection(conn, handler)
	return // return closes connection
}

func HandleTLSConnection(conn net.Conn, handler http.Handler) {
	Info("Handle TLS Connection")
	// defer conn.Close() TODO: not Close() only retun function

	h2conn := NewConn(conn) // convert net.Conn to http2.Conn

	// http.Handler が req, res を必要とするので
	// stream がそれを生成して、その stream を渡すことで
	// req/res が用意できたタイミングで handler を呼ぶコールバックを
	// 生成し h2conn に持っておく。
	h2conn.CallBack = HandlerCallBack(handler)

	err := h2conn.ReadMagic()
	if err != nil {
		Error("%v", err)
		return
	}

	// 別 goroutine で WriteChann に送った
	// frame を書き込むループを回す
	go h2conn.WriteLoop()

	// stream id 0
	zeroStream := h2conn.NewStream(0)
	h2conn.Streams[0] = zeroStream

	// send default settings to id 0
	settingsFrame := NewSettingsFrame(UNSET, 0, DefaultSettings)
	zeroStream.Write(settingsFrame)

	// 送られてきた frame を読み出すループを回す
	// ここで block する。
	h2conn.ReadLoop()
	Info("return TLSNextProto means close connection")
}

// handler を受け取って、将来 stream が渡されたら
// その Bucket につめられた Headers/Data フレームから
// req/res を作って handler を実行する関数を生成
func HandlerCallBack(handler http.Handler) CallBack {
	return func(stream *Stream) {
		headerFrame := stream.Bucket.Headers[0]
		header := headerFrame.Headers

		url := &neturl.URL{
			Scheme: header.Get("scheme"),
			Host:   header.Get("authority"),
			Path:   header.Get("path"),
		}

		body := new(Body)
		if len(stream.Bucket.Data) != 0 {
			for _, data := range stream.Bucket.Data {
				_, err := body.Write(data.Data)
				if err != nil {
					Fatal("%v", err)
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

		// Handle HTTP using handler
		res := NewResponseWriter()
		handler.ServeHTTP(res, req)
		responseHeader := res.Header()
		responseHeader.Add(":status", strconv.Itoa(res.status))

		// Send response headers as HEADERS Frame
		headerList := hpack.ToHeaderList(responseHeader)
		headerBlock := stream.HpackContext.Encode(*headerList)

		headersFrame := NewHeadersFrame(END_HEADERS, stream.Id, nil, headerBlock, nil)
		headersFrame.Headers = responseHeader

		stream.Write(headersFrame)

		// Send response body as DATA Frame
		// each DataFrame has data in window size
		data := res.body.Bytes()
		length := len(data)
		window := int(DEFAULT_WINDOW_SIZE)
		for i := 0; ; i++ {
			start := i * window
			end := start + window
			if end > length {
				dataFrame := NewDataFrame(UNSET, stream.Id, data[start:], nil)
				stream.Write(dataFrame)
				break
			}
			dataFrame := NewDataFrame(UNSET, stream.Id, data[start:end], nil)
			stream.Write(dataFrame)
		}

		// End Stream in empty DATA Frame
		endDataFrame := NewDataFrame(END_STREAM, stream.Id, nil, nil)
		stream.Write(endDataFrame)
	}
}
