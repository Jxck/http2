package http2

import (
	. "github.com/jxck/color"
	"github.com/jxck/hpack"
	. "github.com/jxck/http2/frame"
	. "github.com/jxck/logger"
	"log"
	"net/http"
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
	CallBack func(stream *Stream)
	Bucket   *Bucket
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

func NewStream(id uint32, writeChan chan Frame, windowSize uint32, hpackContext *hpack.Context, callback func(stream *Stream)) *Stream {
	stream := &Stream{
		Id:           id,
		State:        IDLE,
		WindowSize:   windowSize,
		ReadChan:     make(chan Frame),
		WriteChan:    writeChan,
		HpackContext: hpackContext,
		// fix me below
		CallBack: callback,
		Bucket:   NewBucket(),
	}
	go stream.ReadLoop()

	return stream
}

func (stream *Stream) ChangeState(state State) {
	Debug("stream (%d) state (%s)", stream.Id, Pink(state.String()))
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

			// Decode Headers
			header := util.RemovePrefix(stream.DecodeHeader(frame.HeaderBlock))
			frame.Headers = header

			stream.Bucket.Headers = append(stream.Bucket.Headers, frame)

			if frame.Flags&END_STREAM == END_STREAM {
				stream.ChangeState(HALF_CLOSED_REMOTE)
				stream.CallBack(stream)
			}
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
