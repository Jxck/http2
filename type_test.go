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
