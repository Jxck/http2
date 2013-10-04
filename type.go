package http2

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	. "github.com/jxck/color"
	. "github.com/jxck/debug"
	"github.com/jxck/hpack"
	"io"
	"log"
	"net/http"
	"strings"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

const (
	// note: 0x08 dosen't used
	DataFrameType         uint8 = 0x0
	HeadersFrameType            = 0x1
	PriorityFrameType           = 0x2
	RstStreamFrameType          = 0x3
	SettingsFrameType           = 0x4
	PushPrimiseFrameType        = 0x5
	PingFrameType               = 0x6
	GoawayFrameType             = 0x7
	WindowUpdateFrameType       = 0x9
	ContinuationFrameType       = 0xA
)

type Frame interface {
	Write(w io.Writer)
	Read(r io.Reader)
	Header() *FrameHeader
}

// Framer has 2 funcs
// ReadFrame() frame
// WriteFrame(frame)
type Framer struct {
	RW io.ReadWriter
}

func (f *Framer) WriteFrame(frame Frame) { // err
	// DEGBU
	// buf := bytes.NewBuffer([]byte{})
	// frame.Write(buf)
	// log.Println(buf.Bytes())

	frame.Write(f.RW) // err
}

func (f *Framer) ReadFrame() Frame {
	fh := &FrameHeader{} // New
	fh.Read(f.RW)        // err
	Debug(fmt.Sprintf("Type: %v", fh.Type))

	switch fh.Type {
	case DataFrameType:
		frame := &DataFrame{}
		frame.FrameHeader = fh
		frame.Data = make([]byte, frame.Length)
		frame.Read(f.RW)
		return frame
	case HeadersFrameType:
		frame := &HeadersFrame{}
		frame.FrameHeader = fh
		frame.Read(f.RW)
		return frame
	case SettingsFrameType:
		frame := &SettingsFrame{}
		frame.FrameHeader = fh
		frame.Read(f.RW)
		return frame
	case WindowUpdateFrameType:
		frame := &WindowUpdateFrame{}
		frame.FrameHeader = fh
		frame.Read(f.RW)
		return frame
	default:
		log.Printf("unknown type: %v", fh.Type)
		return nil
	}
	return nil
}

// Frame Header
//
//  0                   1                   2                   3
//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |         Length (16)           |   Type (8)    |   Flags (8)   |
// +-+-------------+---------------+-------------------------------+
// |R|                 Stream Identifier (31)                      |
// +-+-------------------------------------------------------------+
// |                   Frame Payload (0...)                      ...
// +---------------------------------------------------------------+

type FrameHeader struct {
	Length   uint16
	Type     uint8
	Flags    uint8
	R        uint8 // not care
	StreamId uint32
}

func (fh *FrameHeader) Read(r io.Reader) {
	binary.Read(r, binary.BigEndian, &fh.Length)   // err
	binary.Read(r, binary.BigEndian, &fh.Type)     // err
	binary.Read(r, binary.BigEndian, &fh.Flags)    // err
	binary.Read(r, binary.BigEndian, &fh.StreamId) // err
}

func (fh *FrameHeader) Write(w io.Writer) {
	binary.Write(w, binary.BigEndian, fh.Length)   // err
	binary.Write(w, binary.BigEndian, fh.Type)     // err
	binary.Write(w, binary.BigEndian, fh.Flags)    // err
	binary.Write(w, binary.BigEndian, fh.StreamId) // err
}

func (fh *FrameHeader) String() string {
	str := fmt.Sprintf(
		" frame <length=%v, flags=%#x, stream_id=%v>",
		fh.Length, fh.Flags, fh.StreamId,
	)
	return str
}

// DATA
type DataFrame struct {
	*FrameHeader
	Data []byte
}

func (frame *DataFrame) Read(r io.Reader) {
	binary.Read(r, binary.BigEndian, &frame.Data) // err
}

func (frame *DataFrame) Write(w io.Writer) {
	binary.Write(w, binary.BigEndian, frame.Data) // err
}

func (frame *DataFrame) Header() *FrameHeader {
	return frame.FrameHeader
}

func (frame *DataFrame) String() string {
	str := Cyan("DATA")
	str += frame.FrameHeader.String()
	return str
}

// HEADERS
//
// 0                   1                   2                   3
// 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |X|                        Priority (31)                        |
// +-+-------------------------------------------------------------+
// |                   Header Block Fragment (*)                 ...
// +---------------------------------------------------------------+

type HeadersFrame struct {
	*FrameHeader
	Priority    uint32
	HeaderBlock []byte
	Headers     http.Header
}

func (frame *HeadersFrame) Write(w io.Writer) {
	frame.FrameHeader.Write(w)
	if frame.Flags == 0x08 {
		binary.Write(w, binary.BigEndian, frame.Priority) // err
	}
	binary.Write(w, binary.BigEndian, frame.HeaderBlock) // err
}

func (frame *HeadersFrame) Read(r io.Reader) {
	if frame.Flags == 0x08 {
		binary.Read(r, binary.BigEndian, &frame.Priority) // err
	}
	b := make([]byte, frame.Length)
	// TODO: Buffer.Read()
	binary.Read(r, binary.BigEndian, &b) // err

	frame.HeaderBlock = b

	client := hpack.NewResponseContext()

	client.Decode(b)
	frame.Headers = client.EmittedSet.Header
}

func (frame *HeadersFrame) Header() *FrameHeader {
	return frame.FrameHeader
}

func (frame *HeadersFrame) String() string {
	str := Cyan("HEADERS")
	str += frame.FrameHeader.String()

	if frame.Flags == 0x4 {
		str += "\n; END_HEADERS"
	}

	// TODO: ; First response header

	for name, value := range frame.Headers {
		str += fmt.Sprintf("\n%s: %s", Blue(name), strings.Join(value, ","))
	}

	return str
}

// PRIORITY
//
// 0                   1                   2                   3
// 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |X|                        Priority (31)                        |
// +-+-------------------------------------------------------------+
//
//
// RST_STREAM
//
// 0                   1                   2                   3
// 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                        Error Code (32)                        |
// +---------------------------------------------------------------+
//
// SETTINGS Frame
//
//  0                   1                   2                   3
//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |  Reserved (8) |            Setting Identifier (24)            |
// +---------------+-----------------------------------------------+
// |                        Value (32)                             |
// +---------------------------------------------------------------+

type SettingsId uint32

const (
	SETTINGS_MAX_CONCURRENT_STREAMS SettingsId = 4
	SETTINGS_INITIAL_WINDOW_SIZE               = 7
	SETTINGS_FLOW_CONTROL_OPTIONS              = 10
)

type Setting struct {
	Reserved   uint8
	SettingsId SettingsId
	Value      uint32
}

type SettingsFrame struct {
	*FrameHeader
	Settings []Setting
}

func DefaultSettingsFrame() *SettingsFrame {
	setting1 := Setting{ // 4:100
		SettingsId: SETTINGS_MAX_CONCURRENT_STREAMS,
		Value:      100,
	}
	setting2 := Setting{ // 7:65535
		SettingsId: SETTINGS_INITIAL_WINDOW_SIZE,
		Value:      65535,
	}
	fh := &FrameHeader{
		Length:   16,
		Type:     SettingsFrameType,
		StreamId: 0,
	}
	settingsFrame := &SettingsFrame{
		FrameHeader: fh,
		Settings:    []Setting{setting1, setting2},
	}
	return settingsFrame
}

func NoFlowSettingsFrame() *SettingsFrame {
	setting1 := Setting{ // 4:100
		SettingsId: SETTINGS_MAX_CONCURRENT_STREAMS,
		Value:      100,
	}
	setting2 := Setting{ // 7:65535
		SettingsId: SETTINGS_INITIAL_WINDOW_SIZE,
		Value:      65535,
	}
	setting3 := Setting{ // 10:1
		SettingsId: SETTINGS_FLOW_CONTROL_OPTIONS,
		Value:      1,
	}
	fh := &FrameHeader{
		Length:   24,
		Type:     SettingsFrameType,
		StreamId: 0,
	}
	settingsFrame := &SettingsFrame{
		FrameHeader: fh,
		Settings:    []Setting{setting1, setting2, setting3},
	}
	return settingsFrame
}

func (frame *SettingsFrame) Write(w io.Writer) {
	frame.FrameHeader.Write(w)
	for _, setting := range frame.Settings {
		binary.Write(w, binary.BigEndian, setting.SettingsId) // err
		binary.Write(w, binary.BigEndian, setting.Value)      // err
	}
}

func (frame *SettingsFrame) Read(r io.Reader) {
	for niv := frame.Length / 8; niv > 0; niv-- {
		s := Setting{}

		var firstByte uint32
		binary.Read(r, binary.BigEndian, &firstByte) // err
		s.SettingsId = SettingsId(firstByte & 0xFFFFFF)
		s.Reserved = uint8(firstByte >> 24)
		binary.Read(r, binary.BigEndian, &s.Value) // err
		frame.Settings = append(frame.Settings, s)
	}
}

// TODO: fixme
func (frame *SettingsFrame) PayloadBase64URL() string {
	buf := bytes.NewBuffer([]byte{})
	frame.Write(buf)
	str := base64.URLEncoding.EncodeToString(buf.Bytes()[8:])
	str = strings.Replace(str, "=", "", -1)
	return str
}

func (frame *SettingsFrame) Header() *FrameHeader {
	return frame.FrameHeader
}

func (frame *SettingsFrame) String() string {
	str := Cyan("SETTINGS")
	str += frame.FrameHeader.String()
	str += fmt.Sprintf("\n(niv=%v)", len(frame.Settings))
	for _, s := range frame.Settings {
		str += fmt.Sprintf("\n[%v:%v]", s.SettingsId, s.Value)
	}
	return str
}

//
// PUSH_PROMISE
//
// 0                   1                   2                   3
// 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |X|                Promised-Stream-ID (31)                      |
// +-+-------------------------------------------------------------+
// |                 Header Block Fragment (*)                   ...
// +---------------------------------------------------------------+
//
//
//
//
// PING
//
// 0                   1                   2                   3
// 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                                                               |
// |                      Opaque Data (64)                         |
// |                                                               |
// +---------------------------------------------------------------+
//
//
// GOAWAY
//
// 0                   1                   2                   3
// 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |X|                  Last-Stream-ID (31)                        |
// +-+-------------------------------------------------------------+
// |                      Error Code (32)                          |
// +---------------------------------------------------------------+
// |                  Additional Debug Data (*)                    |
// +---------------------------------------------------------------+
//
//
//
//
// WINDOW_UPDATE
//
// 0                   1                   2                   3
// 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |X|              Window Size Increment (31)                     |
// +-+-------------------------------------------------------------+

type WindowUpdateFrame struct {
	*FrameHeader
	WindowSizeIncrement uint32
}

func CreateWindowUpdateFrame(size, streamId uint32) *WindowUpdateFrame {
	fh := &FrameHeader{
		Length:   4,
		Type:     WindowUpdateFrameType,
		StreamId: streamId,
	}
	frame := &WindowUpdateFrame{}
	frame.FrameHeader = fh
	frame.WindowSizeIncrement = size
	return frame
}

func (frame *WindowUpdateFrame) Write(w io.Writer) {
	frame.FrameHeader.Write(w)
	binary.Write(w, binary.BigEndian, frame.WindowSizeIncrement) // err
}

func (frame *WindowUpdateFrame) Read(r io.Reader) {
	binary.Read(r, binary.BigEndian, &frame.WindowSizeIncrement) // err
}

func (frame *WindowUpdateFrame) Header() *FrameHeader {
	return frame.FrameHeader
}

func (frame *WindowUpdateFrame) String() string {
	str := Cyan("WINDOW_UPDATE")
	str += frame.FrameHeader.String()
	str += fmt.Sprintf("\n(window_size_increment=%d)", frame.WindowSizeIncrement)
	return str
}

// CONTINUATION
//
// 0                   1                   2                   3
// 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                   Header Block Fragment (*)                 ...
// +---------------------------------------------------------------+
