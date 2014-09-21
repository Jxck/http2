package frame

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	assert "github.com/Jxck/assertion"
	"strings"
	"testing"
)

func TestFrameHeader(t *testing.T) {
	var (
		length   uint32 = 8
		types    uint8  = 1
		flags    uint8  = 2
		streamid uint32 = 3
	)
	expected := NewFrameHeader(length, types, flags, streamid)
	buf := bytes.NewBuffer(make([]byte, 0))
	expected.Write(buf)

	actual := new(FrameHeader)
	actual.Read(buf)

	assert.Equal(t, actual, expected)
}

func TestDataFrame(t *testing.T) {
	var (
		flags    uint8  = 1
		streamid uint32 = 2
		b        []byte = []byte("hello")
	)

	expected := NewDataFrame(flags, streamid)
	expected.Length = uint32(len(b))
	expected.Data = b

	buf := bytes.NewBuffer(make([]byte, 0))
	expected.Write(buf)

	fh := new(FrameHeader)
	fh.Read(buf)

	actual := new(DataFrame)
	actual.FrameHeader = fh
	actual.Read(buf)

	assert.Equal(t, actual, expected)
}

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

	wire := c.Wire
	length := c.Frame.Length

	flags := c.Frame.Flags
	streamId := c.Frame.StreamId
	types := c.Frame.Type

	data := c.Frame.Payload["data"].(string)
	paddlength := uint8(c.Frame.Payload["padding_length"].(float64))
	padding := c.Frame.Payload["padding"].(string)

	expected := NewDataFrame(flags, streamId)
	expected.Length = length
	expected.Type = types
	expected.PadLength = paddlength
	expected.Data = []byte(data)
	expected.Padding = []byte(padding)

	actual, err := ReadFrame(hexToBuffer(wire))
	if err != nil {
		t.Fatal(err)
	}

	// compare struct
	assert.Equal(t, actual, expected)

	// compare wire
	buf := bytes.NewBuffer(make([]byte, 0))
	expected.Write(buf)
	hexdump := strings.ToUpper(hex.EncodeToString(buf.Bytes()))

	assert.Equal(t, wire, hexdump)
}

func TestHeadersFrame(t *testing.T) {
	b := []byte("test header block")
	expected := NewHeadersFrame(END_STREAM, 2)
	expected.Length = uint32(len(b))
	expected.HeaderBlock = b

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
	b := []byte("test header block")
	expected := NewHeadersFrame(PRIORITY, 2)
	expected.StreamDependency = 1
	expected.Weight = 2
	expected.Length = uint32(len(b) + 5)
	expected.HeaderBlock = b

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

	wire := c.Wire
	length := c.Frame.Length

	flags := c.Frame.Flags
	streamId := c.Frame.StreamId
	types := c.Frame.Type

	expected := NewHeadersFrame(flags, streamId)
	expected.Length = length
	expected.Type = types
	expected.HeaderBlock = []byte(c.Frame.Payload["header_block_fragment"].(string))

	actual, err := ReadFrame(hexToBuffer(wire))
	if err != nil {
		t.Fatal(err)
	}

	// compare struct
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

	wire := c.Wire
	length := c.Frame.Length

	flags := c.Frame.Flags
	streamId := c.Frame.StreamId
	types := c.Frame.Type

	expected := NewHeadersFrame(flags, streamId)
	expected.Length = length
	expected.Type = types
	expected.StreamDependency = 20
	expected.Weight = 10
	expected.Exclusive = true
	expected.HeaderBlock = []byte(c.Frame.Payload["header_block_fragment"].(string))

	t.Log(hexToBuffer(wire).Bytes())
	actual, err := ReadFrame(hexToBuffer(wire))
	if err != nil {
		t.Fatal(err)
	}

	// compare struct
	assert.Equal(t, actual, expected)

	// compare wire
	buf := bytes.NewBuffer(make([]byte, 0))
	expected.Write(buf)
	hexdump := strings.ToUpper(hex.EncodeToString(buf.Bytes()))

	assert.Equal(t, wire, hexdump)
}

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

func TestSettingsFrame(t *testing.T) {
	settings := []Setting{
		{SETTINGS_MAX_CONCURRENT_STREAMS, 100},
		{SETTINGS_INITIAL_WINDOW_SIZE, DEFAULT_WINDOW_SIZE},
	}
	expected := NewSettingsFrame(0, settings, 2)

	buf := bytes.NewBuffer(make([]byte, 0))
	expected.Write(buf)

	fh := new(FrameHeader)
	fh.Read(buf)

	actual := new(SettingsFrame)
	actual.FrameHeader = fh
	actual.Read(buf)

	assert.Equal(t, actual, expected)
}

func TestGoAwayFrame(t *testing.T) {
	expected := NewGoAwayFrame(100, NO_ERROR, 0)

	buf := bytes.NewBuffer(make([]byte, 0))
	expected.Write(buf)

	fh := new(FrameHeader)
	fh.Read(buf)

	actual := new(GoAwayFrame)
	actual.FrameHeader = fh
	actual.Read(buf)

	assert.Equal(t, actual, expected)
}

// Helper
func hexToBuffer(str string) *bytes.Buffer {
	w, _ := hex.DecodeString(str)
	return bytes.NewBuffer(w)
}
