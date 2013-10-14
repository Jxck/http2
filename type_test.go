package http2

import (
	"bytes"
	"reflect"
	"testing"
)

func TestFrameHeader(t *testing.T) {
	expected := &FrameHeader{
		Length:   8,
		Type:     1,
		Flags:    2,
		StreamId: 3,
	}
	buf := bytes.NewBuffer([]byte{})
	expected.Write(buf)

	actual := &FrameHeader{}
	actual.Read(buf)

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("got %v\nwant %v", actual, expected)
	}
}

func TestDataFrame(t *testing.T) {
	b := []byte("hello")
	expected := NewDataFrame(1, 2)
	expected.Length = uint16(len(b))
	expected.Data = b

	buf := bytes.NewBuffer([]byte{})
	expected.Write(buf)

	fh := &FrameHeader{}
	fh.Read(buf)

	actual := &DataFrame{}
	actual.FrameHeader = fh
	actual.Read(buf)

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("got %v\nwant %v", actual, expected)
	}
}

func TestHeadersFrame(t *testing.T) {
	b := []byte("test header block")
	expected := NewHeadersFrame(END_STREAM, 2)
	expected.Length = uint16(len(b))
	expected.HeaderBlock = b

	buf := bytes.NewBuffer([]byte{})
	expected.Write(buf)

	fh := &FrameHeader{}
	fh.Read(buf)

	actual := &HeadersFrame{}
	actual.FrameHeader = fh
	actual.Read(buf)

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("got %v\nwant %v", actual, expected)
	}
}

func TestHeadersPriorityFrame(t *testing.T) {
	b := []byte("test header block")
	expected := NewHeadersFrame(PRIORITY, 2)
	expected.Priority = 1
	expected.Length = uint16(len(b) + 4)
	expected.HeaderBlock = b

	buf := bytes.NewBuffer([]byte{})
	expected.Write(buf)

	fh := &FrameHeader{}
	fh.Read(buf)

	actual := &HeadersFrame{}
	actual.FrameHeader = fh
	actual.Read(buf)

	if actual.Priority != 1 {
		t.Errorf("got %v\nwant %v", actual.Priority, 1)
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("got %v\nwant %v", actual, expected)
	}
}

func TestSettingsFrame(t *testing.T) {
	settings := map[SettingsId]uint32{
		SETTINGS_MAX_CONCURRENT_STREAMS: 100,
		SETTINGS_INITIAL_WINDOW_SIZE:    DEFAULT_WINDOW_SIZE,
	}
	expected := NewSettingsFrame(settings, 2)

	buf := bytes.NewBuffer([]byte{})
	expected.Write(buf)

	fh := &FrameHeader{}
	fh.Read(buf)

	actual := &SettingsFrame{}
	actual.FrameHeader = fh
	actual.Read(buf)

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("got %v\nwant %v", actual, expected)
	}
}
