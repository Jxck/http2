package http2

import (
	. "github.com/jxck/http2/frame"
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

type Stream struct {
	Id         uint32
	State      State
	WindowSize uint32
	req        *http.Request
	ReadChan   chan Frame
	WriteChan  chan Frame
}

func NewStream(id uint32, writeChan chan Frame, windowSize uint32) *Stream {
	stream := &Stream{
		Id:         id,
		State:      IDLE,
		WindowSize: windowSize,
		ReadChan:   make(chan Frame),
		WriteChan:  writeChan,
	}
	go stream.ReadLoop()
	return stream
}

func (stream *Stream) ReadLoop() {
	log.Printf("start stream.ReadLoop() (id=%d)", stream.Id)
	for frame := range stream.ReadChan {
		log.Printf("stream %v recv %v\n", stream.Id, frame.Header().Type)
		switch frametype := frame.(type) {
		case *SettingsFrame:

			// if SETTINGS Frame
			settingsFrame := frametype
			if settingsFrame.Flags == 0 {
				// Apply Settings

				// send ACK
				ack := NewSettingsFrame(1 /*flag*/, nil /*setting*/, 0 /*streamid*/)
				stream.WriteChan <- ack
			} else if settingsFrame.Flags == 1 {
				// receive ACK
				// TODO: Apply Settings
			}
		case *HeadersFrame:
			log.Println(frametype)
		}
	}
}

func (stream *Stream) Write(frame Frame) {
	stream.WriteChan <- frame
}
