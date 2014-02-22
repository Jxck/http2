package http2

import (
	"github.com/jxck/hpack"
	. "github.com/jxck/http2/frame"
	"log"
	"net/http"
	neturl "net/url"
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
	Id           uint32
	State        State
	WindowSize   uint32
	req          *http.Request
	ReadChan     chan Frame
	WriteChan    chan Frame
	HpackContext *hpack.Context
}

func NewStream(id uint32, writeChan chan Frame, windowSize uint32, hpackContext *hpack.Context) *Stream {
	stream := &Stream{
		Id:           id,
		State:        IDLE,
		WindowSize:   windowSize,
		ReadChan:     make(chan Frame),
		WriteChan:    writeChan,
		HpackContext: hpackContext,
	}
	go stream.ReadLoop()
	return stream
}

func (stream *Stream) ReadLoop() {
	log.Printf("start stream.ReadLoop() (id=%d)", stream.Id)
	for f := range stream.ReadChan {
		log.Printf("stream %v recv %v\n", stream.Id, f.Header().Type)
		switch frame := f.(type) {
		case *SettingsFrame:

			// if SETTINGS Frame
			settingsFrame := frame
			if settingsFrame.Flags == 0 {
				// Apply Settings

				// send ACK
				ack := NewSettingsFrame(1 /*flag*/, nil /*setting*/, stream.Id /*streamid*/)
				stream.WriteChan <- ack
			} else if settingsFrame.Flags == 1 {
				// receive ACK
				// TODO: Apply Settings
			}
		case *HeadersFrame:
			header := util.RemovePrefix(stream.DecodeHeader(frame.HeaderBlock))
			frame.Headers = header
			for k, v := range header {
				log.Println(k, v)
			}

			url := &neturl.URL{
				Scheme: header.Get("scheme"),
				Host:   header.Get("authority"),
				Path:   header.Get("path"),
			}

			req := &http.Request{
				Method:     header.Get("method"),
				URL:        url,
				Proto:      "HTTP/1.1",
				ProtoMajor: 1,
				ProtoMinor: 1,
				//Header:        header,
				Body:          nil,
				ContentLength: 0,
				// TransferEncoding []string
				Close: false,
				Host:  header.Get("Authority"),
			}

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
