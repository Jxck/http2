package http2

import (
	. "github.com/Jxck/http2/frame"
	"log"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

type Window struct {
	InitialSize     int32
	CurrentSize     int32
	Threshold       int32
	PeerInitialSize int32
	PeerCurrentSize int32
	PeerThreshold   int32
}

func NewWindow() *Window {
	return &Window{
		InitialSize:     DEFAULT_INITIAL_WINDOW_SIZE,
		CurrentSize:     DEFAULT_INITIAL_WINDOW_SIZE,
		Threshold:       DEFAULT_INITIAL_WINDOW_SIZE/2 + 1,
		PeerInitialSize: DEFAULT_INITIAL_WINDOW_SIZE,
		PeerCurrentSize: DEFAULT_INITIAL_WINDOW_SIZE,
		PeerThreshold:   DEFAULT_INITIAL_WINDOW_SIZE/2 + 1,
	}
}
