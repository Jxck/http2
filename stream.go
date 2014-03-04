package http2

import (
	. "github.com/jxck/color"
	"github.com/jxck/hpack"
	. "github.com/jxck/http2/frame"
	. "github.com/jxck/logger"
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
		ReadChan:     make(chan Frame),
		WriteChan:    writeChan,
		HpackContext: hpackContext,
		CallBack:     callback,
		Bucket:       NewBucket(),
		breakloop:    make(chan bool),
	}
	go stream.ReadLoop()

	return stream
}

const (
	RECV = true
	SEND = false
)

func (stream *Stream) ChangeState(frame Frame, context bool) {

	header := frame.Header()
	flags := header.Flags
	types := header.Type

	switch { // TODO:Manage Stream State more carefuly
	case types == DataFrameType:
		if flags&END_STREAM == END_STREAM {
			// END_STREAM を送るとき
			switch {
			case stream.State == OPEN:
				// まだ REMOTE が CLOSE してなかったら
				stream.changeState(HALF_CLOSED_LOCAL)
			case stream.State == HALF_CLOSED_REMOTE:
				// すでに REMOTE が CLOSE してたら
				log.Println("close")
				stream.changeState(CLOSED)
			default:
				log.Printf("END_STREAM at %v", stream.State)
			}
		}
	case types == HeadersFrameType:
		switch {
		case stream.State == IDLE:
			stream.changeState(OPEN)
		case stream.State == RESERVED_REMOTE:
			stream.changeState(HALF_CLOSED_LOCAL)
		default:
			log.Printf("HEADERS at %v", stream.State)
		}
	case types == RstStreamFrameType:
		// RST_STREAM を受け取るとき
		switch {
		case stream.State == OPEN:
			log.Println("close")
			stream.changeState(CLOSED)
		case stream.State == RESERVED_REMOTE:
			log.Println("close")
			stream.changeState(CLOSED)
		case stream.State == HALF_CLOSED_LOCAL:
			log.Println("close")
			stream.changeState(CLOSED)
		default:
			log.Printf("RST at %v", stream.State)
		}
	case types == PushPrimiseFrameType:
		// PUSH_PROMISE を受け取るとき
		switch {
		case stream.State == IDLE:
			// 今後使用するために予約
			stream.changeState(RESERVED_REMOTE)
		default:
			log.Printf("PP at %v", stream.State)
		}
	case flags&END_STREAM == END_STREAM:
		// END_STREAM を受け取るとき
		switch {
		case stream.State == OPEN:
			// OPEN だったら
			stream.changeState(HALF_CLOSED_REMOTE)
		case stream.State == HALF_CLOSED_LOCAL:
			// 自分がもう HALF_CLOSE してたら
			stream.changeState(CLOSED)
		default:
			log.Printf("END_STREAM at %v", stream.State)
		}
	}

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
			Debug("stom stream (%d) ReadLoop()", stream.Id)
			break BreakLoop
		case f := <-stream.ReadChan:
			Debug("stream (%d) recv (%v)", stream.Id, f.Header().Type)

			stream.ChangeState(f, RECV)

			switch frame := f.(type) {
			case *SettingsFrame:

				// if SETTINGS Frame
				settingsFrame := frame
				if settingsFrame.Flags == UNSET {
					// TODO: Apply Settings

					// send ACK
					ack := NewSettingsFrame(ACK, nil /*setting*/, stream.Id)
					stream.Write(ack)
				} else if settingsFrame.Flags == ACK {
					// receive ACK
					log.Println("receive SETTINGS ACK")
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
			case *GoAwayFrame:
				log.Println("GOAWAY")
			}
		}
	}
}

func (stream *Stream) Write(frame Frame) {
	stream.ChangeState(frame, SEND)
	stream.WriteChan <- frame
}

func (stream *Stream) Close() {
	close(stream.breakloop)
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
