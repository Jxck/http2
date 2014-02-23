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
	"net/http"
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
	Handler http.Handler
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

	// TODO: Server dosen't send MagicOctet
	// conn.WriteMagic()
	conn.ReadMagic()

	go conn.WriteLoop()

	// stream id 0
	zeroStream := conn.NewStream(0)
	conn.Streams[0] = zeroStream

	// send default settings to id 0
	settingsFrame := NewSettingsFrame(UNSET, DefaultSettings, 0)
	zeroStream.Write(settingsFrame)

	return conn
}

func (c *Conn) NewStream(streamid uint32) *Stream {
	stream := NewStream(
		streamid,
		c.WriteChan,
		DEFAULT_WINDOW_SIZE,
		c.HpackContext,
		c.Handler)
	c.Streams[stream.Id] = stream
	Debug("adding new stream (id=%d) total (%d)", stream.Id, len(c.Streams))
	return stream
}

func (c *Conn) ReadFrame() (frame Frame, err error) {
	fh := new(FrameHeader)
	err = fh.Read(c.RW) // err
	if err != nil {
		return nil, err
	}

	newframe, ok := FrameMap[fh.Type]
	if !ok {
		return nil, fmt.Errorf("unknown type: %v", fh.Type)
	}

	frame = newframe(fh)
	frame.Read(c.RW)
	Notice("%v %v", Green("recv"), util.Indent(frame.Format()))
	return frame, nil
}

func (c *Conn) ReadLoop() {
	Debug("start conn.ReadLoop()")
	for {
		frame, err := c.ReadFrame()
		if err != nil {
			log.Fatal(err)
			//break
		}
		streamId := frame.Header().StreamId
		stream, ok := c.Streams[streamId]
		if !ok {
			// create stream id = streamId
			stream = c.NewStream(streamId)
		}
		stream.ReadChan <- frame
	}
}

func (c *Conn) WriteFrame(frame Frame) { // err
	Notice("%v %v", Red("send"), util.Indent(frame.Format()))
	frame.Write(c.RW) // err
}

func (c *Conn) WriteLoop() { // err
	Debug("start conn.WriteLoop()")
	for frame := range c.WriteChan {
		Debug("WriteLoop() %T", frame)
		c.WriteFrame(frame)
	}
}

func (c *Conn) WriteString(str string) { // err
	c.Bw.WriteString(str) // err
	c.Bw.Flush()          // err
	Info("%v\n%s", Red("send"), Blue(str))
}

func (c *Conn) WriteMagic() {
	c.WriteString(MagicString)
}

func (c *Conn) ReadMagic() { // err
	magic := make([]byte, len(MagicString))
	c.RW.Read(magic) // err
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
