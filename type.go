package main

import (
	"bytes"
	_ "encoding/base64"
	"encoding/binary"
	"fmt"
	"log"
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

func (fh *FrameHeader) Decode(b []byte) {
	buf := bytes.NewBuffer(b)
	binary.Read(buf, binary.BigEndian, &fh.Length)   // err
	binary.Read(buf, binary.BigEndian, &fh.Type)     // err
	binary.Read(buf, binary.BigEndian, &fh.Flags)    // err
	binary.Read(buf, binary.BigEndian, &fh.StreamId) // err

	switch fh.Type {
	case 4:
		frame := NewSettingsFrame(fh)
		frame.Decode(buf)
	default:
		log.Println("other")
	}
}

func main() {
	buf := []byte{
		0x0, 0x8, 0x4, 0,
		// 00000000 00001000,00000100,00000000
		0, 0, 0, 0,
		// 00000000 00000000 00000000 00000000
		0, 0, 0, 0x4,
		// 00000000 00000000 00000000 00000100
		0, 0, 0, 0xc4,
		// 00000000 00000000 00000000 11000100
	}
	fh := FrameHeader{}
	fh.Decode(buf)

	// str := base64.StdEncoding.EncodeToString(buf)
	// log.Println(str)
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

//type HeadersFrame struct {
//	HeaderFrame
//	Priority uint32
//	HeaderBlock
//}

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
		binary.Read(buf, binary.BigEndian, s.Value) // err
		frame.Settings = append(frame.Settings, s)
	}
	fmt.Println(frame)
}

func (frame *SettingsFrame) String() string {
	niv := len(frame.Settings)
	str := fmt.Sprintf("SETTINGS frame <length=%v, flags=%v, stream_id=%v>\n(niv=%v)",
		frame.Length, frame.Flags, frame.StreamId, niv)
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
