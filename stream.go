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
	Window       *Window
	ReadChan     chan Frame
	WriteChan    chan Frame
	Settings     map[SettingsID]uint32
	PeerSettings map[SettingsID]uint32
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

func NewStream(id uint32, writeChan chan Frame, settings, peerSettings map[SettingsID]uint32, hpackContext *hpack.Context, callback CallBack) *Stream {
	stream := &Stream{
		ID:           id,
		State:        IDLE,
		Window:       NewWindow(),
		ReadChan:     make(chan Frame),
		WriteChan:    writeChan,
		Settings:     settings,
		PeerSettings: peerSettings,
		HpackContext: hpackContext,
		CallBack:     callback,
		Bucket:       NewBucket(),
	}
	go stream.ReadLoop()
	return stream
}

func (stream *Stream) Read(f Frame) {
	Debug("stream (%d) recv (%v)", stream.ID, f.Header().Type)

	switch frame := f.(type) {
	case *HeadersFrame:
		// Decode Headers
		header := stream.DecodeHeader(frame.HeaderBlock)
		frame.Headers = header

		stream.Bucket.Headers = append(stream.Bucket.Headers, frame)

		if frame.Header().Flags&END_STREAM == END_STREAM {
			stream.CallBack(stream)
		}
	case *DataFrame:
		length := int32(frame.Header().Length)
		stream.WindowUpdate(length)
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

func (stream *Stream) WindowUpdate(length int32) {
	Debug("stream(%d) window update %d byte", stream.ID, length)

	stream.Window.CurrentSize = stream.Window.CurrentSize - length

	// この値を下回ったら WindowUpdate を送る
	if stream.Window.CurrentSize < stream.Window.Threshold {
		update := stream.Window.InitialSize - stream.Window.CurrentSize
		stream.Write(NewWindowUpdateFrame(stream.ID, uint32(update)))
		stream.Window.CurrentSize = stream.Window.CurrentSize + update
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
