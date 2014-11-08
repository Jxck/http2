package http2

import (
	"crypto/tls"
	"fmt"
	. "github.com/Jxck/color"
	"github.com/Jxck/hpack"
	. "github.com/Jxck/http2/frame"
	. "github.com/Jxck/logger"
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
	return // return closes connection
}

func HandleTLSConnection(conn net.Conn, handler http.Handler) {
	Info("Handle TLS Connection")
	// do not call "defer conn.Close()" only retun function

	Conn := NewConn(conn) // convert net.Conn to http2.Conn

	// http.Handler が req, res を必要とするので
	// stream がそれを生成して、その stream を渡すことで
	// req/res が用意できたタイミングで handler を呼ぶコールバックを
	// 生成し Conn に持っておく。
	Conn.CallBack = HandlerCallBack(handler)

	err := Conn.ReadMagic()
	if err != nil {
		Error("%v", err)
		return
	}

	// 別 goroutine で WriteChann に送った
	// frame を書き込むループを回す
	go Conn.WriteLoop()

	// send default settings to id 0
	settingsFrame := NewSettingsFrame(UNSET, 0, DefaultSettings)
	Conn.WriteChan <- settingsFrame

	// 送られてきた frame を読み出すループを回す
	// ここで block する。
	Conn.ReadLoop()

	Info("return TLSNextProto means close connection")
	return
}

// handler を受け取って、将来 stream が渡されたら
// その Bucket につめられた Headers/Data フレームから
// req/res を作って handler を実行する関数を生成
func HandlerCallBack(handler http.Handler) CallBack {
	return func(stream *Stream) {
		headerFrame := stream.Bucket.Headers[0]
		header := headerFrame.Headers

		authority := header.Get(":authority")
		method := header.Get(":method")
		path := header.Get(":path")
		scheme := header.Get(":scheme")

		header.Del(":authority")
		header.Del(":method")
		header.Del(":path")
		header.Del(":scheme")

		rawurl := fmt.Sprintf("%s://%s%s", scheme, authority, path)
		url, err := neturl.ParseRequestURI(rawurl)
		if err != nil {
			Fatal("%v", err)
		}

		// request body がある場合
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
			Method:           method,
			URL:              url,
			Proto:            "HTTP/1.1",
			ProtoMajor:       1,
			ProtoMinor:       1,
			Header:           header,
			Body:             body,
			ContentLength:    int64(body.Buffer.Len()),
			TransferEncoding: []string{}, // TODO:
			Close:            false,
			Host:             authority,
		}

		Debug("\n%s", White(util.RequestString(req)))

		// Handle HTTP using handler
		res := NewResponseWriter()
		handler.ServeHTTP(res, req)
		responseHeader := res.Header()
		responseHeader.Add(":status", strconv.Itoa(res.status))

		Debug("\n%s", White(res.String()))

		// Send response headers as HEADERS Frame
		headerList := hpack.ToHeaderList(responseHeader)
		headerBlock := stream.HpackContext.Encode(*headerList)
		Debug("%v", headerList)

		headersFrame := NewHeadersFrame(END_HEADERS, stream.ID, nil, headerBlock, nil)
		headersFrame.Headers = responseHeader

		stream.Write(headersFrame)

		// Send response body as DATA Frame
		// each DataFrame has data in window size
		data := res.body.Bytes()
		frameSize := int(stream.PeerSettings[SETTINGS_MAX_FRAME_SIZE])

		// MaxFrameSize ごとに分けて送る
		for {
			log.Printf("current window of stream(%v) = %v\n", stream.ID, stream.Window.PeerCurrentSize)
			if len(data) == 0 {
				break
			}

			// window size が足りなかったら送らない
			if stream.Window.PeercurrentSize == 0 {
				continue
			}	

			// MaxFrameSize より小さいなら全部送る
			if frameSize > len(data) {
				frameSize = len(data)
			}

			if frameSize > 

			dataFrame := NewDataFrame(UNSET, stream.ID, data[:frameSize], nil)
			stream.Write(dataFrame)
			copy(data, data[frameSize:])
			data = data[:len(data)-frameSize]
		}

		// End Stream in empty DATA Frame
		endDataFrame := NewDataFrame(END_STREAM, stream.ID, nil, nil)
		stream.Write(endDataFrame)
	}
}
