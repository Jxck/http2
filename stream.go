package http2

import (
	"bytes"
	. "github.com/jxck/color"
	. "github.com/jxck/logger"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

type Stream struct {
	Id         uint32
	Conn       *Conn
	req        *http.Request
	WindowSize uint32
}

// send frame using Conn.WriteFrame
func (stream *Stream) Send(frame Frame) {
	stream.Conn.WriteFrame(frame) // err
}

// receive frame using Conn.ReadFrame
func (stream *Stream) Recv() Frame {
	frame := stream.Conn.ReadFrame() // err
	return frame
}

func (stream *Stream) SendRequest(req *http.Request) {
	stream.req = req

	var flags uint8
	if req.Method == "GET" {
		flags = END_STREAM + END_HEADERS
	} else if req.Method == "POST" {
		flags = END_HEADERS
	}

	// send request header via HEADERS Frame
	frame := NewHeadersFrame(flags, stream.Id)
	frame.Headers = req.Header
	frame.HeaderBlock = stream.Conn.EncodeHeader(frame.Headers)
	frame.Length = uint16(len(frame.HeaderBlock))
	stream.Send(frame) // err

	// if request has body data
	// send it via DATA Frame
	if req.Body != nil {
		data := NewDataFrame(0, stream.Id)
		data.Data, _ = ioutil.ReadAll(req.Body) // err
		data.Length = uint16(len(data.Data))
		stream.Send(data)

		data = NewDataFrame(END_STREAM, stream.Id)
		stream.Send(data)
	}
}

func (stream *Stream) RecvResponse() *http.Response {
	looplimit := 0
	resHeader := *new(http.Header)
	resBody := bytes.NewBuffer(make([]byte, 0))

	for {
		// receive frame
		frame := stream.Recv()

		switch frametype := frame.(type) {
		case *HeadersFrame:
			// if HEADERS Frame
			headersFrame := frametype
			resHeader = headersFrame.Headers

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
		if looplimit > 30 {
			Error("over run (loop limit = %v)", looplimit)
			break
		}
		looplimit++
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

func (stream *Stream) WindowUpdate(size uint16) {
	threshold := DEFAULT_WINDOW_SIZE / 2
	s := uint32(size)
	stream.WindowSize -= s
	if stream.WindowSize < threshold {
		frame := NewWindowUpdateFrame(threshold, stream.Id)
		stream.Send(frame) // err
		stream.WindowSize += threshold
	}
	stream.Conn.WindowSize -= s
	if stream.Conn.WindowSize < threshold {
		stream.Conn.SendWindowUpdate(threshold)
		stream.Conn.WindowSize += threshold
	}
}
