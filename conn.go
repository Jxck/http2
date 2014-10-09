package http2

import (
	"fmt"
	. "github.com/Jxck/color"
	"github.com/Jxck/hpack"
	. "github.com/Jxck/http2/frame"
	. "github.com/Jxck/logger"
	"io"
	"log"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

type Conn struct {
	RW           io.ReadWriter
	HpackContext *hpack.Context
	LastStreamId uint32
	WindowSize   uint32
	Streams      map[uint32]*Stream
	WriteChan    chan Frame
	CallBack     func(stream *Stream)
}

func NewConn(rw io.ReadWriter) *Conn {
	conn := &Conn{
		RW:           rw,
		HpackContext: hpack.NewContext(hpack.DEFAULT_HEADER_TABLE_SIZE),
		WindowSize:   DEFAULT_WINDOW_SIZE,
		Streams:      make(map[uint32]*Stream),
		WriteChan:    make(chan Frame),
	}
	return conn
}

func (conn *Conn) NewStream(streamid uint32) *Stream {
	conn.LastStreamId = streamid // TODO: fixme
	stream := NewStream(
		streamid,
		conn.WriteChan,
		DEFAULT_WINDOW_SIZE,
		conn.HpackContext,
		conn.CallBack,
	)
	conn.Streams[stream.Id] = stream
	Debug("adding new stream (id=%d) total (%d)", stream.Id, len(conn.Streams))
	return stream
}

func (conn *Conn) ReadFrame() (frame Frame, err error) {
	frame, err = ReadFrame(conn.RW)
	if frame != nil {
		Notice("%v %v", Green("recv"), util.Indent(frame.String()))
	}
	return frame, err
}

func (conn *Conn) ReadLoop() {
	Debug("start conn.ReadLoop()")
	for {
		frame, err := conn.ReadFrame()
		if err != nil {
			if err == io.EOF {
				log.Println(err)
				break
			}
		}
		streamId := frame.Header().StreamId
		stream, ok := conn.Streams[streamId]
		if !ok {
			// create stream with streamId
			stream = conn.NewStream(streamId)
			conn.Streams[streamId] = stream
		}
		stream.ReadChan <- frame
	}
}

func (conn *Conn) WriteFrame(frame Frame) (err error) {
	Notice("%v %v", Red("send"), util.Indent(frame.String()))
	return frame.Write(conn.RW)
}

func (conn *Conn) WriteLoop() (err error) {
	Debug("start conn.WriteLoop()")
	for frame := range conn.WriteChan {
		Debug("WriteLoop() %T", frame)
		err = conn.WriteFrame(frame)
		if err != nil {
			Error("%v", err)
			return err
		}
	}
	return
}

func (conn *Conn) WriteString(str string) (err error) {
	_, err = conn.RW.Write([]byte(str))
	if err != nil {
		return err
	}
	Info("%v %q", Red("send"), str)
	return
}

func (conn *Conn) WriteMagic() error {
	return conn.WriteString(CONNECTION_PREFACE)
}

func (conn *Conn) ReadMagic() (err error) {
	magic := make([]byte, len(CONNECTION_PREFACE))
	_, err = conn.RW.Read(magic)
	if err != nil {
		return err
	}
	if string(magic) != CONNECTION_PREFACE {
		Error("Invalid Magic String")
		return fmt.Errorf("Invalid Magic String")
	}
	Info("%v %q", Red("recv"), string(magic))
	return
}
