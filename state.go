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

type Context int

const (
	RECV Context = iota
	SEND
)

func (c Context) String() string {
	return []string{
		"RECV",
		"SEND",
	}[int(c)]
}

//  Stream States
//                           +--------+
//                   send PP |        | recv PP
//                  ,--------|  idle  |--------.
//                 /         |        |         \
//                v          +--------+          v
//         +----------+          |           +----------+
//         |          |          | send H /  |          |
//  ,------| reserved |          | recv H    | reserved |------.
//  |      | (local)  |          |           | (remote) |      |
//  |      +----------+          v           +----------+      |
//  |          |             +--------+             |          |
//  |          |     recv ES |        | send ES     |          |
//  |   send H |     ,-------|  open  |-------.     | recv H   |
//  |          |    /        |        |        \    |          |
//  |          v   v         +--------+         v   v          |
//  |      +----------+          |           +----------+      |
//  |      |   half   |          |           |   half   |      |
//  |      |  closed  |          | send R /  |  closed  |      |
//  |      | (remote) |          | recv R    | (local)  |      |
//  |      +----------+          |           +----------+      |
//  |           |                |                 |           |
//  |           | send ES /      |       recv ES / |           |
//  |           | send R /       v        send R / |           |
//  |           | recv R     +--------+   recv R   |           |
//  | send R /  `----------->|        |<-----------'  send R / |
//  | recv R                 | closed |               recv R   |
//  `----------------------->|        |<----------------------'
//                           +--------+
//
//     send:   endpoint sends this frame
//     recv:   endpoint receives this frame
//
//     H:  HEADERS frame (with implied CONTINUATIONs)
//     PP: PUSH_PROMISE frame (with implied CONTINUATIONs)
//     ES: END_STREAM flag
//     R:  RST_STREAM frame
func (stream *Stream) ChangeState(frame Frame, context Context) (err error) {

	header := frame.Header()
	types := header.Type
	flags := header.Flags
	state := stream.State

	Trace("change state(%v) with %v frame type(%v)", state, context, types)

	if types == SettingsFrameType ||
		types == GoAwayFrameType {
		// not a type for consider
		return nil
	}

	switch stream.State {
	case IDLE:
		// H
		if types == HeadersFrameType {
			stream.changeState(OPEN)

			// ES
			if flags&END_STREAM == END_STREAM {
				if context == RECV {
					stream.changeState(HALF_CLOSED_REMOTE)
				} else {
					stream.changeState(HALF_CLOSED_LOCAL)
				}
			}
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

		// P
		if types == PriorityFrameType {
			// accepted
			return
		}
	case OPEN:
		// ES
		if flags&END_STREAM == END_STREAM {
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

		// every type of frame accepted
		return
	case RESERVED_LOCAL:
		// H
		if types == HeadersFrameType && context == SEND {
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
		if types == HeadersFrameType && context == RECV {
			stream.changeState(HALF_CLOSED_LOCAL)
			return
		}

		// R
		if types == RstStreamFrameType {
			stream.changeState(CLOSED)
			return
		}
	case HALF_CLOSED_LOCAL:
		// ES
		if flags&END_STREAM == END_STREAM && context == RECV {
			stream.changeState(CLOSED)
			return
		}

		// R
		if types == RstStreamFrameType {
			stream.changeState(CLOSED)
			return
		}
	case HALF_CLOSED_REMOTE:

		if context == SEND {
			// ES
			if flags&END_STREAM == END_STREAM {
				stream.changeState(CLOSED)
				return
			}

			// R
			if types == RstStreamFrameType {
				stream.changeState(CLOSED)
				return
			}

			// send any type of frame are valid
			return
		}

		if context == RECV {
			if types == WindowUpdateFrameType ||
				types == PriorityFrameType {

				// valid frame
				return
			}

			// R
			if types == RstStreamFrameType {
				stream.changeState(CLOSED)
				return
			}
		}
	}

	msg := fmt.Sprintf("invalid frame type %v at %v state", types, state)
	Error(Red(msg))
	return &H2Error{PROTOCOL_ERROR, msg}
}

func (stream *Stream) changeState(state State) {
	Info("change stream (%d) state (%s -> %s)", stream.ID, stream.State, Pink(state.String()))
	stream.State = state
}
