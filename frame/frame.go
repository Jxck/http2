package frame

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	. "github.com/jxck/color"
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
	PushPrimiseFrameType        = 0x5
	PingFrameType               = 0x6
	GoAwayFrameType             = 0x7
	WindowUpdateFrameType       = 0x8
	ContinuationFrameType       = 0x9
)

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

func (e ErrorCode) Format() string {
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
	END_SEGMENT       = 0x2
	END_HEADERS       = 0x4
	PRIORITY          = 0x8
	PAD_LOW           = 0x10
	PAD_HIGH          = 0x20
)

type Frame interface {
	Write(w io.Writer)
	Read(r io.Reader)
	Header() *FrameHeader
	Format() string
}

// Frame Header
//
//  0                   1                   2                   3
//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// | R |     Length (14)           |   Type (8)    |   Flags (8)   |
// +-+-+-----------+---------------+-------------------------------+
// |R|                 Stream Identifier (31)                      |
// +-+-------------------------------------------------------------+
// |                   Frame Payload (0...)                      ...
// +---------------------------------------------------------------+
type FrameHeader struct {
	Length   uint16
	Type     uint8
	Flags    uint8
	StreamId uint32
}

func NewFrameHeader(length uint16, types uint8, flags uint8, streamid uint32) *FrameHeader {
	fh := &FrameHeader{
		Length:   length,
		Type:     types,
		Flags:    flags,
		StreamId: streamid,
	}
	return fh
}

func (fh *FrameHeader) Read(r io.Reader) error {
	return binary.Read(r, binary.BigEndian, fh) // err
}

func (fh *FrameHeader) Write(w io.Writer) {
	binary.Write(w, binary.BigEndian, fh) // err
}

func (fh *FrameHeader) Format() string {
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
// | [Pad High(8)] | [Pad Low (8)] |          Data (*)             .
// +---------------+---------------+-------------------------------+
// .                            Data (*)                         ...
// +---------------------------------------------------------------+
// |                           Padding (*)                       ...
// +---------------------------------------------------------------+
type DataFrame struct {
	*FrameHeader
	Data []byte
}

func NewDataFrame(flags uint8, streamId uint32) *DataFrame {
	var length uint16 = 0
	fh := NewFrameHeader(length, DataFrameType, flags, streamId)

	dataFrame := &DataFrame{
		FrameHeader: fh,
	}

	return dataFrame
}

func (frame *DataFrame) Read(r io.Reader) {
	var frameLen, padLen uint16

	frameLen = frame.Length
	if (frame.Flags&PAD_LOW == 1) && (frame.Flags&PAD_HIGH == 1) {
		var padHigh, padLow uint8
		binary.Read(r, binary.BigEndian, &padHigh) // err
		binary.Read(r, binary.BigEndian, &padLow)  // err
		padLen = uint16(padHigh)*256 + uint16(padLow)
		frameLen = frameLen - 2 // (Pad Higth + Pad Low)
	}
	data := make([]byte, frameLen)
	binary.Read(r, binary.BigEndian, &data)   // err
	frame.Data = data[:len(data)-int(padLen)] // remove padding
}

func (frame *DataFrame) Write(w io.Writer) {
	frame.FrameHeader.Write(w)
	binary.Write(w, binary.BigEndian, &frame.Data) // err
}

func (frame *DataFrame) Header() *FrameHeader {
	return frame.FrameHeader
}

func (frame *DataFrame) Format() string {
	str := Cyan("DATA")
	str += frame.FrameHeader.Format()

	if frame.Flags&0x1 == 1 {
		str += "\n; END_STREAM"
	}

	// Print first 8 byte of Data or all
	window := len(frame.Data)
	if window == 0 {
		// no data do nothing
		return str
	} else if window > 8 {
		// trim to 8 byte
		window = 8
	}
	str += fmt.Sprintf("\n%q...", string(frame.Data[:window]))
	return str
}

// HEADERS
//
//  0                   1                   2                   3
//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// | [Pad High(8)] | [Pad Low (8)] |X|      [Priority (31)]      ...
// +---------------+---------------+-+-----------------------------+
// ...[Priority]                   | Header Block Fragment (*)   ...
// +-------------------------------+-------------------------------+
// |                   Header Block Fragment (*)                 ...
// +---------------------------------------------------------------+
// |                           Padding (*)                       ...
// +---------------------------------------------------------------+
type HeadersFrame struct {
	*FrameHeader
	Priority    uint32
	HeaderBlock []byte
	Headers     http.Header
}

func NewHeadersFrame(flags uint8, streamId uint32) *HeadersFrame {
	var length uint16 = 0
	fh := NewFrameHeader(length, HeadersFrameType, flags, streamId)

	headersFrame := &HeadersFrame{
		FrameHeader: fh,
	}

	return headersFrame
}

func (frame *HeadersFrame) Read(r io.Reader) {
	var frameLen, padLen uint16

	frameLen = frame.Length
	if (frame.Flags&PAD_LOW == PAD_LOW) && (frame.Flags&PAD_HIGH == PAD_HIGH) {
		var padHigh, padLow uint8
		binary.Read(r, binary.BigEndian, &padHigh) // err
		binary.Read(r, binary.BigEndian, &padLow)  // err
		padLen = uint16(padHigh)*256 + uint16(padLow)
		frameLen = frameLen - 2 // (Pad Higth + Pad Low)
	}

	if frame.Flags&PRIORITY == PRIORITY {
		binary.Read(r, binary.BigEndian, &frame.Priority) // err
		frameLen = frameLen - 4
	}
	data := make([]byte, frameLen)
	binary.Read(r, binary.BigEndian, &data)          // err
	frame.HeaderBlock = data[:len(data)-int(padLen)] // remove padding
}

func (frame *HeadersFrame) Write(w io.Writer) {
	frame.FrameHeader.Write(w)
	if frame.Flags&PRIORITY == PRIORITY {
		binary.Write(w, binary.BigEndian, &frame.Priority) // err
	}
	binary.Write(w, binary.BigEndian, &frame.HeaderBlock) // err
}

func (frame *HeadersFrame) Header() *FrameHeader {
	return frame.FrameHeader
}

func (frame *HeadersFrame) Format() string {
	str := Cyan("HEADERS")
	str += frame.FrameHeader.Format()

	if frame.Flags&END_STREAM == END_STREAM {
		str += "\n; END_STREAM"
	}

	if frame.Flags&END_HEADERS == END_HEADERS {
		str += "\n; END_HEADERS"
	}

	if frame.Flags&PRIORITY == PRIORITY {
		str += "\n; PRIORITY"
	}

	if frame.Flags&PAD_LOW == PAD_LOW {
		str += "\n; PAD_LOW"
	}

	if frame.Flags&PAD_HIGH == PAD_HIGH {
		str += "\n; PAD_HIGH"
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

type RstStreamFrame struct {
	*FrameHeader
	ErrorCode ErrorCode
}

func NewRstStreamFrame(errorCode ErrorCode, streamId uint32) *RstStreamFrame {
	var length uint16 = 4
	var flags uint8 = 0

	fh := NewFrameHeader(length, RstStreamFrameType, flags, streamId)
	frame := &RstStreamFrame{
		FrameHeader: fh,
		ErrorCode:   errorCode,
	}
	return frame
}

func (frame *RstStreamFrame) Read(r io.Reader) {
	binary.Read(r, binary.BigEndian, &frame.ErrorCode) // err
}

func (frame *RstStreamFrame) Write(w io.Writer) {
	frame.FrameHeader.Write(w)
	binary.Write(w, binary.BigEndian, &frame.ErrorCode) // err
}

func (frame *RstStreamFrame) Header() *FrameHeader {
	return frame.FrameHeader
}

func (frame *RstStreamFrame) Format() string {
	str := Cyan("RST_STREAM")
	str += frame.FrameHeader.Format()
	str += fmt.Sprintf("\n(Error Code=%d)", frame.ErrorCode)
	return str
}

// SETTINGS Frame
//
//  0                   1                   2                   3
//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// |Identifier (8) |                 Value (32)                  ...
// +---------------+-----------------------------------------------+
// ...Value        |
// +---------------+

const DEFAULT_WINDOW_SIZE uint32 = 65535

type SettingsId uint8

const (
	SETTINGS_HEADER_TABLE_SIZE      SettingsId = 1
	SETTINGS_ENABLE_PUSH                       = 2
	SETTINGS_MAX_CONCURRENT_STREAMS            = 3
	SETTINGS_INITIAL_WINDOW_SIZE               = 4
)

func (s SettingsId) Format() string {
	m := map[SettingsId]string{
		1: "SETTINGS_HEADER_TABLE_SIZE",
		2: "SETTINGS_ENABLE_PUSH",
		3: "SETTINGS_MAX_CONCURRENT_STREAMS",
		4: "SETTINGS_INITIAL_WINDOW_SIZE",
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

// TODO: flags
func NewSettingsFrame(flags uint8, setting map[SettingsId]uint32, streamId uint32) *SettingsFrame {
	var settings []Setting
	for id, val := range setting {
		s := Setting{
			SettingsId: id,
			Value:      val,
		}
		settings = append(settings, s)
	}

	var length uint16 = uint16(5 * len(settings))
	fh := NewFrameHeader(length, SettingsFrameType, flags, streamId)
	frame := &SettingsFrame{
		FrameHeader: fh,
		Settings:    settings,
	}
	return frame
}

func (frame *SettingsFrame) Read(r io.Reader) {
	for niv := frame.Length / 5; niv > 0; niv-- {
		s := *new(Setting)

		binary.Read(r, binary.BigEndian, &s.SettingsId) // err
		binary.Read(r, binary.BigEndian, &s.Value)      // err
		frame.Settings = append(frame.Settings, s)
	}
}

func (frame *SettingsFrame) Write(w io.Writer) {
	frame.FrameHeader.Write(w)
	for _, setting := range frame.Settings {
		binary.Write(w, binary.BigEndian, &setting.SettingsId) // err
		binary.Write(w, binary.BigEndian, &setting.Value)      // err
	}
}

// TODO: fixme
func (frame *SettingsFrame) PayloadBase64URL() string {
	buf := bytes.NewBuffer(make([]byte, 0))
	frame.Write(buf)
	str := base64.URLEncoding.EncodeToString(buf.Bytes()[8:])
	str = strings.Replace(str, "=", "", -1)
	return str
}

func (frame *SettingsFrame) Header() *FrameHeader {
	return frame.FrameHeader
}

func (frame *SettingsFrame) Format() string {
	str := Cyan("SETTINGS")
	str += frame.FrameHeader.Format()
	if frame.Flags == 1 {
		str += "\n; ACK"
	}
	str += fmt.Sprintf("\n(niv=%v)", len(frame.Settings))
	for _, s := range frame.Settings {
		str += fmt.Sprintf("\n[%v:%v]", s.SettingsId.Format(), s.Value)
	}
	return str
}

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

type GoAwayFrame struct {
	*FrameHeader
	LastStreamID        uint32
	ErrorCode           ErrorCode
	AdditionalDebugData []byte // unsupported
}

func NewGoAwayFrame(lastStreamId uint32, errorCode ErrorCode, streamId uint32) *GoAwayFrame {
	var length uint16 = 8
	var flags uint8 = 0x00
	fh := NewFrameHeader(length, GoAwayFrameType, flags, streamId)
	frame := &GoAwayFrame{
		FrameHeader:  fh,
		LastStreamID: lastStreamId,
		ErrorCode:    errorCode,
	}

	return frame
}

func (frame *GoAwayFrame) Read(r io.Reader) {
	binary.Read(r, binary.BigEndian, &frame.LastStreamID)        // err
	binary.Read(r, binary.BigEndian, &frame.ErrorCode)           // err
	binary.Read(r, binary.BigEndian, &frame.AdditionalDebugData) // err
}

func (frame *GoAwayFrame) Write(w io.Writer) {
	frame.FrameHeader.Write(w)
	binary.Write(w, binary.BigEndian, &frame.LastStreamID)        // err
	binary.Write(w, binary.BigEndian, &frame.ErrorCode)           // err
	binary.Write(w, binary.BigEndian, &frame.AdditionalDebugData) // err
}

func (frame *GoAwayFrame) Header() *FrameHeader {
	return frame.FrameHeader
}

func (frame *GoAwayFrame) Format() string {
	str := Cyan("GOAWAY")
	str += frame.FrameHeader.Format()
	str += fmt.Sprintf("\n(last_stream_id=%d, error_code=%s, opaque_data(unsupported))",
		frame.LastStreamID, Red(frame.ErrorCode.Format()))
	return str
}

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

func NewWindowUpdateFrame(incrementSize, streamId uint32) *WindowUpdateFrame {
	var length uint16 = 4

	// TODO: temp flag
	var flags uint8 = 0

	fh := NewFrameHeader(length, WindowUpdateFrameType, flags, streamId)
	frame := &WindowUpdateFrame{
		FrameHeader:         fh,
		WindowSizeIncrement: incrementSize,
	}
	return frame
}

func (frame *WindowUpdateFrame) Read(r io.Reader) {
	binary.Read(r, binary.BigEndian, &frame.WindowSizeIncrement) // err
}

func (frame *WindowUpdateFrame) Write(w io.Writer) {
	frame.FrameHeader.Write(w)
	binary.Write(w, binary.BigEndian, &frame.WindowSizeIncrement) // err
}

func (frame *WindowUpdateFrame) Header() *FrameHeader {
	return frame.FrameHeader
}

func (frame *WindowUpdateFrame) Format() string {
	str := Cyan("WINDOW_UPDATE")
	str += frame.FrameHeader.Format()
	str += fmt.Sprintf("\n(window_size_increment=%d)", frame.WindowSizeIncrement)
	return str
}

// CONTINUATION
//
//  0                   1                   2                   3
//  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
// +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
// | [Pad High(8)] | [Pad Low (8)] |  Header Block Fragment (*)    .
// +---------------+---------------+-------------------------------+
// |                   Header Block Fragment (*)                 ...
// +---------------------------------------------------------------+
// |                           Padding (*)                       ...
// +---------------------------------------------------------------+
