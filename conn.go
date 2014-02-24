package http2

import (
	"bufio"
	"fmt"
	. "github.com/jxck/color"
	"github.com/jxck/hpack"
	. "github.com/jxck/http2/frame"
	. "github.com/jxck/logger"
	"io"
	"log"
)

const (
	Version               = "h2-10"
	MagicString           = "PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n"
	DefaultSettingsBase64 = "AAAABAAAAGQAAAAHAAD__w"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

type Conn struct {
	RW           io.ReadWriter
	Bw           *bufio.Writer
	Br           *bufio.Reader
	HpackContext *hpack.Context
	LastStreamId uint32
	WindowSize   uint32
	Streams      map[uint32]*Stream
	WriteChan    chan Frame
	// TODO: fix me below
	CallBack func(stream *Stream)
}

func NewConn(rw io.ReadWriter) *Conn {
	conn := &Conn{
		RW:           rw,
		Bw:           bufio.NewWriter(rw),
		Br:           bufio.NewReader(rw),
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
		conn.CallBack)
	conn.Streams[stream.Id] = stream
	Debug("adding new stream (id=%d) total (%d)", stream.Id, len(conn.Streams))
	return stream
}

func (conn *Conn) ReadFrame() (frame Frame, err error) {
	fh := new(FrameHeader)
	err = fh.Read(conn.RW) // err
	if err != nil {
		return nil, err
	}

	newframe, ok := FrameMap[fh.Type]
	if !ok {
		return nil, fmt.Errorf("unknown type: %v", fh.Type)
	}

	frame = newframe(fh)
	frame.Read(conn.RW)
	Notice("%v %v", Green("recv"), util.Indent(frame.Format()))
	return frame, nil
}

func (conn *Conn) ReadLoop() {
	Debug("start conn.ReadLoop()")
	for {
		frame, err := conn.ReadFrame()
		if err != nil {
			log.Fatal(err)
			//break
		}
		streamId := frame.Header().StreamId
		stream, ok := conn.Streams[streamId]
		if !ok {
			// create stream id = streamId
			stream = conn.NewStream(streamId)
		}
		stream.ReadChan <- frame
	}
}

func (conn *Conn) WriteFrame(frame Frame) { // err
	Notice("%v %v", Red("send"), util.Indent(frame.Format()))
	frame.Write(conn.RW) // err
}

func (conn *Conn) WriteLoop() { // err
	Debug("start conn.WriteLoop()")
	for frame := range conn.WriteChan {
		Debug("WriteLoop() %T", frame)
		conn.WriteFrame(frame)
	}
}

func (conn *Conn) WriteString(str string) { // err
	conn.Bw.WriteString(str) // err
	conn.Bw.Flush()          // err
	Info("%v %q", Red("send"), str)
}

func (conn *Conn) WriteMagic() {
	conn.WriteString(MagicString)
}

func (conn *Conn) ReadMagic() { // err
	magic := make([]byte, len(MagicString))
	conn.RW.Read(magic) // err
	if string(magic) != MagicString {
		Error("Invalid Magic String") // err
	}
	Info("%v %q", Red("recv"), string(magic))
}

// map of FrameType and FrameInitializer
var FrameMap = map[uint8](func(*FrameHeader) Frame){
	DataFrameType:         func(fh *FrameHeader) Frame { return &DataFrame{FrameHeader: fh} },
	HeadersFrameType:      func(fh *FrameHeader) Frame { return &HeadersFrame{FrameHeader: fh} },
	RstStreamFrameType:    func(fh *FrameHeader) Frame { return &RstStreamFrame{FrameHeader: fh} },
	SettingsFrameType:     func(fh *FrameHeader) Frame { return &SettingsFrame{FrameHeader: fh} },
	GoAwayFrameType:       func(fh *FrameHeader) Frame { return &GoAwayFrame{FrameHeader: fh} },
	WindowUpdateFrameType: func(fh *FrameHeader) Frame { return &WindowUpdateFrame{FrameHeader: fh} },
}
