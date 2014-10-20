package frame

import (
	"fmt"
	. "github.com/Jxck/color"
	"io"
	"net/http"
	"strings"
)

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
	NO_ERROR            ErrorCode = 0
	PROTOCOL_ERROR                = 1
	INTERNAL_ERROR                = 2
	FLOW_CONTROL_ERROR            = 3
	SETTINGS_TIMEOUT              = 4
	STREAM_CLOSED                 = 5
	FRAME_SIZE_ERROR              = 6
	REFUSED_STREAM                = 7
	CANCEL                        = 8
	COMPRESSION_ERROR             = 9
	CONNECT_ERROR                 = 10
	ENHANCE_YOUR_CALM             = 11
	INADEQUATE_SECURITY           = 12
)

func (e ErrorCode) String() string {
	errors := []string{
		"NO_ERROR",
		"PROTOCOL_ERROR",
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
	}
	return errors[int(e)]
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
//  0                   1                   2                   3
//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                 Length (24)                   |
// +---------------+---------------+---------------+
// |   Type (8)    |   Flags (8)   |
// +-+-+-----------+---------------+-------------------------------+
// |R|                 Stream Identifier (31)                      |
// +=+=============================================================+
// |                   Frame Payload (0...)                      ...
// +---------------------------------------------------------------+

type FrameHeader struct {
	Length   uint32 // 24bit
	Type     FrameType
	Flags    Flag
	StreamID uint32 // R+31bit
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
	defer func() {
		err = Recovery(recover())
	}()

	// read 32 bit
	var first uint32
	MustRead(r, &first)

	// last 8 bit for type
	fh.Type = FrameType(first & 0xFF)
	// first 24 bit for length
	fh.Length = first >> 8

	// read 8 bit for Flags
	MustRead(r, &fh.Flags)

	// read 32 bit for StreamID
	var last uint32
	MustRead(r, &last)
	fh.StreamID = last & 0x7FFFFFFF

	return err
}

func (fh *FrameHeader) Write(w io.Writer) (err error) {
	defer func() {
		err = Recovery(recover())
	}()

	// write length + type as 32bit
	var first uint32 = fh.Length<<8 + uint32(fh.Type)
	MustWrite(w, &first)

	// write flags
	MustWrite(w, &fh.Flags)

	// write stream id
	MustWrite(w, &fh.StreamID)

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
//  0                   1                   2                   3
//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
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
	defer func() {
		err = Recovery(recover())
	}()

	var frameLen uint32 = frame.Length
	var padded bool = frame.Flags&PADDED == PADDED

	if padded {
		// read 8 bit for padding length
		MustRead(r, &frame.PadLength)
		frameLen = frameLen - 1 // (remove pad length)
	}

	// read frame length bit for data
	data := make([]byte, frameLen)
	MustRead(r, &data)

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
	defer func() {
		err = Recovery(recover())
	}()

	err = frame.FrameHeader.Write(w)
	if err != nil {
		return err
	}

	var padded bool = frame.Flags&PADDED == PADDED

	if padded {
		// write padding length
		MustWrite(w, &frame.PadLength)
	}

	// write data
	MustWrite(w, &frame.Data)

	if padded {
		// write padding data
		MustWrite(w, &frame.Padding)
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
//  0                   1                   2                   3
//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
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
	PadLength      uint8
	DependencyTree *DependencyTree
	HeaderBlock    []byte
	Headers        http.Header
	Padding        []byte
}

type DependencyTree struct {
	Exclusive        bool
	StreamDependency uint32
	Weight           uint8
}

func NewHeadersFrame(flags Flag, streamID uint32, dependencyTree *DependencyTree, headerBlock []byte, padding []byte) *HeadersFrame {
	var padded bool = flags&PADDED == PADDED
	var priority bool = flags&PRIORITY == PRIORITY

	length := len(headerBlock)
	if padded {
		length = length + len(padding) + 1
	}

	if priority {
		length = length + 5
	}

	fh := NewFrameHeader(uint32(length), HeadersFrameType, flags, streamID)

	headersFrame := &HeadersFrame{
		FrameHeader:    fh,
		PadLength:      uint8(len(padding)),
		DependencyTree: dependencyTree,
		HeaderBlock:    headerBlock,
		Padding:        padding,
	}

	return headersFrame
}

func (frame *HeadersFrame) Read(r io.Reader) (err error) {
	defer func() {
		err = Recovery(recover())
	}()

	var frameLen uint32 = frame.Length
	var padded bool = frame.Flags&PADDED == PADDED
	var priority bool = frame.Flags&PRIORITY == PRIORITY

	if padded {
		MustRead(r, &frame.PadLength)
		frameLen = frameLen - 1 // remove pad length
	}

	if priority {
		dependencyTree := new(DependencyTree)

		// read 32bit for e + streamdependency
		var u32 uint32
		MustRead(r, &u32)

		if u32&0x80000000 == 0x80000000 {
			dependencyTree.Exclusive = true
		}
		dependencyTree.StreamDependency = u32 & 0x7FFFFFFF

		frameLen = frameLen - 4 // remove stream dependency length

		MustRead(r, &dependencyTree.Weight)

		// add 1 for weight
		dependencyTree.Weight = dependencyTree.Weight + 1
		frameLen = frameLen - 1 // remove weight length

		frame.DependencyTree = dependencyTree
	}

	// read frame length bit for data
	data := make([]byte, frameLen)
	MustRead(r, &data)

	if padded {
		// header block + padding
		boundary := len(data) - int(frame.PadLength)
		frame.HeaderBlock = data[:boundary]
		frame.Padding = data[boundary:]
	} else {
		// header block only
		frame.HeaderBlock = data
	}

	return err
}

func (frame *HeadersFrame) Write(w io.Writer) (err error) {
	defer func() {
		err = Recovery(recover())
	}()

	err = frame.FrameHeader.Write(w)
	if err != nil {
		return err
	}

	var padded bool = frame.Flags&PADDED == PADDED
	var priority bool = frame.Flags&PRIORITY == PRIORITY

	if padded {
		MustWrite(w, &frame.PadLength)
	}

	if priority {
		streamDependency := frame.DependencyTree.StreamDependency
		if frame.DependencyTree.Exclusive {
			streamDependency = streamDependency + 0x80000000
		}
		MustWrite(w, &streamDependency)

		weight := frame.DependencyTree.Weight - 1
		MustWrite(w, &weight)
	}
	MustWrite(w, &frame.HeaderBlock)

	if padded {
		MustWrite(w, &frame.Padding)
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
			str += fmt.Sprintf("\n%s: %s", Blue(name), value)
		}
	}

	for name, value := range frame.Headers {
		if strings.HasPrefix(name, ":") {
			continue
		}
		str += fmt.Sprintf("\n%s: %s", Blue(name), strings.Join(value, ","))
	}

	return str
}

// PRIORITY
//
//  0                   1                   2                   3
//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
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
	defer func() {
		err = Recovery(recover())
	}()

	var u32 uint32
	MustRead(r, &u32)

	if u32&0x80000000 == 0x80000000 {
		frame.Exclusive = true
	}
	frame.StreamDependency = u32 & 0x7FFFFFFF

	MustRead(r, &frame.Weight)
	return err
}

func (frame *PriorityFrame) Write(w io.Writer) (err error) {
	defer func() {
		err = Recovery(recover())
	}()

	frame.FrameHeader.Write(w)

	streamDependency := frame.StreamDependency
	if frame.Exclusive {
		streamDependency = streamDependency + 0x80000000
	}
	MustWrite(w, &streamDependency)

	MustWrite(w, &frame.Weight)
	return err
}

func (frame *PriorityFrame) Header() *FrameHeader {
	return frame.FrameHeader
}

func (frame *PriorityFrame) String() string {
	str := Cyan("RRIORITY")
	str += frame.FrameHeader.String()
	str += fmt.Sprintf("\n(Stream Dependency=%d, Weight)", frame.StreamDependency, frame.Weight)
	return str
}

// RST_STREAM
//
//  0                   1                   2                   3
//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
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
	defer func() {
		err = Recovery(recover())
	}()

	MustRead(r, &frame.ErrorCode)
	return err
}

func (frame *RstStreamFrame) Write(w io.Writer) (err error) {
	defer func() {
		err = Recovery(recover())
	}()

	frame.FrameHeader.Write(w)
	MustWrite(w, &frame.ErrorCode)
	return err
}

func (frame *RstStreamFrame) Header() *FrameHeader {
	return frame.FrameHeader
}

func (frame *RstStreamFrame) String() string {
	str := Cyan("RST_STREAM")
	str += frame.FrameHeader.String()
	str += fmt.Sprintf("\n(Error Code=%d)", frame.ErrorCode)
	return str
}

// SETTINGS Frame
//
//  0                   1                   2                   3
//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |       Identifier (16)         |
// +-------------------------------+-------------------------------+
// |                        Value (32)                             |
// +---------------------------------------------------------------+
const (
	DEFAULT_HEADER_TABLE_SIZE   uint32 = 4096
	DEFAULT_ENABLE_PUSH                = 1
	DEFAULT_INITIAL_WINDOW_SIZE        = 65535
	DEFAULT_MAX_FRAME_SIZE             = 16384
	// DEFAULT_MAX_CONCURRENT_STREAMS  = infinite
	// DEFAULT_MAX_HEADER_LIST_SIZE    = infinite
)

type SettingsID uint16

const (
	SETTINGS_HEADER_TABLE_SIZE      SettingsID = 1 // 4096
	SETTINGS_ENABLE_PUSH                       = 2 // 1
	SETTINGS_MAX_CONCURRENT_STREAMS            = 3 // (infinite)
	SETTINGS_INITIAL_WINDOW_SIZE               = 4 // 65535
	SETTINGS_MAX_FRAME_SIZE                    = 5 // 65536
	SETTINGS_MAX_HEADER_LIST_SIZE              = 6 // (infinite)
)

func (s SettingsID) String() string {
	m := map[SettingsID]string{
		1: "SETTINGS_HEADER_TABLE_SIZE",
		2: "SETTINGS_ENABLE_PUSH",
		3: "SETTINGS_MAX_CONCURRENT_STREAMS",
		4: "SETTINGS_INITIAL_WINDOW_SIZE",
		5: "SETTINGS_MAX_FRAME_SIZE",
		6: "SETTINGS_MAX_HEADER_LIST_SIZE",
	}
	return fmt.Sprintf("%s(%d)", m[s], s)
}

type SettingsFrame struct {
	*FrameHeader
	Settings map[SettingsID]uint32
}

func NewSettingsFrame(flags Flag, streamID uint32, settings map[SettingsID]uint32) *SettingsFrame {
	var length uint32 = uint32(6 * len(settings))
	fh := NewFrameHeader(length, SettingsFrameType, flags, streamID)
	frame := &SettingsFrame{
		FrameHeader: fh,
		Settings:    settings,
	}
	return frame
}

func (frame *SettingsFrame) Read(r io.Reader) (err error) {
	defer func() {
		err = Recovery(recover())
	}()

	frame.Settings = make(map[SettingsID]uint32)

	for niv := frame.Length / 6; niv > 0; niv-- {
		var settingsID SettingsID
		var value uint32

		MustRead(r, &settingsID)
		MustRead(r, &value)
		frame.Settings[settingsID] = value
	}
	return err
}

func (frame *SettingsFrame) Write(w io.Writer) (err error) {
	defer func() {
		err = Recovery(recover())
	}()

	frame.FrameHeader.Write(w)
	for settingsID, value := range frame.Settings {
		MustWrite(w, &settingsID)
		MustWrite(w, &value)
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
//  0                   1                   2                   3
//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
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
	defer func() {
		err = Recovery(recover())
	}()

	var frameLen uint32 = frame.Length
	var padded bool = frame.Flags&PADDED == PADDED

	if padded {
		// read 8 bit for padding length
		MustRead(r, &frame.PadLength)
		frameLen = frameLen - 1 // (remove pad length)
	}

	// read promised stream id
	MustRead(r, &frame.PromisedStreamID)
	frameLen = frameLen - 4 // remove promised stream id length

	// read frame length bit for data
	data := make([]byte, frameLen)
	MustRead(r, &data)
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
	defer func() {
		err = Recovery(recover())
	}()

	err = frame.FrameHeader.Write(w)
	if err != nil {
		return err
	}

	var padded bool = frame.Flags&PADDED == PADDED

	if padded {
		// write padding length
		MustWrite(w, &frame.PadLength)
	}

	// write Promised Stream ID
	MustWrite(w, &frame.PromisedStreamID)

	// write Header Block Fragmetn
	MustWrite(w, &frame.HeaderBlockFragment)

	if padded {
		// write padding data
		MustWrite(w, &frame.Padding)
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
//  0                   1                   2                   3
//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
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
	defer func() {
		err = Recovery(recover())
	}()

	frame.OpaqueData = make([]byte, 8)
	MustRead(r, &frame.OpaqueData)
	return err
}

func (frame *PingFrame) Write(w io.Writer) (err error) {
	defer func() {
		err = Recovery(recover())
	}()

	err = frame.FrameHeader.Write(w)
	if err != nil {
		return err
	}

	MustWrite(w, &frame.OpaqueData)
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
//  0                   1                   2                   3
//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
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
	defer func() {
		err = Recovery(recover())
	}()

	MustRead(r, &frame.LastStreamID)
	MustRead(r, &frame.ErrorCode)

	if frame.Length > 8 {
		additionalLength := frame.Length - 8
		frame.AdditionalDebugData = make([]byte, additionalLength)
		MustRead(r, &frame.AdditionalDebugData)
	}
	return err
}

func (frame *GoAwayFrame) Write(w io.Writer) (err error) {
	defer func() {
		err = Recovery(recover())
	}()

	frame.FrameHeader.Write(w)
	MustWrite(w, &frame.LastStreamID)
	MustWrite(w, &frame.ErrorCode)
	MustWrite(w, &frame.AdditionalDebugData)
	return err
}

func (frame *GoAwayFrame) Header() *FrameHeader {
	return frame.FrameHeader
}

func (frame *GoAwayFrame) String() string {
	str := Cyan("GOAWAY")
	str += frame.FrameHeader.String()
	str += fmt.Sprintf("\n(last_stream_id=%d, error_code=%s, opaque_data(%q))",
		frame.LastStreamID, Red(frame.ErrorCode.String()), frame.AdditionalDebugData)
	return str
}

// WINDOW_UPDATE
//
//  0                   1                   2                   3
//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
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
	defer func() {
		err = Recovery(recover())
	}()

	MustRead(r, &frame.WindowSizeIncrement)
	return err
}

func (frame *WindowUpdateFrame) Write(w io.Writer) (err error) {
	defer func() {
		err = Recovery(recover())
	}()

	err = frame.FrameHeader.Write(w)
	if err != nil {
		return err
	}

	MustWrite(w, &frame.WindowSizeIncrement)
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
//  0                   1                   2                   3
//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                   Header Block Fragment (*)                 ...
// +---------------------------------------------------------------+
type ContinuationFrame struct {
	*FrameHeader
	HeaderBlockFragment []byte
	// TODO: support headers encode/decode
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
	defer func() {
		err = Recovery(recover())
	}()

	frame.HeaderBlockFragment = make([]byte, frame.Length)
	MustRead(r, &frame.HeaderBlockFragment)
	return err
}

func (frame *ContinuationFrame) Write(w io.Writer) (err error) {
	defer func() {
		err = Recovery(recover())
	}()

	frame.FrameHeader.Write(w)
	MustWrite(w, &frame.HeaderBlockFragment)
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
func ReadFrame(r io.Reader) (frame Frame, err error) {
	fh := new(FrameHeader)
	err = fh.Read(r)
	if err != nil {
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
