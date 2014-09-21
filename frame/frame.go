package frame

import (
	"fmt"
	. "github.com/Jxck/color"
	"io"
	"log"
	"net/http"
	"strings"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

const (
	DataFrameType         uint8 = 0x0
	HeadersFrameType            = 0x1
	PriorityFrameType           = 0x2
	RstStreamFrameType          = 0x3
	SettingsFrameType           = 0x4
	PushPromiseFrameType        = 0x5
	PingFrameType               = 0x6
	GoAwayFrameType             = 0x7
	WindowUpdateFrameType       = 0x8
	ContinuationFrameType       = 0x9
)

func FrameName(i uint8) string {
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
	return names[i]
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
const (
	UNSET       uint8 = 0x0
	END_STREAM        = 0x1
	ACK               = 0x1 // for settings
	END_HEADERS       = 0x4
	PADDED            = 0x8
	PRIORITY          = 0x20
)

type Frame interface {
	Write(w io.Writer) error
	Read(r io.Reader) error
	Header() *FrameHeader
	String() string
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
	Type     uint8
	Flags    uint8
	StreamId uint32 // R+31bit
}

func NewFrameHeader(length uint32, types uint8, flags uint8, streamid uint32) *FrameHeader {
	fh := &FrameHeader{
		Length:   length,
		Type:     types,
		Flags:    flags,
		StreamId: streamid,
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
	fh.Type = uint8(first & 0xFF)
	// first 24 bit for length
	fh.Length = first >> 8

	// read 8 bit for Flags
	MustRead(r, &fh.Flags)

	// read 32 bit for StreamId
	var last uint32
	MustRead(r, &last)
	fh.StreamId = last & 127

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
	MustWrite(w, &fh.StreamId)

	return err
}

func (fh *FrameHeader) String() string {
	str := fmt.Sprintf(
		" frame <length=%v, flags=%#x, stream_id=%v>",
		fh.Length, fh.Flags, fh.StreamId,
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

func NewDataFrame(flags uint8, streamId uint32) *DataFrame {
	var length uint32 = 0
	fh := NewFrameHeader(length, DataFrameType, flags, streamId)

	dataFrame := &DataFrame{
		FrameHeader: fh,
	}

	return dataFrame
}

func (frame *DataFrame) Read(r io.Reader) (err error) {
	defer func() {
		err = Recovery(recover())
	}()

	var frameLen uint32 = frame.Length
	var padding bool = frame.Flags&PADDED == PADDED

	if padding {
		// read 8 bit for padding length
		MustRead(r, &frame.PadLength)
		frameLen = frameLen - 1 // (remove pad length)
	}

	// read frame length bit for data
	data := make([]byte, frameLen)
	MustRead(r, &data)

	if padding {
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

	var padding bool = frame.Flags&PADDED == PADDED

	if padding {
		// write padding length
		MustWrite(w, &frame.PadLength)
	}

	// write data
	MustWrite(w, &frame.Data)

	if padding {
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
	PadLength        uint8
	Exclusive        bool
	StreamDependency uint32
	Weight           uint8
	HeaderBlock      []byte
	Headers          http.Header
	Padding          []byte
}

func NewHeadersFrame(flags uint8, streamId uint32) *HeadersFrame {
	var length uint32 = 0
	fh := NewFrameHeader(length, HeadersFrameType, flags, streamId)

	headersFrame := &HeadersFrame{
		FrameHeader: fh,
	}

	return headersFrame
}

func (frame *HeadersFrame) Read(r io.Reader) (err error) {
	defer func() {
		err = Recovery(recover())
	}()

	var frameLen uint32 = frame.Length
	var padding bool = frame.Flags&PADDED == PADDED
	var priority bool = frame.Flags&PRIORITY == PRIORITY

	if padding {
		MustRead(r, &frame.PadLength)
		frameLen = frameLen - 1 // remove pad length
	}

	if priority {
		// read 32bit for e + streamdependency
		var u32 uint32
		MustRead(r, &u32)

		if u32&0x80000000 == 0x80000000 {
			frame.Exclusive = true
		}
		frame.StreamDependency = u32 & 0x7FFFFFFF

		frameLen = frameLen - 4 // remove stream dependency length

		MustRead(r, &frame.Weight)

		// add 1 for weight
		frame.Weight = frame.Weight + 1
		frameLen = frameLen - 1 // remove weight length
	}

	// read frame length bit for data
	data := make([]byte, frameLen)
	MustRead(r, &data)

	if padding {
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

	// TODO: support padding
	err = frame.FrameHeader.Write(w)
	if err != nil {
		return err
	}

	var padding bool = frame.Flags&PADDED == PADDED
	var priority bool = frame.Flags&PRIORITY == PRIORITY

	if padding {
		MustWrite(w, &frame.PadLength)
	}

	if priority {
		streamDependency := frame.StreamDependency
		if frame.Exclusive {
			streamDependency = streamDependency + 0x80000000
		}
		MustWrite(w, &streamDependency)

		weight := frame.Weight - 1
		MustWrite(w, &weight)
	}
	MustWrite(w, &frame.HeaderBlock)

	if padding {
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

	for name, value := range frame.Headers {
		str += fmt.Sprintf("\n%s: %s", Blue(name), strings.Join(value, ","))
	}

	return str
}

// PRIORITY
//
// 0                   1                   2                   3
//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |E|                  Stream Dependency (31)                     |
// +-+-------------+-----------------------------------------------+
// |   Weight (8)  |
// +-+-------------+
//
//
// RST_STREAM
//
// 0                   1                   2                   3
// 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |                        Error Code (32)                        |
// +---------------------------------------------------------------+
type RstStreamFrame struct {
	*FrameHeader
	ErrorCode ErrorCode
}

func NewRstStreamFrame(errorCode ErrorCode, streamId uint32) *RstStreamFrame {
	var length uint32 = 4
	var flags uint8 = 0

	fh := NewFrameHeader(length, RstStreamFrameType, flags, streamId)
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
const DEFAULT_WINDOW_SIZE uint32 = 65535

type SettingsId uint16

const (
	SETTINGS_HEADER_TABLE_SIZE      SettingsId = 1 // 4096
	SETTINGS_ENABLE_PUSH                       = 2 // 1
	SETTINGS_MAX_CONCURRENT_STREAMS            = 3 // (infinite)
	SETTINGS_INITIAL_WINDOW_SIZE               = 4 // 65535
	SETTINGS_MAX_FRAME_SIZE                    = 5 // 65536
	SETTINGS_MAX_HEADER_LIST_SIZE              = 6 // (infinite)
)

func (s SettingsId) String() string {
	m := map[SettingsId]string{
		1: "SETTINGS_HEADER_TABLE_SIZE",
		2: "SETTINGS_ENABLE_PUSH",
		3: "SETTINGS_MAX_CONCURRENT_STREAMS",
		4: "SETTINGS_INITIAL_WINDOW_SIZE",
		5: "SETTINGS_MAX_FRAME_SIZE",
		6: "SETTINGS_MAX_HEADER_LIST_SIZE",
	}
	return fmt.Sprintf("%s(%d)", m[s], s)
}

type Setting struct {
	SettingsId SettingsId
	Value      uint32
}

type SettingsFrame struct {
	*FrameHeader
	Settings []Setting
}

func NewSettingsFrame(flags uint8, settings []Setting, streamId uint32) *SettingsFrame {
	var length uint32 = uint32(6 * len(settings))
	fh := NewFrameHeader(length, SettingsFrameType, flags, streamId)
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

	for niv := frame.Length / 6; niv > 0; niv-- {
		s := *new(Setting)

		MustRead(r, &s.SettingsId)
		MustRead(r, &s.Value)
		frame.Settings = append(frame.Settings, s)
	}
	return err
}

func (frame *SettingsFrame) Write(w io.Writer) (err error) {
	defer func() {
		err = Recovery(recover())
	}()

	frame.FrameHeader.Write(w)
	for _, setting := range frame.Settings {
		MustWrite(w, &setting.SettingsId)
		MustWrite(w, &setting.Value)
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
	for _, s := range frame.Settings {
		str += fmt.Sprintf("\n[%v:%v]", s.SettingsId.String(), s.Value)
	}
	return str
}

// PUSH_PROMISE
//
// 0                   1                   2                   3
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
	AdditionalDebugData []byte // unsupported
}

func NewGoAwayFrame(lastStreamId uint32, errorCode ErrorCode, streamId uint32) *GoAwayFrame {
	var length uint32 = 8
	var flags uint8 = 0x00
	fh := NewFrameHeader(length, GoAwayFrameType, flags, streamId)
	frame := &GoAwayFrame{
		FrameHeader:  fh,
		LastStreamID: lastStreamId,
		ErrorCode:    errorCode,
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
// 0                   1                   2                   3
// 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |R|              Window Size Increment (31)                     |
// +-+-------------------------------------------------------------+
type WindowUpdateFrame struct {
	*FrameHeader
	WindowSizeIncrement uint32
}

func NewWindowUpdateFrame(incrementSize, streamId uint32) *WindowUpdateFrame {
	var length uint32 = 4

	// TODO: temp flag
	var flags uint8 = 0

	fh := NewFrameHeader(length, WindowUpdateFrameType, flags, streamId)
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
// 0                   1                   2                   3
//   0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
//  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//  |                   Header Block Fragment (*)                 ...
//  +---------------------------------------------------------------+

// Reade
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
