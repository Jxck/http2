package frame

import (
	"bytes"
	"encoding/hex"
	"reflect"
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

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("got %v\nwant %v", actual, expected)
	}
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

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("got %v\nwant %v", actual, expected)
	}
}

// {
// 	"error": null,
// 	"wire": "0000140008000000020648656C6C6F2C20776F726C6421486F77647921",
// 	"frame": {
// 		"length": 20,
// 		"frame_payload": {
// 			"data": "Hello, world!",
// 			"padding_length": 6,
// 			"padding": "Howdy!"
// 		},
// 		"flags": 8,
// 		"stream_identifier": 2,
// 		"type": 0
// 	},
// 	"draft": 14,
// 	"description": "noraml data frame"
// }
func TestCase(t *testing.T) {
	var (
		flags    uint8  = 8
		streamId uint32 = 2
		length   uint32 = 20
		types    uint8  = 0
	)

	expected := NewDataFrame(flags, streamId)
	expected.Length = 20
	expected.Data = []byte("Hello, world!")
	expected.FrameHeader.Length = length
	expected.FrameHeader.Type = types

	var wire = "0000140008000000020648656C6C6F2C20776F726C6421486F77647921"
	w, _ := hex.DecodeString(wire)
	buf := bytes.NewBuffer(w)
	actual, err := ReadFrame(buf)
	if err != nil {
		t.Fatal(err)
	}

	// compare struct
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("got %v\nwant %v", actual, expected)
	}
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

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("got %v\nwant %v", actual, expected)
	}
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

	if actual.StreamDependency != 1 {
		t.Errorf("got %v\nwant %v", actual.StreamDependency, 1)
	}

	if actual.Weight != 2 {
		t.Errorf("got %v\nwant %v", actual.Weight, 2)
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("got %v\nwant %v", actual, expected)
	}
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

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("got %v\nwant %v", actual, expected)
	}
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

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("got %v\nwant %v", actual, expected)
	}
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

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("got %v\nwant %v", actual, expected)
	}
}
