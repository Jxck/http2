package frame

import (
	"encoding/binary"
	"fmt"
	. "github.com/Jxck/color"
	. "github.com/Jxck/logger"
	"io"
	"log"
	"net/http"
	"strings"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

type FrameType uint8

const (
	DataFrameType         FrameType = 0x0
	HeadersFrameType                = 0x1
	PriorityFrameType               = 0x2
	RstStreamFrameType              = 0x3
	SettingsFrameType               = 0x4
	PushPromiseFrameType            = 0x5
	PingFrameType                   = 0x6
	GoAwayFrameType                 = 0x7
	WindowUpdateFrameType           = 0x8
	ContinuationFrameType           = 0x9
)

func (frameType FrameType) String() string {
	names := []string{
		"DATA",
		"HEADERS",
		"PRIORITY",
		"RST_STREAM",
		"SETTINGS",
		"PUSH_PROMISE",
		"PING",
		"GOAWAY",
		"WINDOW_UPDATE",
		"CONTINUATION",
	}
	return names[int(frameType)]
}

// For RST_STREAM and GOAWAY Frame
type ErrorCode uint32

const (
	NO_ERROR            ErrorCode = 0x0
	PROTOCOL_ERROR      ErrorCode = 0x1
	INTERNAL_ERROR      ErrorCode = 0x2
	FLOW_CONTROL_ERROR  ErrorCode = 0x3
	SETTINGS_TIMEOUT    ErrorCode = 0x4
	STREAM_CLOSED       ErrorCode = 0x5
	FRAME_SIZE_ERROR    ErrorCode = 0x6
	REFUSED_STREAM      ErrorCode = 0x7
	CANCEL              ErrorCode = 0x8
	COMPRESSION_ERROR   ErrorCode = 0x9
	CONNECT_ERROR       ErrorCode = 0xa
	ENHANCE_YOUR_CALM   ErrorCode = 0xb
	INADEQUATE_SECURITY ErrorCode = 0xc
	HTTP_1_1_REQUIRED   ErrorCode = 0xd
)

func (e ErrorCode) String() string {
	errors := []string{
		"NO_ERROR",
		"PROTOCOL_ERROR",
		"INTERNAL_ERROR",
		"FLOW_CONTROL_ERROR",
		"SETTINGS_TIMEOUT",
		"STREAM_CLOSED",
		"FRAME_SIZE_ERROR",
		"REFUSED_STREAM",
		"CANCEL",
		"COMPRESSION_ERROR",
		"CONNECT_ERROR",
		"ENHANCE_YOUR_CALM",
		"INADEQUATE_SECURITY",
		"HTTP_1_1_REQUIRED",
	}
	return errors[int(e)]
}

type H2Error struct {
	ErrorCode           ErrorCode
	AdditiolanDebugData string
}

func (e H2Error) Error() string {
	return e.ErrorCode.String()
}

func (e H2Error) String() string {
	return fmt.Sprintf("%v(%v)", e.ErrorCode, e.AdditiolanDebugData)
}

// Flags
type Flag uint8

const (
	UNSET       Flag = 0x0
	END_STREAM       = 0x1
	ACK              = 0x1 // for settings
	END_HEADERS      = 0x4
	PADDED           = 0x8
	PRIORITY         = 0x20
)

type Frame interface {
	Write(w io.Writer) error
	Read(r io.Reader) error
	Header() *FrameHeader
	String() string
}

// map of FrameType and FrameInitializer
var FrameMap = map[FrameType](func(*FrameHeader) Frame){
	DataFrameType:         func(fh *FrameHeader) Frame { return &DataFrame{FrameHeader: fh} },
	HeadersFrameType:      func(fh *FrameHeader) Frame { return &HeadersFrame{FrameHeader: fh} },
	PriorityFrameType:     func(fh *FrameHeader) Frame { return &PriorityFrame{FrameHeader: fh} },
	RstStreamFrameType:    func(fh *FrameHeader) Frame { return &RstStreamFrame{FrameHeader: fh} },
	SettingsFrameType:     func(fh *FrameHeader) Frame { return &SettingsFrame{FrameHeader: fh} },
	PushPromiseFrameType:  func(fh *FrameHeader) Frame { return &PushPromiseFrame{FrameHeader: fh} },
	PingFrameType:         func(fh *FrameHeader) Frame { return &PingFrame{FrameHeader: fh} },
	GoAwayFrameType:       func(fh *FrameHeader) Frame { return &GoAwayFrame{FrameHeader: fh} },
	WindowUpdateFrameType: func(fh *FrameHeader) Frame { return &WindowUpdateFrame{FrameHeader: fh} },
	ContinuationFrameType: func(fh *FrameHeader) Frame { return &ContinuationFrame{FrameHeader: fh} },
}

// Frame Header
//
// +-----------------------------------------------+
// |                 Length (24)                   |
// +---------------+---------------+---------------+
// |   Type (8)    |   Flags (8)   |
// +-+-------------+---------------+-------------------------------+
// |R|                 Stream Identifier (31)                      |
// +=+=============================================================+
// |                   Frame Payload (0...)                      ...
// +---------------------------------------------------------------+
type FrameHeader struct {
	Length            uint32 // 24bit
	Type              FrameType
	Flags             Flag
	StreamID          uint32 // R+31bit
	MaxFrameSize      int32
	MaxHeaderListSize int32
}

func NewFrameHeader(length uint32, types FrameType, flags Flag, streamid uint32) *FrameHeader {
	fh := &FrameHeader{
		Length:   length,
		Type:     types,
		Flags:    flags,
		StreamID: streamid,
	}
	return fh
}

func (fh *FrameHeader) Read(r io.Reader) (err error) {
	// read 32 bit
	var first uint32
	err = binary.Read(r, binary.BigEndian, &first)
	if err != nil {
		return err
	}

	// last 8 bit for type
	fh.Type = FrameType(first & 0xFF)
	Trace("type = %s", fh.Type)

	if fh.Type < 0 || 0x9 < fh.Type {
		Error("ingore this frame")
		// TODO: ignore this frame or return err ?
		return
	}

	// first 24 bit for length
	fh.Length = first >> 8
	Trace("length = %d", fh.Length)

	if fh.Type == RstStreamFrameType && fh.Length != 4 {
		msg := fmt.Sprintf("frame size of RST_STREAM should be 4 but %v", fh.Length)
		Error(Red(msg))
		return &H2Error{FRAME_SIZE_ERROR, msg}
	}

	if int32(fh.Length) > fh.MaxFrameSize {
		msg := fmt.Sprintf("frame size(%v) is larger than MAX_FRAME_SIZE(%v)", fh.Length, fh.MaxFrameSize)
		Error(Red(msg))
		return &H2Error{FRAME_SIZE_ERROR, msg}
	}

	// read 8 bit for Flags
	err = binary.Read(r, binary.BigEndian, &fh.Flags)
	if err != nil {
		return err
	}
	Trace("flags = %d", fh.Flags)

	// read 32 bit for StreamID
	var last uint32
	err = binary.Read(r, binary.BigEndian, &last)
	if err != nil {
		return err
	}
	fh.StreamID = last & 0x7FFFFFFF
	Trace("streamId = %d", fh.StreamID)

	return err
}

func (fh *FrameHeader) Write(w io.Writer) (err error) {
	// write length + type as 32bit
	var first uint32 = fh.Length<<8 + uint32(fh.Type)
	err = binary.Write(w, binary.BigEndian, &first)
	if err != nil {
		return err
	}

	// write flags
	err = binary.Write(w, binary.BigEndian, &fh.Flags)
	if err != nil {
		return err
	}

	// write stream id
	err = binary.Write(w, binary.BigEndian, &fh.StreamID)
	if err != nil {
		return err
	}

	return err
}

func (fh *FrameHeader) String() string {
	str := fmt.Sprintf(
		" frame <length=%v, flags=%#x, stream_id=%v>",
		fh.Length, fh.Flags, fh.StreamID,
	)
	return str
}

// DATA
//
// +---------------+
// |Pad Length? (8)|
// +---------------+-----------------------------------------------+
// |                            Data (*)                         ...
// +---------------------------------------------------------------+
// |                           Padding (*)                       ...
// +---------------------------------------------------------------+
type DataFrame struct {
	*FrameHeader
	PadLength uint8
	Data      []byte
	Padding   []byte
}

func NewDataFrame(flags Flag, streamID uint32, data []byte, padding []byte) *DataFrame {
	var padded bool = flags&PADDED == PADDED

	length := len(data)

	if padded {
		length = length + len(padding) + 1
	} else {
		padding = nil
	}

	fh := NewFrameHeader(uint32(length), DataFrameType, flags, streamID)

	dataFrame := &DataFrame{
		FrameHeader: fh,
		PadLength:   uint8(len(padding)),
		Data:        data,
		Padding:     padding,
	}

	return dataFrame
}

func (frame *DataFrame) Read(r io.Reader) (err error) {
	var frameLen uint32 = frame.Length
	var padded bool = frame.Flags&PADDED == PADDED

	if padded {
		// read 8 bit for padding length
		err = binary.Read(r, binary.BigEndian, &frame.PadLength)
		if err != nil {
			return err
		}

		if uint32(frame.PadLength) > frameLen {
			msg := fmt.Sprintf("Pad Length(%v) is larger than frame.Length(%v)", frame.PadLength, frameLen)
			Error(Red(msg))
			return &H2Error{PROTOCOL_ERROR, msg}
		}

		frameLen = frameLen - 1 // (remove pad length)
	}

	// read frame length bit for data
	data := make([]byte, frameLen)
	err = binary.Read(r, binary.BigEndian, &data)
	if err != nil {
		return err
	}

	if padded {
		// data + padding
		boundary := len(data) - int(frame.PadLength)
		frame.Data = data[:boundary]
		frame.Padding = data[boundary:]
	} else {
		// data only
		frame.Data = data
	}

	return err
}

func (frame *DataFrame) Write(w io.Writer) (err error) {
	err = frame.FrameHeader.Write(w)
	if err != nil {
		return err
	}

	var padded bool = frame.Flags&PADDED == PADDED

	if padded {
		// write padding length
		err = binary.Write(w, binary.BigEndian, &frame.PadLength)
		if err != nil {
			return err
		}
	}

	// write data
	err = binary.Write(w, binary.BigEndian, &frame.Data)
	if err != nil {
		return err
	}

	if padded {
		// write padding data
		err = binary.Write(w, binary.BigEndian, &frame.Padding)
		if err != nil {
			return err
		}
	}
	return err
}

func (frame *DataFrame) Header() *FrameHeader {
	return frame.FrameHeader
}

func (frame *DataFrame) String() string {
	str := Cyan("DATA")
	str += frame.FrameHeader.String()

	if frame.Flags&END_STREAM == END_STREAM {
		str += "\n; END_STREAM"
	}

	// Print first 8 byte of Data or all
	window := len(frame.Data)
	if window == 0 {
		// no data do nothing
		return str
	} else if window > 32 {
		// trim to 32 byte
		window = 32
	}
	str += fmt.Sprintf("\n%q...", string(frame.Data[:window]))
	return str
}

// HEADERS
//
// +---------------+
// |Pad Length? (8)|
// +-+-------------+-----------------------------------------------+
// |E|                 Stream Dependency? (31)                     |
// +-+-------------+-----------------------------------------------+
// |  Weight? (8)  |
// +-+-------------+-----------------------------------------------+
// |                   Header Block Fragment (*)                 ...
// +---------------------------------------------------------------+
// |                           Padding (*)                       ...
// +---------------------------------------------------------------+
type HeadersFrame struct {
	*FrameHeader
	PadLength           uint8
	DependencyTree      *DependencyTree
	HeaderBlockFragment []byte
	Headers             http.Header
	Padding             []byte
}

type DependencyTree struct {
	Exclusive        bool
	StreamDependency uint32
	Weight           uint8
}

func NewHeadersFrame(flags Flag, streamID uint32, dependencyTree *DependencyTree, headerBlockFragment []byte, padding []byte) *HeadersFrame {
	var padded bool = flags&PADDED == PADDED
	var priority bool = flags&PRIORITY == PRIORITY

	length := len(headerBlockFragment)
	if padded {
		length = length + len(padding) + 1
	}

	if priority {
		length = length + 5
	}

	fh := NewFrameHeader(uint32(length), HeadersFrameType, flags, streamID)

	headersFrame := &HeadersFrame{
		FrameHeader:         fh,
		PadLength:           uint8(len(padding)),
		DependencyTree:      dependencyTree,
		HeaderBlockFragment: headerBlockFragment,
		Padding:             padding,
	}

	return headersFrame
}

func (frame *HeadersFrame) Read(r io.Reader) (err error) {
	var frameLen uint32 = frame.Length
	var padded bool = frame.Flags&PADDED == PADDED
	var priority bool = frame.Flags&PRIORITY == PRIORITY

	if padded {
		err = binary.Read(r, binary.BigEndian, &frame.PadLength)
		if err != nil {
			return err
		}

		if uint32(frame.PadLength) > frameLen {
			msg := fmt.Sprintf("Pad Length(%v) is larger than frame.Length(%v)", frame.PadLength, frameLen)
			Error(Red(msg))
			return &H2Error{PROTOCOL_ERROR, msg}
		}

		frameLen = frameLen - 1 // remove pad length
	}

	if priority {
		dependencyTree := new(DependencyTree)

		// read 32bit for e + streamdependency
		var u32 uint32
		err = binary.Read(r, binary.BigEndian, &u32)
		if err != nil {
			return err
		}

		if u32&0x80000000 == 0x80000000 {
			dependencyTree.Exclusive = true
		}
		dependencyTree.StreamDependency = u32 & 0x7FFFFFFF

		frameLen = frameLen - 4 // remove stream dependency length

		err = binary.Read(r, binary.BigEndian, &dependencyTree.Weight)
		if err != nil {
			return err
		}

		// add 1 for weight
		dependencyTree.Weight = dependencyTree.Weight + 1
		frameLen = frameLen - 1 // remove weight length

		frame.DependencyTree = dependencyTree
	}

	// read frame length bit for data
	data := make([]byte, frameLen)
	err = binary.Read(r, binary.BigEndian, &data)
	if err != nil {
		return err
	}

	if padded {
		// header block + padding
		boundary := len(data) - int(frame.PadLength)
		frame.HeaderBlockFragment = data[:boundary]
		frame.Padding = data[boundary:]
	} else {
		// header block only
		frame.HeaderBlockFragment = data
	}

	return err
}

func (frame *HeadersFrame) Write(w io.Writer) (err error) {
	err = frame.FrameHeader.Write(w)
	if err != nil {
		return err
	}

	var padded bool = frame.Flags&PADDED == PADDED
	var priority bool = frame.Flags&PRIORITY == PRIORITY

	if padded {
		err = binary.Write(w, binary.BigEndian, &frame.PadLength)
		if err != nil {
			return err
		}
	}

	if priority {
		streamDependency := frame.DependencyTree.StreamDependency
		if frame.DependencyTree.Exclusive {
			streamDependency = streamDependency + 0x80000000
		}
		err = binary.Write(w, binary.BigEndian, &streamDependency)
		if err != nil {
			return err
		}

		weight := frame.DependencyTree.Weight - 1
		err = binary.Write(w, binary.BigEndian, &weight)
		if err != nil {
			return err
		}
	}
	err = binary.Write(w, binary.BigEndian, &frame.HeaderBlockFragment)
	if err != nil {
		return err
	}

	if padded {
		err = binary.Write(w, binary.BigEndian, &frame.Padding)
		if err != nil {
			return err
		}
	}

	return err
}

func (frame *HeadersFrame) Header() *FrameHeader {
	return frame.FrameHeader
}

func (frame *HeadersFrame) String() string {
	str := Cyan("HEADERS")
	str += frame.FrameHeader.String()

	if frame.Flags&END_STREAM == END_STREAM {
		str += "\n; END_STREAM"
	}

	if frame.Flags&END_HEADERS == END_HEADERS {
		str += "\n; END_HEADERS"
	}

	if frame.Flags&PRIORITY == PRIORITY {
		str += "\n; PRIORITY"
	}

	if frame.Flags&PADDED == PADDED {
		str += "\n; PADDED"
	}

	// TODO: ; First response header

	// print pseudo headers first
	pseudo := []string{":status", ":method", ":scheme", ":authority", ":path"}
	for _, name := range pseudo {
		value := frame.Headers.Get(name)
		if value != "" {
			str += fmt.Sprintf("\n%s: %s", Navy(name), value)
		}
	}

	for name, value := range frame.Headers {
		if strings.HasPrefix(name, ":") {
			continue
		}
		str += fmt.Sprintf("\n%s: %s", Navy(name), strings.Join(value, ","))
	}

	return str
}

// PRIORITY
//
// +-+-------------------------------------------------------------+
// |E|                  Stream Dependency (31)                     |
// +-+-------------+-----------------------------------------------+
// |   Weight (8)  |
// +-+-------------+
type PriorityFrame struct {
	*FrameHeader
	Exclusive        bool
	StreamDependency uint32
	Weight           uint8
}

func NewPriorityFrame(streamID uint32, exclusive bool, streamDependency uint32, weight uint8) *PriorityFrame {
	var length uint32 = 5

	fh := NewFrameHeader(length, PriorityFrameType, UNSET, streamID)
	frame := &PriorityFrame{
		FrameHeader:      fh,
		Exclusive:        exclusive,
		StreamDependency: streamDependency,
		Weight:           weight,
	}
	return frame
}

func (frame *PriorityFrame) Read(r io.Reader) (err error) {
	var u32 uint32
	err = binary.Read(r, binary.BigEndian, &u32)
	if err != nil {
		return err
	}

	if u32&0x80000000 == 0x80000000 {
		frame.Exclusive = true
	}
	frame.StreamDependency = u32 & 0x7FFFFFFF

	err = binary.Read(r, binary.BigEndian, &frame.Weight)
	if err != nil {
		return err
	}
	return err
}

func (frame *PriorityFrame) Write(w io.Writer) (err error) {
	frame.FrameHeader.Write(w)

	streamDependency := frame.StreamDependency
	if frame.Exclusive {
		streamDependency = streamDependency + 0x80000000
	}
	err = binary.Write(w, binary.BigEndian, &streamDependency)
	if err != nil {
		return err
	}

	err = binary.Write(w, binary.BigEndian, &frame.Weight)
	if err != nil {
		return err
	}
	return err
}

func (frame *PriorityFrame) Header() *FrameHeader {
	return frame.FrameHeader
}

func (frame *PriorityFrame) String() string {
	str := Cyan("RRIORITY")
	str += frame.FrameHeader.String()
	str += fmt.Sprintf("\n(Stream Dependency=%d, Weight=%d)", frame.StreamDependency, frame.Weight)
	return str
}

// RST_STREAM
//
// +---------------------------------------------------------------+
// |                        Error Code (32)                        |
// +---------------------------------------------------------------+
type RstStreamFrame struct {
	*FrameHeader
	ErrorCode ErrorCode
}

func NewRstStreamFrame(streamID uint32, errorCode ErrorCode) *RstStreamFrame {
	var length uint32 = 4

	fh := NewFrameHeader(length, RstStreamFrameType, UNSET, streamID)
	frame := &RstStreamFrame{
		FrameHeader: fh,
		ErrorCode:   errorCode,
	}
	return frame
}

func (frame *RstStreamFrame) Read(r io.Reader) (err error) {
	err = binary.Read(r, binary.BigEndian, &frame.ErrorCode)
	if err != nil {
		return err
	}
	return err
}

func (frame *RstStreamFrame) Write(w io.Writer) (err error) {
	frame.FrameHeader.Write(w)
	err = binary.Write(w, binary.BigEndian, &frame.ErrorCode)
	if err != nil {
		return err
	}
	return err
}

func (frame *RstStreamFrame) Header() *FrameHeader {
	return frame.FrameHeader
}

func (frame *RstStreamFrame) String() string {
	str := Cyan("RST_STREAM")
	str += frame.FrameHeader.String()
	str += fmt.Sprintf("\n(Error Code=%s(%d))", Red(frame.ErrorCode.String()), frame.ErrorCode)
	return str
}

// SETTINGS Frame
//
// +-------------------------------+
// |       Identifier (16)         |
// +-------------------------------+-------------------------------+
// |                        Value (32)                             |
// +---------------------------------------------------------------+
const (
	DEFAULT_HEADER_TABLE_SIZE      int32 = 4096
	DEFAULT_ENABLE_PUSH                  = 1
	DEFAULT_MAX_CONCURRENT_STREAMS       = 2<<30 - 1 // actual infinite but 2^31-1 in this imple
	DEFAULT_INITIAL_WINDOW_SIZE          = 65535
	DEFAULT_MAX_FRAME_SIZE               = 16384
	DEFAULT_MAX_HEADER_LIST_SIZE         = 2<<30 - 1 // actual infinite but 2^31-1 in this imple
)

type SettingsID uint16

const (
	SETTINGS_HEADER_TABLE_SIZE      SettingsID = 0x1 // 4096
	SETTINGS_ENABLE_PUSH                       = 0x2 // 1
	SETTINGS_MAX_CONCURRENT_STREAMS            = 0x3 // (infinite)
	SETTINGS_INITIAL_WINDOW_SIZE               = 0x4 // 65535
	SETTINGS_MAX_FRAME_SIZE                    = 0x5 // 65536
	SETTINGS_MAX_HEADER_LIST_SIZE              = 0x6 // (infinite)
)

func (s SettingsID) String() string {
	m := map[SettingsID]string{
		0x1: "SETTINGS_HEADER_TABLE_SIZE",
		0x2: "SETTINGS_ENABLE_PUSH",
		0x3: "SETTINGS_MAX_CONCURRENT_STREAMS",
		0x4: "SETTINGS_INITIAL_WINDOW_SIZE",
		0x5: "SETTINGS_MAX_FRAME_SIZE",
		0x6: "SETTINGS_MAX_HEADER_LIST_SIZE",
	}
	return fmt.Sprintf("%s(%d)", m[s], s)
}

type SettingsFrame struct {
	*FrameHeader
	Settings map[SettingsID]int32
}

func NewSettingsFrame(flags Flag, streamID uint32, settings map[SettingsID]int32) *SettingsFrame {
	length := uint32(6 * len(settings))
	fh := NewFrameHeader(length, SettingsFrameType, flags, streamID)
	frame := &SettingsFrame{
		FrameHeader: fh,
		Settings:    settings,
	}
	return frame
}

func (frame *SettingsFrame) Read(r io.Reader) (err error) {
	frame.Settings = make(map[SettingsID]int32)

	for niv := frame.Length / 6; niv > 0; niv-- {
		var settingsID SettingsID
		var value int32

		err = binary.Read(r, binary.BigEndian, &settingsID)
		if err != nil {
			return err
		}
		err = binary.Read(r, binary.BigEndian, &value)
		if err != nil {
			return err
		}
		frame.Settings[settingsID] = value
	}
	return err
}

func (frame *SettingsFrame) Write(w io.Writer) (err error) {
	frame.FrameHeader.Write(w)
	for settingsID, value := range frame.Settings {
		err = binary.Write(w, binary.BigEndian, &settingsID)
		if err != nil {
			return err
		}
		err = binary.Write(w, binary.BigEndian, &value)
		if err != nil {
			return err
		}
	}
	return err
}

func (frame *SettingsFrame) Header() *FrameHeader {
	return frame.FrameHeader
}

func (frame *SettingsFrame) String() string {
	str := Cyan("SETTINGS")
	str += frame.FrameHeader.String()
	if frame.Flags == ACK {
		str += "\n; ACK"
	}
	str += fmt.Sprintf("\n(niv=%v)", len(frame.Settings))
	for settingsID, value := range frame.Settings {
		str += fmt.Sprintf("\n[%v:%v]", settingsID.String(), value)
	}
	return str
}

// PUSH_PROMISE
//
// +---------------+
// |Pad Length? (8)|
// +-+-------------+-----------------------------------------------+
// |R|                  Promised Stream ID (31)                    |
// +-+-----------------------------+-------------------------------+
// |                   Header Block Fragment (*)                 ...
// +---------------------------------------------------------------+
// |                           Padding (*)                       ...
// +---------------------------------------------------------------+
type PushPromiseFrame struct {
	*FrameHeader
	PadLength           uint8
	PromisedStreamID    uint32
	HeaderBlockFragment []byte
	Padding             []byte
}

func NewPushPromiseFrame(flags Flag, streamID, promisedStreamID uint32, headerBlockFragment, padding []byte) *PushPromiseFrame {
	var padded bool = flags&PADDED == PADDED
	length := 4 + len(headerBlockFragment)

	if padded {
		length = length + len(padding) + 1
	}

	fh := NewFrameHeader(uint32(length), PushPromiseFrameType, flags, streamID)
	frame := &PushPromiseFrame{
		FrameHeader:         fh,
		PadLength:           uint8(len(padding)),
		PromisedStreamID:    promisedStreamID,
		HeaderBlockFragment: headerBlockFragment,
		Padding:             padding,
	}
	return frame
}

func (frame *PushPromiseFrame) Read(r io.Reader) (err error) {
	var frameLen uint32 = frame.Length
	var padded bool = frame.Flags&PADDED == PADDED

	if padded {
		// read 8 bit for padding length
		err = binary.Read(r, binary.BigEndian, &frame.PadLength)
		if err != nil {
			return err
		}
		frameLen = frameLen - 1 // (remove pad length)
	}

	// read promised stream id
	err = binary.Read(r, binary.BigEndian, &frame.PromisedStreamID)
	if err != nil {
		return err
	}
	frameLen = frameLen - 4 // remove promised stream id length

	// read frame length bit for data
	data := make([]byte, frameLen)
	err = binary.Read(r, binary.BigEndian, &data)
	if err != nil {
		return err
	}
	if padded {
		// data + padding
		boundary := len(data) - int(frame.PadLength)
		frame.HeaderBlockFragment = data[:boundary]
		frame.Padding = data[boundary:]
	} else {
		// data only
		frame.HeaderBlockFragment = data
	}

	return err
}

func (frame *PushPromiseFrame) Write(w io.Writer) (err error) {
	err = frame.FrameHeader.Write(w)
	if err != nil {
		return err
	}

	var padded bool = frame.Flags&PADDED == PADDED

	if padded {
		// write padding length
		err = binary.Write(w, binary.BigEndian, &frame.PadLength)
		if err != nil {
			return err
		}
	}

	// write Promised Stream ID
	err = binary.Write(w, binary.BigEndian, &frame.PromisedStreamID)
	if err != nil {
		return err
	}

	// write Header Block Fragmetn
	err = binary.Write(w, binary.BigEndian, &frame.HeaderBlockFragment)
	if err != nil {
		return err
	}

	if padded {
		// write padding data
		err = binary.Write(w, binary.BigEndian, &frame.Padding)
		if err != nil {
			return err
		}
	}
	return err
}

func (frame *PushPromiseFrame) Header() *FrameHeader {
	return frame.FrameHeader
}

func (frame *PushPromiseFrame) String() string {
	str := Cyan("PUSH_PROMISE")
	str += frame.FrameHeader.String()

	str += fmt.Sprintf("\npromised streamid=%x", frame.PromisedStreamID)
	// Print first 8 byte of HeaderBlockFragment or all
	window := len(frame.HeaderBlockFragment)
	if window == 0 {
		// no data do nothing
		return str
	} else if window > 32 {
		// trim to 32 byte
		window = 32
	}
	str += fmt.Sprintf("\n%q...", string(frame.HeaderBlockFragment[:window]))
	return str
}

// PING
//
// +---------------------------------------------------------------+
// |                                                               |
// |                      Opaque Data (64)                         |
// |                                                               |
// +---------------------------------------------------------------+
type PingFrame struct {
	*FrameHeader
	OpaqueData []byte
}

func NewPingFrame(flags Flag, streamID uint32, opaqueData []byte) *PingFrame {
	var length uint32 = 8
	fh := NewFrameHeader(length, PingFrameType, flags, streamID)
	frame := &PingFrame{
		FrameHeader: fh,
		OpaqueData:  opaqueData,
	}
	return frame
}

func (frame *PingFrame) Read(r io.Reader) (err error) {
	if frame.Length != 8 {
		Trace("invalid length: %v", frame.Length)
		return fmt.Errorf("invalid length: %v", frame.Length)
	}

	frame.OpaqueData = make([]byte, 8)
	err = binary.Read(r, binary.BigEndian, &frame.OpaqueData)
	if err != nil {
		return err
	}
	return err
}

func (frame *PingFrame) Write(w io.Writer) (err error) {
	err = frame.FrameHeader.Write(w)
	if err != nil {
		return err
	}

	err = binary.Write(w, binary.BigEndian, &frame.OpaqueData)
	if err != nil {
		return err
	}
	return err
}

func (frame *PingFrame) Header() *FrameHeader {
	return frame.FrameHeader
}

func (frame *PingFrame) String() string {
	str := Cyan("PING")
	str += frame.FrameHeader.String()
	str += fmt.Sprintf("\nopaque_data=%x", frame.OpaqueData)
	return str
}

// GOAWAY
//
// +-+-------------------------------------------------------------+
// |R|                  Last-Stream-ID (31)                        |
// +-+-------------------------------------------------------------+
// |                      Error Code (32)                          |
// +---------------------------------------------------------------+
// |                  Additional Debug Data (*)                    |
// +---------------------------------------------------------------+
type GoAwayFrame struct {
	*FrameHeader
	LastStreamID        uint32
	ErrorCode           ErrorCode
	AdditionalDebugData []byte
}

func NewGoAwayFrame(streamID uint32, lastStreamID uint32, errorCode ErrorCode, additionalDebugData []byte) *GoAwayFrame {
	var length = 8 + len(additionalDebugData)
	fh := NewFrameHeader(uint32(length), GoAwayFrameType, UNSET, streamID)

	frame := &GoAwayFrame{
		FrameHeader:         fh,
		LastStreamID:        lastStreamID,
		ErrorCode:           errorCode,
		AdditionalDebugData: additionalDebugData,
	}

	return frame
}

func (frame *GoAwayFrame) Read(r io.Reader) (err error) {
	err = binary.Read(r, binary.BigEndian, &frame.LastStreamID)
	if err != nil {
		return err
	}
	err = binary.Read(r, binary.BigEndian, &frame.ErrorCode)
	if err != nil {
		return err
	}

	if frame.Length > 8 {
		additionalLength := frame.Length - 8
		frame.AdditionalDebugData = make([]byte, additionalLength)
		err = binary.Read(r, binary.BigEndian, &frame.AdditionalDebugData)
		if err != nil {
			return err
		}
	}
	return err
}

func (frame *GoAwayFrame) Write(w io.Writer) (err error) {
	frame.FrameHeader.Write(w)
	err = binary.Write(w, binary.BigEndian, &frame.LastStreamID)
	if err != nil {
		return err
	}
	err = binary.Write(w, binary.BigEndian, &frame.ErrorCode)
	if err != nil {
		return err
	}
	err = binary.Write(w, binary.BigEndian, &frame.AdditionalDebugData)
	if err != nil {
		return err
	}
	return err
}

func (frame *GoAwayFrame) Header() *FrameHeader {
	return frame.FrameHeader
}

func (frame *GoAwayFrame) String() string {
	str := Cyan("GOAWAY")
	str += frame.FrameHeader.String()
	str += fmt.Sprintf("\n(last_stream_id=%d, error_code=%s(%d), opaque_data(%q))",
		frame.LastStreamID, Red(frame.ErrorCode.String()), frame.ErrorCode, frame.AdditionalDebugData)
	return str
}

// WINDOW_UPDATE
//
// +-+-------------------------------------------------------------+
// |R|              Window Size Increment (31)                     |
// +-+-------------------------------------------------------------+
type WindowUpdateFrame struct {
	*FrameHeader
	WindowSizeIncrement uint32
}

func NewWindowUpdateFrame(streamID, incrementSize uint32) *WindowUpdateFrame {
	var length uint32 = 4

	fh := NewFrameHeader(length, WindowUpdateFrameType, UNSET, streamID)
	frame := &WindowUpdateFrame{
		FrameHeader:         fh,
		WindowSizeIncrement: incrementSize,
	}
	return frame
}

func (frame *WindowUpdateFrame) Read(r io.Reader) (err error) {
	err = binary.Read(r, binary.BigEndian, &frame.WindowSizeIncrement)
	if err != nil {
		return err
	}
	return err
}

func (frame *WindowUpdateFrame) Write(w io.Writer) (err error) {
	err = frame.FrameHeader.Write(w)
	if err != nil {
		return err
	}

	err = binary.Write(w, binary.BigEndian, &frame.WindowSizeIncrement)
	if err != nil {
		return err
	}
	return err
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
// +---------------------------------------------------------------+
// |                   Header Block Fragment (*)                 ...
// +---------------------------------------------------------------+
type ContinuationFrame struct {
	*FrameHeader
	Headers             http.Header
	HeaderBlockFragment []byte
}

func NewContinuationFrame(flags Flag, streamID uint32, headerBlockFragment []byte) *ContinuationFrame {
	length := len(headerBlockFragment)

	fh := NewFrameHeader(uint32(length), ContinuationFrameType, flags, streamID)
	frame := &ContinuationFrame{
		FrameHeader:         fh,
		HeaderBlockFragment: headerBlockFragment,
	}
	return frame
}

func (frame *ContinuationFrame) Read(r io.Reader) (err error) {
	frame.HeaderBlockFragment = make([]byte, frame.Length)
	err = binary.Read(r, binary.BigEndian, &frame.HeaderBlockFragment)
	if err != nil {
		return err
	}
	return err
}

func (frame *ContinuationFrame) Write(w io.Writer) (err error) {
	frame.FrameHeader.Write(w)
	err = binary.Write(w, binary.BigEndian, &frame.HeaderBlockFragment)
	if err != nil {
		return err
	}
	return err
}

func (frame *ContinuationFrame) Header() *FrameHeader {
	return frame.FrameHeader
}

func (frame *ContinuationFrame) String() string {
	str := Cyan("CONTINUATION")
	str += frame.FrameHeader.String()
	str += fmt.Sprintf("\n(=%x)", frame.HeaderBlockFragment)
	return str
}

// Read
func ReadFrame(r io.Reader, settings map[SettingsID]int32) (frame Frame, err error) {
	fh := new(FrameHeader)
	fh.MaxFrameSize = settings[SETTINGS_MAX_FRAME_SIZE]
	fh.MaxHeaderListSize = settings[SETTINGS_MAX_HEADER_LIST_SIZE]

	err = fh.Read(r)
	if err != nil {
		Error("%v", err)
		return nil, err
	}

	newframe, ok := FrameMap[fh.Type]
	if !ok {
		return nil, fmt.Errorf("unknown type: %v", fh.Type)
	}

	frame = newframe(fh)
	err = frame.Read(r)
	if err != nil {
		return nil, err
	}

	return frame, nil
}
