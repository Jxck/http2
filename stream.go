package http2

import (
	"github.com/jxck/hpack"
	. "github.com/jxck/http2/frame"
	. "github.com/jxck/logger"
	"log"
	"net/http"
	neturl "net/url"
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

func (s State) String() string {
	states := []string{
		"IDLE State",
		"RESERVED_LOCAL",
		"RESERVED_REMOTE",
		"OPEN",
		"HALF_CLOSED_LOCAL",
		"HALF_CLOSED_REMOTE",
		"CLOSED",
	}
	return states[int(s)]
}

type Stream struct {
	Id           uint32
	State        State
	WindowSize   uint32
	ReadChan     chan Frame
	WriteChan    chan Frame
	HpackContext *hpack.Context
	// fix me below
	Handler http.Handler
	Bucket  *Bucket
}

type Bucket struct {
	Headers []*HeadersFrame
	Body    []*DataFrame
}

func NewBucket() *Bucket {
	return &Bucket{
		make([]*HeadersFrame, 0),
		make([]*DataFrame, 0),
	}
}

func NewStream(id uint32, writeChan chan Frame, windowSize uint32, hpackContext *hpack.Context, handler http.Handler) *Stream {
	stream := &Stream{
		Id:           id,
		State:        IDLE,
		WindowSize:   windowSize,
		ReadChan:     make(chan Frame),
		WriteChan:    writeChan,
		HpackContext: hpackContext,
		// fix me below
		Handler: handler,
		Bucket:  NewBucket(),
	}
	go stream.ReadLoop()

	return stream
}

func (stream *Stream) ChangeState(state State) {
	Debug("stream (%d) state (%s)", stream.Id, state)
	stream.State = state
}

func (stream *Stream) ReadLoop() {
	Debug("start stream (%d) ReadLoop()", stream.Id)
	for f := range stream.ReadChan {
		Debug("stream (%d) recv (%v)", stream.Id, f.Header().Type)
		switch frame := f.(type) {
		case *SettingsFrame:

			// if SETTINGS Frame
			settingsFrame := frame
			if settingsFrame.Flags == 0 {
				// Apply Settings

				// send ACK
				ack := NewSettingsFrame(1 /*flag*/, nil /*setting*/, stream.Id /*streamid*/)
				stream.Write(ack)
			} else if settingsFrame.Flags == 1 {
				// receive ACK
				log.Println("receive SETTINGS ACK")
				// TODO: Apply Settings
			}
		case *HeadersFrame:
			stream.ChangeState(OPEN)

			stream.Bucket.Headers = append(stream.Bucket.Headers, frame)
			if frame.Flags&END_STREAM == END_STREAM {
				stream.ChangeState(HALF_CLOSED_REMOTE)
			}

			header := util.RemovePrefix(stream.DecodeHeader(frame.HeaderBlock))
			frame.Headers = header

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
			stream.Handler.ServeHTTP(res, req)
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

		case *DataFrame:
			log.Println(string(frame.Data))
		case *GoAwayFrame:
			log.Println("GOAWAY")
		}
	}
}

func (stream *Stream) Write(frame Frame) {
	stream.WriteChan <- frame
}

// Encode Header using HPACK
func (stream *Stream) EncodeHeader(header http.Header) []byte {
	headerSet := hpack.ToHeaderSet(header)
	return stream.HpackContext.Encode(headerSet)
}

// Decode Header using HPACK
func (stream *Stream) DecodeHeader(headerBlock []byte) http.Header {
	stream.HpackContext.Decode(headerBlock)
	return stream.HpackContext.ES.ToHeader()
}
