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

	// 読み込んだフレームでエラーがあったら、
	// ReadLoop を抜けてここに来る。
	// その場合、 Close() を呼ぶ。
	// これはストリームやチャネルを閉じるだけで、
	// net.Conn は close しない
	Conn.Close()

	Info("return TLSNextProto will close connection")
	return
}

// handler を受け取って、将来 stream が渡されたら
// その Bucket につめられた Headers/Data フレームから
// req/res を作って handler を実行する関数を生成
func HandlerCallBack(handler http.Handler) CallBack {
	return func(stream *Stream) {
		header := stream.Bucket.Headers

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

		Info("\n%s", Lime(util.RequestString(req)))

		// Handle HTTP using handler
		res := NewResponseWriter()
		handler.ServeHTTP(res, req)
		responseHeader := res.Header()
		responseHeader.Add(":status", strconv.Itoa(res.status))

		Info("\n%s", Aqua((res.String())))

		// Send response headers as HEADERS Frame
		headerList := hpack.ToHeaderList(responseHeader)
		headerBlockFragment := stream.HpackContext.Encode(*headerList)
		Debug("%v", headerList)

		headersFrame := NewHeadersFrame(END_HEADERS, stream.ID, nil, headerBlockFragment, nil)
		headersFrame.Headers = responseHeader

		stream.Write(headersFrame)

		// Send response body as DATA Frame
		// each DataFrame has data in window size
		data := res.body.Bytes()
		maxFrameSize := stream.PeerSettings[SETTINGS_MAX_FRAME_SIZE]
		rest := int32(len(data))
		frameSize := rest

		// MaxFrameSize を基準に考え、そこから送れるサイズまで減らして行く
		for {
			Debug("rest data size(%v), current peer(%v) window(%v)", rest, stream.ID, stream.Window)

			// 送り終わってれば終わり
			if rest == 0 {
				break
			}

			frameSize = stream.Window.Consumable(rest)

			if frameSize <= 0 {
				Debug("peer stream(%v) blocked with full window\n", stream.ID)
				continue
			}

			// MaxFrameSize より大きいなら切り詰める
			if frameSize > maxFrameSize {
				frameSize = maxFrameSize
			}

			Debug("send %v/%v data", frameSize, rest)

			// ここまでに算出した frameSize 分のデータを DATA Frame を作って送る
			dataFrame := NewDataFrame(UNSET, stream.ID, data[:frameSize], nil)
			stream.Write(dataFrame)

			// 送った分を削る
			rest -= frameSize
			copy(data, data[frameSize:])
			data = data[:rest]

			// Peer の Window Size を減らす
			stream.Window.ConsumePeer(frameSize)
		}

		// End Stream in empty DATA Frame
		endDataFrame := NewDataFrame(END_STREAM, stream.ID, nil, nil)
		stream.Write(endDataFrame)
	}
}
