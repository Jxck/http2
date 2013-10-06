package http2

import (
	"io"
	"log"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

// Conn has 2 funcs
// ReadFrame() frame
// WriteFrame(frame)
type Conn struct {
	RW io.ReadWriter
}

func (c *Conn) WriteFrame(frame Frame) { // err
	// DEGBU
	// buf := bytes.NewBuffer([]byte{})
	// frame.Write(buf)
	// log.Println(buf.Bytes())

	frame.Write(c.RW) // err
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
