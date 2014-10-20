package http2

import (
	"github.com/Jxck/hpack"
	. "github.com/Jxck/http2/frame"
	. "github.com/Jxck/logger"
	"log"
	"net/http"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

type Stream struct {
	ID           uint32
	State        State
	WindowSize   uint32
	ReadChan     chan Frame
	WriteChan    chan Frame
	HpackContext *hpack.Context
	CallBack     CallBack
	Bucket       *Bucket
}

type Bucket struct {
	Headers []*HeadersFrame
	Data    []*DataFrame
}

func NewBucket() *Bucket {
	return &Bucket{
		Headers: make([]*HeadersFrame, 0),
		Data:    make([]*DataFrame, 0),
	}
}

type CallBack func(stream *Stream)

func NewStream(id uint32, writeChan chan Frame, windowSize uint32, hpackContext *hpack.Context, callback CallBack) *Stream {
	stream := &Stream{
		ID:           id,
		State:        IDLE,
		WindowSize:   windowSize,
		ReadChan:     make(chan Frame),
		WriteChan:    writeChan,
		HpackContext: hpackContext,
		CallBack:     callback,
		Bucket:       NewBucket(),
	}
	go stream.ReadLoop()
	return stream
}

func (stream *Stream) Read(f Frame) {
	Debug("stream (%d) recv (%v)", stream.ID, f.Header().Type)
	stream.WindowUpdate(f.Header().Length)

	switch frame := f.(type) {
	case *SettingsFrame:

		// if SETTINGS Frame
		settingsFrame := frame
		if settingsFrame.Flags == UNSET {
			// TODO: Apply Settings

			// send ACK
			ack := NewSettingsFrame(ACK, stream.ID, NilSettings)
			stream.Write(ack)
		} else if settingsFrame.Flags == ACK {
			// receive ACK
			Trace("receive SETTINGS ACK")
		}
	case *HeadersFrame:
		// Decode Headers
		header := stream.DecodeHeader(frame.HeaderBlock)
		frame.Headers = header

		stream.Bucket.Headers = append(stream.Bucket.Headers, frame)

		if frame.Header().Flags&END_STREAM == END_STREAM {
			stream.CallBack(stream)
		}
	case *DataFrame:
		stream.Bucket.Data = append(stream.Bucket.Data, frame)

		if frame.Header().Flags&END_STREAM == END_STREAM {
			stream.CallBack(stream)
		}
	case *RstStreamFrame:
		Debug("close stream by RST_STREAM")
		Error("RST_STREAM(%v)", frame.ErrorCode)
		stream.Close()
	case *PingFrame:
		Debug("response to PING")
		ping := NewPingFrame(ACK, stream.ID, frame.OpaqueData)
		stream.Write(ping)
	}
}

func (stream *Stream) ReadLoop() {
	Debug("start stream (%d) ReadLoop()", stream.ID)
	for f := range stream.ReadChan {
		stream.Read(f)
	}
	Debug("stop stream (%d) ReadLoop()", stream.ID)
}

func (stream *Stream) Write(frame Frame) {
	stream.ChangeState(frame, SEND)
	stream.WriteChan <- frame
}

func (stream *Stream) WindowUpdate(length uint32) {
	total := stream.WindowSize

	total = total - length
	if total < WINDOW_UPDATE_THRESHOLD {
		// この値を下回ったら WindowUpdate を送る
		update := stream.WindowSize - total
		stream.Write(NewWindowUpdateFrame(stream.ID, update))
		stream.Write(NewWindowUpdateFrame(0, update))
	} else {
		stream.WindowSize = total
	}
}

func (stream *Stream) Close() {
	Debug("stream(%d) Close()", stream.ID)
	// write chan は close しない
	close(stream.ReadChan)
}

// Encode Header using HPACK
func (stream *Stream) EncodeHeader(header http.Header) []byte {
	headerList := hpack.ToHeaderList(header)
	Trace("sending header list %s", headerList)
	return stream.HpackContext.Encode(*headerList)
}

// Decode Header using HPACK
func (stream *Stream) DecodeHeader(headerBlock []byte) http.Header {
	stream.HpackContext.Decode(headerBlock)
	return stream.HpackContext.ES.ToHeader()
}
