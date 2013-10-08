package http2

import (
	"bufio"
	"fmt"
	. "github.com/jxck/color"
	"github.com/jxck/hpack"
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
}

func NewConn(rw io.ReadWriter) *Conn {
	conn := &Conn{
		RW:              rw,
		Bw:              bufio.NewWriter(rw),
		Br:              bufio.NewReader(rw),
		RequestContext:  hpack.NewRequestContext(),
		ResponseContext: hpack.NewResponseContext(),
	}
	return conn
}

func (c *Conn) ReadFrame() Frame {
	fh := &FrameHeader{} // New
	fh.Read(c.RW)        // err

	switch fh.Type {
	case DataFrameType:
		frame := &DataFrame{}
		frame.FrameHeader = fh
		frame.Read(c.RW)
		return frame
	case HeadersFrameType:
		frame := &HeadersFrame{}
		frame.FrameHeader = fh
		frame.Read(c.RW)

		frame.Headers = c.DecodeHeader(frame.HeaderBlock)
		return frame
	case SettingsFrameType:
		frame := &SettingsFrame{}
		frame.FrameHeader = fh
		frame.Read(c.RW)
		return frame
	case WindowUpdateFrameType:
		frame := &WindowUpdateFrame{}
		frame.FrameHeader = fh
		frame.Read(c.RW)
		return frame
	default:
		log.Printf("unknown type: %v", fh.Type)
		return nil
	}
	return nil
}

func (c *Conn) WriteFrame(frame Frame) { // err
	// DEGBU
	// buf := bytes.NewBuffer([]byte{})
	// frame.Write(buf)
	// log.Println(buf.Bytes())
	switch f := frame.(type) {
	case *HeadersFrame:
		f.HeaderBlock = c.EncodeHeader(f.Headers)
		f.Length = uint16(len(f.HeaderBlock))
		f.Write(c.RW)
	default:
		f.Write(c.RW)
	}
	//	frame.Write(c.RW) // err
}

func (c *Conn) WriteString(str string) { // err
	c.Bw.WriteString(str) // err
	c.Bw.Flush()          // err
	fmt.Println(Red("send"), Indent(Blue(str)))
}

func (c *Conn) ReadResponse() *http.Response {
	res, _ := http.ReadResponse(c.Br, &http.Request{Method: "GET"}) // err

	fmt.Println(Green("recv"), Blue(Indent(ResponseString(res))), "\n")
	return res
}

func (c *Conn) EncodeHeader(header http.Header) []byte {
	return c.RequestContext.Encode(header)
}

func (c *Conn) DecodeHeader(headerBlock []byte) http.Header {
	c.ResponseContext.Decode(headerBlock)
	return c.ResponseContext.EmittedSet.Header
}
