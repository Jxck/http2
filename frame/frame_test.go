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
	Length   uint32                 `json:"length"`
	Payload  map[string]interface{} `json:"frame_payload"`
	Flags    uint8                  `json:"flags"`
	StreamId uint32                 `json:"stream_identifier"`
	Type     uint8                  `json:"type"`
}

// Frame Header
func TestFrameHeaderQuickCheck(t *testing.T) {
	f := func(length uint32, types uint8, flags uint8, streamId uint32) bool {
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
	f := func(flags uint8, streamId uint32, data []byte) bool {
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

func TestDataCase(t *testing.T) {
	var c TestCase
	DataFrameCase := []byte(`{
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

	err := json.Unmarshal(DataFrameCase, &c)
	if err != nil {
		t.Fatal(err)
	}

	// trace data
	wire := c.Wire
	flags := c.Frame.Flags
	streamId := c.Frame.StreamId
	data := []byte(c.Frame.Payload["data"].(string))
	padding := []byte(c.Frame.Payload["padding"].(string))

	// compare struct
	expected := NewDataFrame(flags, streamId, data, padding)
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

func TestHeadersCase(t *testing.T) {
	var c TestCase
	HeadersFrameCase := []byte(`{
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

	err := json.Unmarshal(HeadersFrameCase, &c)
	if err != nil {
		t.Fatal(err)
	}

	// trace data
	wire := c.Wire
	flags := c.Frame.Flags
	streamId := c.Frame.StreamId
	headerBlock := []byte(c.Frame.Payload["header_block_fragment"].(string))

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

func TestHeadersPriorityCase(t *testing.T) {
	var c TestCase
	HeadersFrameCase := []byte(`{
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

	err := json.Unmarshal(HeadersFrameCase, &c)
	if err != nil {
		t.Fatal(err)
	}

	// trace data
	wire := c.Wire
	flags := c.Frame.Flags
	streamId := c.Frame.StreamId
	hb := []byte(c.Frame.Payload["header_block_fragment"].(string))
	padding := []byte(c.Frame.Payload["padding"].(string))
	dependencyTree := &DependencyTree{
		Exclusive:        true,
		StreamDependency: 20,
		Weight:           10,
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
	RstStreamFrameCase := []byte(`{
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

	err := json.Unmarshal(RstStreamFrameCase, &c)
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
	settings := []Setting{
		{SETTINGS_MAX_CONCURRENT_STREAMS, 100},
		{SETTINGS_INITIAL_WINDOW_SIZE, DEFAULT_WINDOW_SIZE},
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
	SettingsFrameCase := []byte(`{
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

	err := json.Unmarshal(SettingsFrameCase, &c)
	if err != nil {
		t.Fatal(err)
	}

	// trace data
	wire := c.Wire
	flags := c.Frame.Flags
	streamId := c.Frame.StreamId
	settings := []Setting{
		{1, 8192},
		{3, 5000},
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

// PUSH_PROMISE Frame
func TestPushPromiseCase(t *testing.T) {
	var c TestCase
	casestr := []byte(`{
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

	err := json.Unmarshal(casestr, &c)
	if err != nil {
		t.Fatal(err)
	}

	// trace data
	wire := c.Wire
	flags := c.Frame.Flags
	streamId := c.Frame.StreamId
	promisedStreamId := uint32(c.Frame.Payload["promised_stream_id"].(float64))
	headerBlockFragment := []byte(c.Frame.Payload["header_block_fragment"].(string))
	padding := []byte(c.Frame.Payload["padding"].(string))

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

// PING Frame
func TestPingCase(t *testing.T) {
	var c TestCase
	PingFrameCase := []byte(`{
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

	err := json.Unmarshal(PingFrameCase, &c)
	if err != nil {
		t.Fatal(err)
	}

	// trace data
	wire := c.Wire
	flags := c.Frame.Flags
	streamId := c.Frame.StreamId
	opaqueData := []byte(c.Frame.Payload["opaque_data"].(string))

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

func TestGoAwayCase(t *testing.T) {
	var c TestCase
	GoAwayFrameCase := []byte(`{
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

	err := json.Unmarshal(GoAwayFrameCase, &c)
	if err != nil {
		t.Fatal(err)
	}

	// trace data
	wire := c.Wire
	streamId := c.Frame.StreamId
	lastStreamId := uint32(c.Frame.Payload["last_stream_id"].(float64))
	errorCode := ErrorCode(c.Frame.Payload["error_code"].(float64))
	additional := []byte(c.Frame.Payload["additional_debug_data"].(string))

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

// WINDOW_UPDATE Frame
func TestWindowUpdate(t *testing.T) {
	var c TestCase
	WindowUpdateFrameCase := []byte(`{
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

	err := json.Unmarshal(WindowUpdateFrameCase, &c)
	if err != nil {
		t.Fatal(err)
	}

	// trace data
	wire := c.Wire
	streamId := c.Frame.StreamId
	incrementSize := uint32(c.Frame.Payload["window_size_increment"].(float64))

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

// CONTINUATION Frame
func TestContinuationCase(t *testing.T) {
	var c TestCase
	ContinuationFrameCase := []byte(`{
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

	err := json.Unmarshal(ContinuationFrameCase, &c)
	if err != nil {
		t.Fatal(err)
	}

	// trace data
	wire := c.Wire
	streamId := c.Frame.StreamId
	flags := c.Frame.Flags
	headerBlockFragment := []byte(c.Frame.Payload["header_block_fragment"].(string))

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
