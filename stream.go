package http2

import (
	"bytes"
	. "github.com/jxck/color"
	. "github.com/jxck/http2/frame"
	. "github.com/jxck/logger"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

type State int

const (
	IDLE State = iota
	RESERVED_LOCAL
	RESERVED_REMOTE
	OPEN
	HALF_CLOSED_LOCAL
	HALF_CLOSED_REMOTE
	CLOSED
)

type Stream struct {
	Id         uint32
	Conn       *Conn
	State      State
	WindowSize uint32
	req        *http.Request
	FromConn   chan Frame
	ToConn     chan Frame
}

func NewStream(id uint32, c *Conn, windowSize uint32) *Stream {
	return &Stream{
		Id:         id,
		Conn:       c,
		State:      IDLE,
		WindowSize: windowSize,
		FromConn:   make(chan Frame),
		ToConn:     c.WriteChan,
	}
}

// send frame using Conn.WriteFrame
func (stream *Stream) send(frame Frame) {
	stream.ToConn <- frame // err
}

// receive frame using Conn.ReadFrame
func (stream *Stream) recv() Frame {
	frame := stream.Conn.ReadFrame() // err
	return frame
}

// send http.request as HEADER & DATA frame
func (stream *Stream) SendRequest(req *http.Request) {
	stream.req = req

	var flags uint8
	var nextstate State
	if req.Method == "GET" {
		flags = END_STREAM + END_HEADERS
		nextstate = HALF_CLOSED_LOCAL
	} else if req.Method == "POST" {
		flags = END_HEADERS
		nextstate = OPEN
	}

	// send request header via HEADERS Frame
	frame := NewHeadersFrame(flags, stream.Id)
	frame.Headers = req.Header
	frame.HeaderBlock = stream.Conn.EncodeHeader(frame.Headers)
	frame.Length = uint16(len(frame.HeaderBlock))
	stream.send(frame) // err
	stream.State = nextstate

	// if request has body data
	// send it via DATA Frame
	if req.Body != nil {
		// send payload with DATA Frame
		data := NewDataFrame(UNSET, stream.Id)
		data.Data, _ = ioutil.ReadAll(req.Body) // err
		data.Length = uint16(len(data.Data))
		stream.send(data)

		// send END_STREAM with empty body
		data = NewDataFrame(END_STREAM, stream.Id)
		stream.send(data)
		stream.State = HALF_CLOSED_LOCAL
	}
}

// receive frame and parse to http.Response
func (stream *Stream) RecvResponse() *http.Response {
	looplimit := 100
	resHeader := *new(http.Header)
	resBody := bytes.NewBuffer(make([]byte, 0))

	for {
		// receive frame
		frame := stream.recv()

		switch frametype := frame.(type) {
		case *HeadersFrame:
			// if HEADERS Frame
			headersFrame := frametype
			resHeader = stream.Conn.DecodeHeader(headersFrame.HeaderBlock)
			headersFrame.Headers = resHeader

		case *DataFrame:
			// if DATA Frame
			dataFrame := frametype
			resBody.Write(dataFrame.Data)
			stream.WindowUpdate(dataFrame.Length)
			if dataFrame.Flags == END_STREAM {
				// END_STREAM
				goto BREAK
			}

		case *SettingsFrame:
			// if SETTINGS Frame
			settingsFrame := frametype
			if settingsFrame.Flags == 0 {
				// Apply Settings

				// send ACK
				stream.Conn.SendSettingsAck()
			} else if settingsFrame.Flags == 1 {
				// receive ACK
				// TODO: Apply Settings
			}

		case *GoAwayFrame:
			Info("%s", Yellow("go away"))
		}

		// Limitter for avoid infini loop ;p
		if looplimit < 0 {
			Error("over run (loop limit = %v)", looplimit)
			break
		}
		looplimit--
	}
BREAK:

	status := resHeader.Get("Status")
	statuscode, _ := strconv.Atoi(status) // err

	// build http response
	res := &http.Response{ // TODO
		Status:           status + http.StatusText(statuscode),
		StatusCode:       statuscode,
		Proto:            Version,
		ProtoMajor:       2,
		ProtoMinor:       0,
		Header:           resHeader,
		Body:             ioutil.NopCloser(resBody),
		ContentLength:    int64(resBody.Len()),
		TransferEncoding: nil,
		Close:            false,
		Trailer:          nil,
		Request:          stream.req,
	}
	return res
}

// send WINDOW_UPDATE for flow control
func (stream *Stream) WindowUpdate(size uint16) {
	threshold := DEFAULT_WINDOW_SIZE / 2
	s := uint32(size)
	stream.WindowSize -= s
	if stream.WindowSize < threshold {
		frame := NewWindowUpdateFrame(threshold, stream.Id)
		stream.send(frame) // err
		stream.WindowSize += threshold
	}
	stream.Conn.WindowSize -= s
	if stream.Conn.WindowSize < threshold {
		stream.Conn.SendWindowUpdate(threshold)
		stream.Conn.WindowSize += threshold
	}
}
