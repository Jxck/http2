package http2

import (
	"bytes"
	"fmt"
	. "github.com/jxck/color"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

type Stream struct {
	Id   uint32
	Conn *Conn
	req  *http.Request
}

func (stream *Stream) Send(frame Frame) {
	stream.Conn.WriteFrame(frame) // err
	fmt.Println(Red("send"), Indent(frame))
}

func (stream *Stream) Recv() Frame {
	frame := stream.Conn.ReadFrame() // err
	fmt.Println(Green("recv"), Indent(frame))
	return frame
}

func (stream *Stream) SendRequest(req *http.Request) {
	stream.req = req
	frame := NewHeadersFrame(END_STREAM+END_HEADERS, stream.Id)
	frame.Headers = req.Header
	frame.HeaderBlock = stream.Conn.EncodeHeader(frame.Headers)
	frame.Length = uint16(len(frame.HeaderBlock))
	stream.Send(frame) // err
}

func (stream *Stream) RecvResponse() *http.Response {
	c := 0
	header := http.Header{}
	resBody := bytes.NewBuffer([]byte{})

	for {
		frame := stream.Recv()
		frameHeader := frame.Header()

		if frameHeader.Type == HeadersFrameType {
			headersFrame := frame.(*HeadersFrame)
			header = headersFrame.Headers
		}

		if frameHeader.Type == DataFrameType {
			dataFrame := frame.(*DataFrame)
			resBody.Write(dataFrame.Data)
		}

		if frameHeader.Flags == 0x1 {
			break
		}

		if c > 50 {
			break
		}
		c++
	}

	stream.Send(NewGoAwayFrame(0, NO_ERROR, 0)) // err
	status := header.Get("Status")
	statuscode, _ := strconv.Atoi(status) // err
	res := &http.Response{                // TODO
		Status:           status + http.StatusText(statuscode),
		StatusCode:       statuscode,
		Proto:            Version,
		ProtoMajor:       2,
		ProtoMinor:       0,
		Header:           header,
		Body:             ioutil.NopCloser(resBody),
		ContentLength:    int64(resBody.Len()),
		TransferEncoding: nil,
		Close:            false,
		Trailer:          nil,
		Request:          stream.req,
	}
	return res
}
