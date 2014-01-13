package http2

import (
	"bufio"
	. "github.com/jxck/color"
	"github.com/jxck/hpack"
	. "github.com/jxck/logger"
	"io"
	"log"
	"net/http"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

// Conn has 2 funcs
// ReadFrame() frame
// WriteFrame(frame)
type Conn struct {
	RW              io.ReadWriter
	Bw              *bufio.Writer
	Br              *bufio.Reader
	RequestContext  *hpack.Context
	ResponseContext *hpack.Context
	LastStreamId    uint32
	WindowSize      uint32
}

func NewConn(rw io.ReadWriter) *Conn {
	conn := &Conn{
		RW:              rw,
		Bw:              bufio.NewWriter(rw),
		Br:              bufio.NewReader(rw),
		RequestContext:  hpack.NewContext(hpack.REQUEST, hpack.DEFAULT_HEADER_TABLE_SIZE),
		ResponseContext: hpack.NewContext(hpack.RESPONSE, hpack.DEFAULT_HEADER_TABLE_SIZE),
		WindowSize:      DEFAULT_WINDOW_SIZE,
	}
	return conn
}

func (c *Conn) NextStreamId() uint32 {
	id := c.LastStreamId
	if id == 4294967295 { // 2^32-1
		// err
	}
	if id == 0 {
		id = 1
	} else if id > 0 {
		id += 2
	}
	if id%2 == 0 {
		id += 1
	}
	c.LastStreamId = id
	return id
}

func (c *Conn) NewStream() *Stream {
	stream := &Stream{
		Id:         c.NextStreamId(),
		Conn:       c,
		WindowSize: DEFAULT_WINDOW_SIZE,
	}
	return stream
}

func (c *Conn) ReadFrame() (frame Frame) {
	fh := new(FrameHeader)
	fh.Read(c.RW) // err

	switch fh.Type {
	case DataFrameType:
		frame = &DataFrame{
			FrameHeader: fh,
		}
		frame.Read(c.RW)
	case HeadersFrameType:
		f := &HeadersFrame{
			FrameHeader: fh,
		}
		f.Read(c.RW)

		f.Headers = c.DecodeHeader(f.HeaderBlock)
		frame = f
	case RstStreamFrameType:
		frame = &RstStreamFrame{
			FrameHeader: fh,
		}
		frame.Read(c.RW)
	case SettingsFrameType:
		frame = &SettingsFrame{
			FrameHeader: fh,
		}
		frame.Read(c.RW)
	case GoAwayFrameType:
		frame = &GoAwayFrame{
			FrameHeader: fh,
		}
		frame.Read(c.RW)
	case WindowUpdateFrameType:
		frame = &WindowUpdateFrame{
			FrameHeader: fh,
		}
		frame.Read(c.RW)
	default:
		Error("unknown type: %v", fh.Type)
		return nil // err
	}
	Info("%v %v", Green("recv"), Indent(frame.Format()))
	return frame
}

func (c *Conn) WriteFrame(frame Frame) { // err
	// DEGBU
	// buf := bytes.NewBuffer([]byte{})
	// frame.Write(buf)
	// log.Println(buf.Bytes())
	frame.Write(c.RW) // err
	Info("%v %v", Red("send"), Indent(frame.Format()))
}

func (c *Conn) SendSettings(settings map[SettingsId]uint32) { // err
	c.WriteFrame(NewSettingsFrame(0, settings, 0)) // err
}

func (c *Conn) SendSettingsAck() { // err
	c.WriteFrame(NewSettingsFrame(1, nil, 0)) // err
}

func (c *Conn) SendGoAway(errorCode ErrorCode) { // err
	c.WriteFrame(NewGoAwayFrame(c.LastStreamId, errorCode, 0)) // err
}

func (c *Conn) SendWindowUpdate(incrementSize uint32) { // err
	c.WriteFrame(NewWindowUpdateFrame(incrementSize, 0)) // err
}

func (c *Conn) WriteString(str string) { // err
	c.Bw.WriteString(str) // err
	c.Bw.Flush()          // err
	Info("%v %q", Red("send"), str)
}

func (c *Conn) ReadString() { // err
	line, _, _ := c.Br.ReadLine() // err
	Debug(Red("recv"), Blue(string(line)))
}

func (c *Conn) ReadResponse() *http.Response {
	res, _ := http.ReadResponse(c.Br, &http.Request{Method: "GET"}) // err
	Debug(Green("recv"), Blue(ResponseString(res)), "\n")
	return res
}

func (c *Conn) ReadRequest() *http.Request {
	req, _ := http.ReadRequest(c.Br) // err
	Debug(Green("recv"), Blue(RequestString(req)), "\n")
	return req
}

// Encode Header using HPACK
func (c *Conn) EncodeHeader(header http.Header) []byte {
	headerSet := hpack.ToHeaderSet(header)
	return c.RequestContext.Encode(headerSet)
}

// Decode Header using HPACK
func (c *Conn) DecodeHeader(headerBlock []byte) http.Header {
	c.ResponseContext.Decode(headerBlock)
	return c.ResponseContext.ES.ToHeader()
}
