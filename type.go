package http2

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	. "github.com/jxck/color"
	. "github.com/jxck/debug"
	"github.com/jxck/hpack"
	"log"
	"net"
	"net/http"
	"strings"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

type Frame interface {
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
	R        uint8
	StreamId uint32
}

func (fh *FrameHeader) Decode(conn net.Conn) Frame {
	b := make([]byte, 8)
	conn.Read(b) // err

	buf := bytes.NewBuffer(b)
	binary.Read(buf, binary.BigEndian, &fh.Length)   // err
	binary.Read(buf, binary.BigEndian, &fh.Type)     // err
	binary.Read(buf, binary.BigEndian, &fh.Flags)    // err
	binary.Read(buf, binary.BigEndian, &fh.StreamId) // err

	Debug(fmt.Sprintf("Type: %v", fh.Type))

	var l, n uint16
	l = fh.Length
	b = make([]byte, fh.Length)

	// read until fh.Length
	for l > 0 {
		bb := make([]byte, l)
		nn, _ := conn.Read(bb) // err
		copy(b[n:], bb[:nn])
		n += uint16(nn)
		l -= uint16(nn)
	}

	buf = bytes.NewBuffer(b)

	switch fh.Type {
	case DataFrameType:
		frame := NewDataFrame(fh)
		frame.Decode(buf)
		return frame
	case HeadersFrameType:
		frame := NewHeadersFrame(fh)
		frame.Decode(buf)
		return frame
	case SettingsFrameType:
		frame := NewSettingsFrame(fh)
		frame.Decode(buf)
		return frame
	case WindowUpdateFrameType:
		return nil
	default:
		log.Println("other")
		return nil
	}
	return nil
}

// DATA
type DataFrame struct {
	FrameHeader
	Data []byte
}

func NewDataFrame(fh *FrameHeader) *DataFrame {
	frame := &DataFrame{}
	frame.FrameHeader = *fh
	frame.Data = make([]byte, frame.Length)
	return frame
}

func (frame *DataFrame) Decode(buf *bytes.Buffer) {
	binary.Read(buf, binary.BigEndian, &frame.Data) // err
}

func (frame *DataFrame) String() string {
	str := Cyan("DATA")
	str += White(fmt.Sprintf(" frame <length=%v, flags=%#x, stream_id=%v>",
		frame.Length, frame.Flags, frame.StreamId))
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
	FrameHeader
	Priority    uint32
	HeaderBlock []byte
	Header      http.Header
}

func NewHeadersFrame(fh *FrameHeader) *HeadersFrame {
	frame := &HeadersFrame{}
	frame.FrameHeader = *fh
	return frame
}

func (frame *HeadersFrame) Encode() *bytes.Buffer {
	buf := bytes.NewBuffer([]byte{})

	binary.Write(buf, binary.BigEndian, frame.Length)      // err
	binary.Write(buf, binary.BigEndian, frame.Type)        // err
	binary.Write(buf, binary.BigEndian, frame.Flags)       // err
	binary.Write(buf, binary.BigEndian, frame.StreamId)    // err
	binary.Write(buf, binary.BigEndian, frame.Priority)    // err
	binary.Write(buf, binary.BigEndian, frame.HeaderBlock) // err

	return buf
}

func (frame *HeadersFrame) Decode(buf *bytes.Buffer) {
	if frame.Flags == 0x08 {
		binary.Read(buf, binary.BigEndian, &frame.Priority) // err
	}
	b := make([]byte, frame.Length)
	// TODO: Buffer.Read()
	binary.Read(buf, binary.BigEndian, &b) // err

	frame.HeaderBlock = b

	client := hpack.NewResponseContext()

	client.Decode(b)
	frame.Header = client.EmittedSet.Header
}

func (frame *HeadersFrame) String() string {
	str := Cyan("HEADERS")
	str += White(fmt.Sprintf(" frame <length=%v, flags=%#x, stream_id=%v>\n",
		frame.Length, frame.Flags, frame.StreamId))

	if frame.Flags == 0x4 {
		str += White("; END_HEADERS\n")
	}

	// TODO: ; First response header

	for name, value := range frame.Header {
		str += fmt.Sprintf("%s: %s\n", Blue(name), strings.Join(value, ","))
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

func NewSettingsFrame(fh *FrameHeader) *SettingsFrame {
	frame := &SettingsFrame{}
	frame.FrameHeader = *fh
	return frame
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
	fh := FrameHeader{
		Length:   64,
		Type:     SettingsFrameType,
		StreamId: 0,
	}
	settingsFrame := &SettingsFrame{
		FrameHeader: fh,
		Settings:    []Setting{setting1, setting2},
	}
	return settingsFrame
}

type SettingsFrame struct {
	FrameHeader
	Settings []Setting
}

func (frame *SettingsFrame) Encode() *bytes.Buffer {
	buf := bytes.NewBuffer([]byte{})

	binary.Write(buf, binary.BigEndian, frame.Length)   // err
	binary.Write(buf, binary.BigEndian, frame.Type)     // err
	binary.Write(buf, binary.BigEndian, frame.Flags)    // err
	binary.Write(buf, binary.BigEndian, frame.StreamId) // err
	for _, setting := range frame.Settings {
		binary.Write(buf, binary.BigEndian, setting.SettingsId) // err
		binary.Write(buf, binary.BigEndian, setting.Value)      // err
	}

	return buf
}

func (frame *SettingsFrame) Decode(buf *bytes.Buffer) {
	for niv := frame.Length / 8; niv > 0; niv-- {
		s := Setting{}

		var firstByte uint32
		binary.Read(buf, binary.BigEndian, &firstByte) // err
		s.SettingsId = SettingsId(firstByte & 0xFFFFFF)
		s.Reserved = uint8(firstByte >> 24)
		binary.Read(buf, binary.BigEndian, &s.Value) // err
		frame.Settings = append(frame.Settings, s)
	}
}

func (frame *SettingsFrame) PayloadBase64URL() string {
	bytes := frame.Encode().Bytes()
	str := base64.URLEncoding.EncodeToString(bytes[8:])
	str = strings.Replace(str, "=", "", -1)
	return str
}

func (frame *SettingsFrame) String() string {
	str := Cyan("SETTINGS")
	str += White(fmt.Sprintf(" frame <length=%v, flags=%#x, stream_id=%v>\n(niv=%v)",
		frame.Length, frame.Flags, frame.StreamId, len(frame.Settings)))
	for _, s := range frame.Settings {
		str += White(fmt.Sprintf("\n[%v:%v]", s.SettingsId, s.Value))
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
//
//
//
// CONTINUATION
//
// 0                   1                   2                   3
// 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                   Header Block Fragment (*)                 ...
// +---------------------------------------------------------------+
