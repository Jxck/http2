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

func NewWindowDefault() *Window {
	return &Window{
		InitialSize:     DEFAULT_INITIAL_WINDOW_SIZE,
		CurrentSize:     DEFAULT_INITIAL_WINDOW_SIZE,
		Threshold:       DEFAULT_INITIAL_WINDOW_SIZE/2 + 1,
		PeerInitialSize: DEFAULT_INITIAL_WINDOW_SIZE,
		PeerCurrentSize: DEFAULT_INITIAL_WINDOW_SIZE,
		PeerThreshold:   DEFAULT_INITIAL_WINDOW_SIZE/2 + 1,
	}
}

func NewWindow(settings, peerSettings map[SettingsID]int32) *Window {
	initialWindow := settings[SETTINGS_INITIAL_WINDOW_SIZE]
	peerInitilaWindow := peerSettings[SETTINGS_INITIAL_WINDOW_SIZE]
	return &Window{
		InitialSize:     initialWindow,
		CurrentSize:     initialWindow,
		Threshold:       initialWindow/2 + 1,
		PeerInitialSize: peerInitilaWindow,
		PeerCurrentSize: peerInitilaWindow,
		PeerThreshold:   peerInitilaWindow/2 + 1,
	}
}
