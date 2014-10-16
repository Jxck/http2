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
func (stream *Stream) ChangeState(frame Frame, context Context) (err error) {

	header := frame.Header()
	types := header.Type
	flags := header.Flags
	state := stream.State

	if types == SettingsFrameType ||
		types == GoAwayFrameType {
		// not a type for consider
		return nil
	}

	if types != PushPromiseFrameType &&
		types != HeadersFrameType &&
		types != RstStreamFrameType &&
		flags != END_STREAM {
		// not a type/flag for consider
		return nil
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
	Info("change stream (%d) state (%s -> %s)", stream.ID, stream.State, Pink(state.String()))
	stream.State = state
}
