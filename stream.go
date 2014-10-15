package http2

import (
	"fmt"
	. "github.com/Jxck/color"
	"github.com/Jxck/hpack"
	. "github.com/Jxck/http2/frame"
	. "github.com/Jxck/logger"
	"log"
	"net/http"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

// state of stream
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
		"IDLE",
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
	WindowUpdate chan uint32
	ReadChan     chan Frame
	WriteChan    chan Frame
	HpackContext *hpack.Context
	CallBack     CallBack
	Bucket       *Bucket
	breakloop    chan bool
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
		Id:           id,
		State:        IDLE,
		WindowSize:   windowSize,
		WindowUpdate: make(chan uint32),
		ReadChan:     make(chan Frame),
		WriteChan:    writeChan,
		HpackContext: hpackContext,
		CallBack:     callback,
		Bucket:       NewBucket(),
		breakloop:    make(chan bool),
	}
	go stream.ReadLoop()
	go stream.WindowUpdateLoop()
	return stream
}

const (
	RECV = true
	SEND = false
)

//  Stream States
//                        +--------+
//                  PP    |        |    PP
//               ,--------|  idle  |--------.
//              /         |        |         \
//             v          +--------+          v
//      +----------+          |           +----------+
//      |          |          | H         |          |
//  ,---| reserved |          |           | reserved |---.
//  |   | (local)  |          v           | (remote) |   |
//  |   +----------+      +--------+      +----------+   |
//  |      |          ES  |        |  ES          |      |
//  |      | H    ,-------|  open  |-------.      | H    |
//  |      |     /        |        |        \     |      |
//  |      v    v         +--------+         v    v      |
//  |   +----------+          |           +----------+   |
//  |   |   half   |          |           |   half   |   |
//  |   |  closed  |          | R         |  closed  |   |
//  |   | (remote) |          |           | (local)  |   |
//  |   +----------+          |           +----------+   |
//  |        |                v                 |        |
//  |        |  ES / R    +--------+  ES / R    |        |
//  |        `----------->|        |<-----------'        |
//  |  R                  | closed |                  R  |
//  `-------------------->|        |<--------------------'
//                        +--------+
//  H:  HEADERS frame (with implied CONTINUATIONs)
//  PP: PUSH_PROMISE frame (with implied CONTINUATIONs)
//  ES: END_STREAM flag
//  R:  RST_STREAM frame
func (stream *Stream) ChangeState(frame Frame, context bool) (err error) {

	header := frame.Header()
	flags := header.Flags
	types := header.Type
	state := stream.State

	if types == SettingsFrameType {
		return
	}

	switch stream.State {
	case IDLE:
		// H
		if types == HeadersFrameType {
			stream.changeState(OPEN)
			return
		}

		// PP
		if types == PushPromiseFrameType {
			if context == RECV {
				stream.changeState(RESERVED_REMOTE)
			} else {
				stream.changeState(RESERVED_LOCAL)
			}
			return
		}
	case OPEN:
		// ES
		if flags == END_STREAM {
			if context == RECV {
				stream.changeState(HALF_CLOSED_REMOTE)
			} else {
				stream.changeState(HALF_CLOSED_LOCAL)
			}
			return
		}

		// R
		if types == RstStreamFrameType {
			stream.changeState(CLOSED)
			return
		}
	case RESERVED_LOCAL:
		// H
		if types == HeadersFrameType {
			stream.changeState(HALF_CLOSED_REMOTE)
			return
		}

		// R
		if types == RstStreamFrameType {
			stream.changeState(CLOSED)
			return
		}
	case RESERVED_REMOTE:
		// H
		if types == HeadersFrameType {
			stream.changeState(HALF_CLOSED_LOCAL)
			return
		}

		// R
		if types == RstStreamFrameType {
			stream.changeState(CLOSED)
			return
		}
	case HALF_CLOSED_LOCAL:
		// same as half closed remote
		fallthrough
	case HALF_CLOSED_REMOTE:
		// ES
		if flags == END_STREAM {
			stream.changeState(CLOSED)
			return
		}

		// R
		if types == RstStreamFrameType {
			stream.changeState(CLOSED)
			return
		}
	}

	return fmt.Errorf("invalid frame type %v at %v state", types, state)
}

func (stream *Stream) changeState(state State) {
	Debug("change stream (%d) state (%s -> %s)", stream.Id, stream.State, Pink(state.String()))
	stream.State = state
}

func (stream *Stream) ReadLoop() {
	Debug("start stream (%d) ReadLoop()", stream.Id)
BreakLoop:
	for {
		select {
		case <-stream.breakloop:
			Debug("stop stream (%d) ReadLoop()", stream.Id)
			break BreakLoop
		case f := <-stream.ReadChan:
			Debug("stream (%d) recv (%v)", stream.Id, f.Header().Type)
			stream.WindowUpdate <- uint32(f.Header().Length)

			stream.ChangeState(f, RECV)

			switch frame := f.(type) {
			case *SettingsFrame:

				// if SETTINGS Frame
				settingsFrame := frame
				if settingsFrame.Flags == UNSET {
					// TODO: Apply Settings

					// send ACK
					ack := NewSettingsFrame(ACK, stream.Id, NilSettings)
					stream.Write(ack)
				} else if settingsFrame.Flags == ACK {
					// receive ACK
					Trace("receive SETTINGS ACK")
				}
			case *HeadersFrame:
				// Decode Headers
				header := util.RemovePrefix(stream.DecodeHeader(frame.HeaderBlock))
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
				ping := NewPingFrame(ACK, stream.Id, frame.OpaqueData)
				stream.Write(ping)
			case *GoAwayFrame:
				Debug("close stream by GOAWAY")
				stream.Close()
			}
		}
	}
}

func (stream *Stream) Write(frame Frame) {
	stream.ChangeState(frame, SEND)
	stream.WriteChan <- frame
}

func (stream *Stream) WindowUpdateLoop() {
	total := stream.WindowSize

BreakLoop:
	for {
		select {
		case <-stream.breakloop:
			Debug("stom stream (%d) ReadLoop()", stream.Id)
			break BreakLoop
		case size := <-stream.WindowUpdate:
			total = total - size
			if total < WINDOW_UPDATE_THRESHOLD {
				// この値を下回ったら WindowUpdate を送る
				update := stream.WindowSize - total
				stream.Write(NewWindowUpdateFrame(update, stream.Id))
				stream.Write(NewWindowUpdateFrame(update, 0))
			}
		}
	}
}

func (stream *Stream) Close() {
	close(stream.WindowUpdate)
	close(stream.breakloop)
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
