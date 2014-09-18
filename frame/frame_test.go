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

type DataFrameJSON struct {
	Error string    `json:"error"`
	Wire  string    `json:"wire"`
	Frame FrameType `json:"frame"`
}

type FrameType struct {
	Length           uint32      `json:"length"`
	Payload          PayloadType `json:"frame_payload"`
	Flags            uint8       `json:"flags"`
	StreamIdentifier uint32      `json:"stream_identifier"`
	Type             uint8       `json:"type"`
}

type PayloadType struct {
	Data          string `json:"data"`
	PaddingLength uint8  `json:"padding_length"`
	Padding       string `json:"padding"`
}

func TestCase(t *testing.T) {
	var dataFrameJSON DataFrameJSON
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

	err := json.Unmarshal(DataFrameCase, &dataFrameJSON)
	if err != nil {
		t.Fatal(err)
	}

	wire := dataFrameJSON.Wire
	length := dataFrameJSON.Frame.Length
	flags := dataFrameJSON.Frame.Flags
	streamId := dataFrameJSON.Frame.StreamIdentifier
	data := dataFrameJSON.Frame.Payload.Data
	paddlength := dataFrameJSON.Frame.Payload.PaddingLength
	padding := dataFrameJSON.Frame.Payload.Padding
	types := dataFrameJSON.Frame.Type

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

func hexToBuffer(str string) *bytes.Buffer {
	w, _ := hex.DecodeString(str)
	return bytes.NewBuffer(w)
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
