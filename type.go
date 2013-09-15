package http2

import (
	"bytes"
	_ "encoding/base64"
	"encoding/binary"
	"fmt"
	. "github.com/jxck/color"
	"log"
	"net"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

type FrameType uint8

const (
	// note: 0x08 dosen't used
	HeadersFrameType      FrameType = 0x1
	PriorityFrameType               = 0x2
	RstStreamFrameType              = 0x3
	SettingsFrameType               = 0x4
	PushPrimiseFrameType            = 0x5
	PingFrameType                   = 0x6
	GoawayFrameType                 = 0x7
	WindowUpdateFrameType           = 0x9
	ContinuationFrameType           = 0xA
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

func (fh *FrameHeader) Decode(conn net.Conn) {
	b := make([]byte, 8)
	n, err := conn.Read(b)

	buf := bytes.NewBuffer(b)
	binary.Read(buf, binary.BigEndian, &fh.Length)   // err
	binary.Read(buf, binary.BigEndian, &fh.Type)     // err
	binary.Read(buf, binary.BigEndian, &fh.Flags)    // err
	binary.Read(buf, binary.BigEndian, &fh.StreamId) // err

	b = make([]byte, fh.Length)
	n, err = conn.Read(b)
	log.Println(n, err)
	buf = bytes.NewBuffer(b)

	switch fh.Type {
	case 4:
		frame := NewSettingsFrame(fh)
		frame.Decode(buf)
		fmt.Println(&frame)
	default:
		log.Println("other")
	}
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

func NewSettingsFrame(fh *FrameHeader) SettingsFrame {
	frame := SettingsFrame{}
	frame.FrameHeader = *fh
	return frame
}

type SettingsFrame struct {
	FrameHeader
	Settings []Setting
}

type Setting struct {
	Reserved   uint8
	SettingsId SettingsId
	Value      uint32
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

func (frame *SettingsFrame) String() string {
	str := Pink("SETTINGS")
	str += White(fmt.Sprintf(" frame <length=%v, flags=%v, stream_id=%v>\n(niv=%v)",
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
