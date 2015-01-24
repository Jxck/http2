package frame

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	assert "github.com/Jxck/assertion"
	"reflect"
	"strings"
	"testing"
	"testing/quick"
)

var (
	maxLength    int     = 0xFFFFFF
	maxPadLength int     = 0xFF
	count        float64 = 100
)

type TestCase struct {
	Error string    `json:"error"`
	Wire  string    `json:"wire"`
	Frame TestFrame `json:"frame"`
}

type TestFrame struct {
	Length   uint32          `json:"length"`
	Payload  json.RawMessage `json:"frame_payload"`
	Flags    uint8           `json:"flags"`
	StreamId uint32          `json:"stream_identifier"`
	Type     uint8           `json:"type"`
}

// Frame Header
func TestFrameHeaderQuickCheck(t *testing.T) {
	f := func(length uint32, types FrameType, flags Flag, streamId uint32) bool {
		// setup data
		length = length >> 8
		streamId = streamId >> 1

		// expected
		expected := NewFrameHeader(length, types, flags, streamId)
		buf := bytes.NewBuffer(make([]byte, 0))
		expected.Write(buf)

		// actual
		actual := new(FrameHeader)
		actual.Read(buf)

		return reflect.DeepEqual(actual, expected)
	}

	c := &quick.Config{
		MaxCountScale: count,
	}

	if err := quick.Check(f, c); err != nil {
		t.Error(err)
	}
}

// DATA Frame
func TestDataFrameQuickCheck(t *testing.T) {
	f := func(flags Flag, streamId uint32, data []byte) bool {
		// setup data
		streamId = streamId >> 1
		if len(data) > maxLength {
			data = data[:maxLength-1]
		}

		var window = len(data)
		if len(data) > maxPadLength {
			window = len(data) - maxPadLength
		}

		// expected
		buf := bytes.NewBuffer(make([]byte, 0))
		expected := NewDataFrame(flags, streamId, data[:window], data[window:])
		expected.Write(buf)

		// actual
		fh := new(FrameHeader)
		fh.Read(buf)
		actual := FrameMap[DataFrameType](fh)
		actual.Read(buf)

		return reflect.DeepEqual(actual, expected)
	}

	c := &quick.Config{
		MaxCountScale: count,
	}

	if err := quick.Check(f, c); err != nil {
		t.Error(err)
	}
}

type DataPayload struct {
	Data          string `json:"data"`
	PaddingLength uint8  `json:"padding_length"`
	Padding       string `json:"padding"`
}

func TestDataCase(t *testing.T) {
	var c TestCase
	var p DataPayload
	framecase := []byte(`{
    "error": null,
    "wire": "0000140008000000020648656C6C6F2C20776F726C6421486F77647921",
    "frame": {
      "length": 20,
      "frame_payload": {
        "data": "Hello, world!",
        "padding_length": 6,
        "padding": "Howdy!"
      },
      "flags": 8,
      "stream_identifier": 2,
      "type": 0
    },
    "draft": 14,
    "description": "noraml data frame"
  }`)

	// ignore error
	json.Unmarshal(framecase, &c)
	json.Unmarshal(c.Frame.Payload, &p)

	// trace data
	wire := c.Wire
	flags := Flag(c.Frame.Flags)
	streamId := c.Frame.StreamId
	data := []byte(p.Data)
	padding := []byte(p.Padding)

	// compare struct
	expected := NewDataFrame(flags, streamId, data, padding)
	actual, err := ReadFrame(hexToBuffer(wire))
	if err != nil {
		t.Fatal(err)
	}
	_ = actual
	assert.Equal(t, actual, expected)

	// compare wire
	buf := bytes.NewBuffer(make([]byte, 0))
	expected.Write(buf)
	hexdump := strings.ToUpper(hex.EncodeToString(buf.Bytes()))
	assert.Equal(t, wire, hexdump)
}

// HEADERS Frame
func TestHeadersFrame(t *testing.T) {
	hb := []byte("test header block")
	expected := NewHeadersFrame(END_STREAM, 2, nil, hb, nil)

	buf := bytes.NewBuffer(make([]byte, 0))
	expected.Write(buf)

	fh := new(FrameHeader)
	fh.Read(buf)

	actual := new(HeadersFrame)
	actual.FrameHeader = fh
	actual.Read(buf)

	assert.Equal(t, actual, expected)
}

func TestHeadersPriorityFrame(t *testing.T) {
	hb := []byte("test header block")
	dt := &DependencyTree{
		Exclusive:        true,
		StreamDependency: 1,
		Weight:           2,
	}
	expected := NewHeadersFrame(PRIORITY, 2, dt, hb, nil)

	buf := bytes.NewBuffer(make([]byte, 0))
	expected.Write(buf)

	fh := new(FrameHeader)
	fh.Read(buf)

	actual := new(HeadersFrame)
	actual.FrameHeader = fh
	actual.Read(buf)

	assert.Equal(t, actual, expected)
}

type HeadersPayload struct {
	HeaderBlockFragment string `json:"header_block_fragment"`
	Padding             string `json:"padding"`
	PaddingLength       uint8  `json:"padding_length"`
	Priority            uint8  `json:"priority"`
}

func TestHeadersCase(t *testing.T) {
	var c TestCase
	var p HeadersPayload
	framecase := []byte(`{
    "error": null,
    "wire": "00000D010000000001746869732069732064756D6D79",
    "frame": {
      "length": 13,
      "frame_payload": {
        "priority": null,
        "header_block_fragment": "this is dummy",
        "padding_length": null,
        "padding": null
      },
      "flags": 0,
      "stream_identifier": 1,
      "type": 1
    },
    "draft": 14,
    "description": "noraml headers frame"
  }`)

	// ignore error
	json.Unmarshal(framecase, &c)
	json.Unmarshal(c.Frame.Payload, &p)

	// trace data
	wire := c.Wire
	flags := Flag(c.Frame.Flags)
	streamId := c.Frame.StreamId
	headerBlock := []byte(p.HeaderBlockFragment)

	// compare struct
	expected := NewHeadersFrame(flags, streamId, nil, headerBlock, nil)
	actual, err := ReadFrame(hexToBuffer(wire))
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, actual, expected)

	// compare wire
	buf := bytes.NewBuffer(make([]byte, 0))
	expected.Write(buf)
	hexdump := strings.ToUpper(hex.EncodeToString(buf.Bytes()))
	assert.Equal(t, wire, hexdump)
}

type HeadersPriorityPayload struct {
	HeaderBlockFragment string `json:"header_block_fragment"`
	Padding             string `json:"padding"`
	PaddingLength       uint8  `json:"padding_length"`
	Priority            struct {
		Exclusive        bool   `json:"exclusive"`
		StreamDependency uint32 `json:"stream_dependency"`
		Weight           uint8  `json:"weight"`
	} `json:"priority"`
}

func TestHeadersPriorityCase(t *testing.T) {
	var c TestCase
	var p HeadersPriorityPayload
	framecase := []byte(`{
    "error": null,
    "wire": "000023012800000003108000001409746869732069732064756D6D79546869732069732070616464696E672E",
    "frame": {
      "length": 35,
      "frame_payload": {
        "priority": {
          "stream_dependency": 20,
          "weight": 10,
          "exclusive": true
        },
        "header_block_fragment": "this is dummy",
        "padding_length": 16,
        "padding": "This is padding."
      },
      "flags": 40,
      "stream_identifier": 3,
      "type": 1
    },
    "draft": 14,
    "description": "noraml headers frame including priority"
  }`)

	// ignore error
	json.Unmarshal(framecase, &c)
	json.Unmarshal(c.Frame.Payload, &p)

	// trace data
	wire := c.Wire
	flags := Flag(c.Frame.Flags)
	streamId := c.Frame.StreamId
	hb := []byte(p.HeaderBlockFragment)
	padding := []byte(p.Padding)
	dependencyTree := &DependencyTree{
		Exclusive:        p.Priority.Exclusive,
		StreamDependency: p.Priority.StreamDependency,
		Weight:           p.Priority.Weight,
	}

	// compare struct
	expected := NewHeadersFrame(flags, streamId, dependencyTree, hb, padding)
	actual, err := ReadFrame(hexToBuffer(wire))
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, actual, expected)

	// compare wire
	buf := bytes.NewBuffer(make([]byte, 0))
	expected.Write(buf)
	hexdump := strings.ToUpper(hex.EncodeToString(buf.Bytes()))
	assert.Equal(t, wire, hexdump)
}

// TODO: PRIORITY Frame

// RST_STREAM Frame
func TestRstStreamFrame(t *testing.T) {
	expected := NewRstStreamFrame(PROTOCOL_ERROR, 1)

	buf := bytes.NewBuffer(make([]byte, 0))
	expected.Write(buf)

	fh := new(FrameHeader)
	fh.Read(buf)

	actual := new(RstStreamFrame)
	actual.FrameHeader = fh
	actual.Read(buf)

	assert.Equal(t, actual, expected)
}

func TestRstStreamCase(t *testing.T) {
	var c TestCase
	framecase := []byte(`{
    "error": null,
    "wire": "00000403000000000500000008",
    "frame": {
      "length": 4,
      "frame_payload": {
        "error_code": 8,
        "padding_length": null,
        "padding": null
      },
      "flags": 0,
      "stream_identifier": 5,
      "type": 3
    },
    "draft": 14,
    "description": "noraml rst stream frame"
  }`)

	err := json.Unmarshal(framecase, &c)
	if err != nil {
		t.Fatal(err)
	}

	// trace data
	wire := c.Wire
	streamId := c.Frame.StreamId

	// compare struct
	expected := NewRstStreamFrame(streamId, 8)
	actual, err := ReadFrame(hexToBuffer(wire))
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, actual, expected)

	// compare wire
	buf := bytes.NewBuffer(make([]byte, 0))
	expected.Write(buf)
	hexdump := strings.ToUpper(hex.EncodeToString(buf.Bytes()))

	assert.Equal(t, wire, hexdump)
}

// SETTINGS Frame
func TestSettingsFrame(t *testing.T) {
	settings := map[SettingsID]int32{
		SETTINGS_MAX_CONCURRENT_STREAMS: 100,
		SETTINGS_INITIAL_WINDOW_SIZE:    DEFAULT_INITIAL_WINDOW_SIZE,
	}
	expected := NewSettingsFrame(2, 0, settings)

	buf := bytes.NewBuffer(make([]byte, 0))
	expected.Write(buf)

	fh := new(FrameHeader)
	fh.Read(buf)

	actual := new(SettingsFrame)
	actual.FrameHeader = fh
	actual.Read(buf)

	assert.Equal(t, actual, expected)
}

func TestSettingsCase(t *testing.T) {
	var c TestCase
	framecase := []byte(`{
    "error": null,
    "wire": "00000C040000000000000100002000000300001388",
    "frame": {
      "length": 12,
      "frame_payload": {
        "settings": [
          [
            1,
            8192
          ],
          [
            3,
            5000
          ]
        ],
        "padding_length": null,
        "padding": null
      },
      "flags": 0,
      "stream_identifier": 0,
      "type": 4
    },
    "draft": 14,
    "description": "noraml rst stream frame"
  }`)

	err := json.Unmarshal(framecase, &c)
	if err != nil {
		t.Fatal(err)
	}

	// trace data
	wire := c.Wire
	flags := Flag(c.Frame.Flags)
	streamId := c.Frame.StreamId

	settings := map[SettingsID]int32{
		1: 8192,
		3: 5000,
	}

	// compare struct
	expected := NewSettingsFrame(flags, streamId, settings)
	actual, err := ReadFrame(hexToBuffer(wire))
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, actual, expected)

	// compare wire
	buf := bytes.NewBuffer(make([]byte, 0))
	expected.Write(buf)
	hexdump := strings.ToUpper(hex.EncodeToString(buf.Bytes()))
	assert.Equal(t, wire, hexdump)
}

type PushPromisePayload struct {
	HeaderBlockFragment string `json:"header_block_fragment"`
	Padding             string `json:"padding"`
	PaddingLength       uint8  `json:"padding_length"`
	PromisedStreamID    uint32 `json:"promised_stream_id"`
}

// PUSH_PROMISE Frame
func TestPushPromiseCase(t *testing.T) {
	var c TestCase
	var p PushPromisePayload
	framecase := []byte(`{
    "error": null,
    "wire": "000018050800000009060000000B746869732069732064756D6D79486F77647921",
    "frame": {
      "length": 24,
      "frame_payload": {
        "header_block_fragment": "this is dummy",
        "padding_length": 6,
        "promised_stream_id": 11,
        "padding": "Howdy!"
      },
      "flags": 8,
      "stream_identifier": 9,
      "type": 5
    },
    "draft": 14,
    "description": "noraml push promise frame"
  }`)

	// ignore error
	json.Unmarshal(framecase, &c)
	json.Unmarshal(c.Frame.Payload, &p)

	// trace data
	wire := c.Wire
	flags := Flag(c.Frame.Flags)
	streamId := c.Frame.StreamId
	promisedStreamId := p.PromisedStreamID
	headerBlockFragment := []byte(p.HeaderBlockFragment)
	padding := []byte(p.Padding)

	// compare struct
	expected := NewPushPromiseFrame(flags, streamId, promisedStreamId, headerBlockFragment, padding)
	actual, err := ReadFrame(hexToBuffer(wire))
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, actual, expected)

	// compare wire
	buf := bytes.NewBuffer(make([]byte, 0))
	expected.Write(buf)
	hexdump := strings.ToUpper(hex.EncodeToString(buf.Bytes()))
	assert.Equal(t, wire, hexdump)
}

type PingPayload struct {
	OpaqueData    string `json:"opaque_data"`
	Padding       string `json:"padding"`
	PaddingLength uint8  `json:"padding_length"`
}

// PING Frame
func TestPingCase(t *testing.T) {
	var c TestCase
	var p PingPayload
	framecase := []byte(`{
    "error": null,
    "wire": "0000080600000000006465616462656566",
    "frame": {
      "length": 8,
      "frame_payload": {
        "opaque_data": "deadbeef",
        "padding_length": null,
        "padding": null
      },
      "flags": 0,
      "stream_identifier": 0,
      "type": 6
    },
    "draft": 14,
    "description": "noraml ping frame"
  }`)

	// ignore error
	json.Unmarshal(framecase, &c)
	json.Unmarshal(c.Frame.Payload, &p)

	// trace data
	wire := c.Wire
	flags := Flag(c.Frame.Flags)
	streamId := c.Frame.StreamId
	opaqueData := []byte(p.OpaqueData)

	// compare struct
	expected := NewPingFrame(flags, streamId, opaqueData)
	actual, err := ReadFrame(hexToBuffer(wire))
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, actual, expected)

	// compare wire
	buf := bytes.NewBuffer(make([]byte, 0))
	expected.Write(buf)
	hexdump := strings.ToUpper(hex.EncodeToString(buf.Bytes()))
	assert.Equal(t, wire, hexdump)
}

// GOAWAY Frame
func TestGoAwayFrame(t *testing.T) {
	expected := NewGoAwayFrame(101, 100, NO_ERROR, nil)

	buf := bytes.NewBuffer(make([]byte, 0))
	expected.Write(buf)

	fh := new(FrameHeader)
	fh.Read(buf)

	actual := new(GoAwayFrame)
	actual.FrameHeader = fh
	actual.Read(buf)

	assert.Equal(t, actual, expected)
}

type GoAwayPayload struct {
	AdditionalDebugData string    `json:"additional_debug_data"`
	ErrorCode           ErrorCode `json:"error_code"`
	LastStreamID        uint32    `json:"last_stream_id"`
	Padding             string    `json:"padding"`
	PaddingLength       uint8     `json:"padding_length"`
}

func TestGoAwayCase(t *testing.T) {
	var c TestCase
	var p GoAwayPayload
	framecase := []byte(`{
    "error": null,
    "wire": "0000170700000000000000001E00000009687061636B2069732062726F6B656E",
    "frame": {
      "length": 23,
      "frame_payload": {
        "error_code": 9,
        "additional_debug_data": "hpack is broken",
        "padding_length": null,
        "last_stream_id": 30,
        "padding": null
      },
      "flags": 0,
      "stream_identifier": 0,
      "type": 7
    },
    "draft": 14,
    "description": "normal goaway frame"
  }`)

	json.Unmarshal(framecase, &c)
	json.Unmarshal(c.Frame.Payload, &p)

	// trace data
	wire := c.Wire
	streamId := c.Frame.StreamId
	lastStreamId := p.LastStreamID
	errorCode := p.ErrorCode
	additional := []byte(p.AdditionalDebugData)

	// compare struct
	expected := NewGoAwayFrame(streamId, lastStreamId, errorCode, additional)
	actual, err := ReadFrame(hexToBuffer(wire))
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, actual, expected)

	// compare wire
	buf := bytes.NewBuffer(make([]byte, 0))
	expected.Write(buf)
	hexdump := strings.ToUpper(hex.EncodeToString(buf.Bytes()))
	assert.Equal(t, wire, hexdump)
}

type WindowUpdatePayload struct {
	Padding             []byte `json:"padding"`
	PaddingLength       uint8  `json:"padding_length"`
	WindowSizeIncrement uint32 `json:"window_size_increment"`
}

// WINDOW_UPDATE Frame
func TestWindowUpdate(t *testing.T) {
	var c TestCase
	var p WindowUpdatePayload
	framecase := []byte(`{
    "error": null,
    "wire": "000004080000000032000003E8",
    "frame": {
      "length": 4,
      "frame_payload": {
        "window_size_increment": 1000,
        "padding_length": null,
        "padding": null
      },
      "flags": 0,
      "stream_identifier": 50,
      "type": 8
    },
    "draft": 14,
    "description": "noraml window update frame"
  }`)

	json.Unmarshal(framecase, &c)
	json.Unmarshal(c.Frame.Payload, &p)

	// trace data
	wire := c.Wire
	streamId := c.Frame.StreamId
	incrementSize := p.WindowSizeIncrement

	// compare struct
	expected := NewWindowUpdateFrame(streamId, incrementSize)
	actual, err := ReadFrame(hexToBuffer(wire))
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, actual, expected)

	// compare wire
	buf := bytes.NewBuffer(make([]byte, 0))
	expected.Write(buf)
	hexdump := strings.ToUpper(hex.EncodeToString(buf.Bytes()))
	assert.Equal(t, wire, hexdump)
}

type ContinuationPayload struct {
	HeaderBlockFragment string `json:"header_block_fragment"`
	Padding             string `json:"padding"`
	PaddingLength       uint8  `json:"padding_length"`
}

// CONTINUATION Frame
func TestContinuationCase(t *testing.T) {
	var c TestCase
	var p ContinuationPayload
	framecase := []byte(`{
    "error": null,
    "wire": "00000D090000000032746869732069732064756D6D79",
    "frame": {
      "length": 13,
      "frame_payload": {
        "header_block_fragment": "this is dummy",
        "padding_length": null,
        "padding": null
      },
      "flags": 0,
      "stream_identifier": 50,
      "type": 9
    },
    "draft": 14,
    "description": "normal continuation frame without header block fragment"
  }`)

	// ignore error
	json.Unmarshal(framecase, &c)
	json.Unmarshal(c.Frame.Payload, &p)

	// trace data
	wire := c.Wire
	streamId := c.Frame.StreamId
	flags := Flag(c.Frame.Flags)
	headerBlockFragment := []byte(p.HeaderBlockFragment)

	// compare struct
	expected := NewContinuationFrame(flags, streamId, headerBlockFragment)
	actual, err := ReadFrame(hexToBuffer(wire))
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, actual, expected)

	// compare wire
	buf := bytes.NewBuffer(make([]byte, 0))
	expected.Write(buf)
	hexdump := strings.ToUpper(hex.EncodeToString(buf.Bytes()))
	assert.Equal(t, wire, hexdump)
}

// Helper
func hexToBuffer(str string) *bytes.Buffer {
	w, _ := hex.DecodeString(str)
	return bytes.NewBuffer(w)
}
