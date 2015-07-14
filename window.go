package http2

import (
	"fmt"
	. "github.com/Jxck/color"
	. "github.com/Jxck/http2/frame"
	. "github.com/Jxck/logger"
	"log"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

type Window struct {
	initialSize     int32
	currentSize     int32
	threshold       int32
	peerInitialSize int32
	peerCurrentSize int32
	peerThreshold   int32
}

func NewWindowDefault() *Window {
	return &Window{
		initialSize:     DEFAULT_INITIAL_WINDOW_SIZE,
		currentSize:     DEFAULT_INITIAL_WINDOW_SIZE,
		threshold:       DEFAULT_INITIAL_WINDOW_SIZE/2 + 1,
		peerInitialSize: DEFAULT_INITIAL_WINDOW_SIZE,
		peerCurrentSize: DEFAULT_INITIAL_WINDOW_SIZE,
		peerThreshold:   DEFAULT_INITIAL_WINDOW_SIZE/2 + 1,
	}
}

func NewWindow(initialWindow, peerInitilaWindow int32) *Window {
	return &Window{
		initialSize:     initialWindow,
		currentSize:     initialWindow,
		threshold:       initialWindow/2 + 1,
		peerInitialSize: peerInitilaWindow,
		peerCurrentSize: peerInitilaWindow,
		peerThreshold:   peerInitilaWindow/2 + 1,
	}
}

func (window *Window) UpdateInitialSize(initialWindowSize int32) {
	window.peerCurrentSize -= window.initialSize
	window.peerCurrentSize += initialWindowSize
	window.initialSize = initialWindowSize
}

func (window *Window) Update(windowSizeIncrement int32) {
	window.currentSize += windowSizeIncrement
}

func (window *Window) UpdatePeer(windowSizeIncrement int32) {
	window.peerCurrentSize += windowSizeIncrement
}

func (window *Window) Consume(length int32) (update int32) {
	window.currentSize -= length

	if window.currentSize < window.threshold {
		update = window.initialSize - window.currentSize
	}

	return update
}

func (window *Window) ConsumePeer(length int32) {
	Trace("consume peer window current(%v), use(%v), rest(%v)", window.peerCurrentSize, length, window.peerCurrentSize-length)
	window.peerCurrentSize -= length
}

func (window *Window) Consumable(length int32) int32 {
	if window.peerCurrentSize < length {
		return window.peerCurrentSize
	} else {
		return length
	}
}

func (window *Window) String() string {
	return fmt.Sprintf(Yellow("window: curr(%d) - peer(%d)"), window.currentSize, window.peerCurrentSize)
}
