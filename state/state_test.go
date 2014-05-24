package state

import (
	. "github.com/jxck/http2/frame"
	"testing"
)

var headers *HeadersFrame = new(HeadersFrame)
var rst *RstStreamFrame = new(RstStreamFrame)
var data *DataFrame = new(DataFrame)

func TestStory(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatal(r)
		}
	}()

	var idle, open, halfclosed, closed State

	idle = NewIdle()
	open, _ = idle.H(headers)
	closed, _ = open.R(rst)

	idle = NewIdle()
	open, _ = idle.H(headers)
	halfclosed, _ = open.ES(data)
	closed, _ = halfclosed.ES(data)

	idle = NewIdle()
	open, _ = idle.H(headers)
	halfclosed, _ = open.ES(data)
	closed, _ = halfclosed.R(rst)

	_ = closed
}

func TestSuccess(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatal(r)
		}
	}()

	NewIdle().H(headers)

	NewOpen().ES(data)
	NewOpen().R(rst)

	NewHalfClosed().ES(data)
	NewHalfClosed().R(rst)
}

func TestFail(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Log(r)
		}
	}()

	NewIdle().ES(data)
	NewIdle().R(rst)

	NewOpen().H(headers)

	NewHalfClosed().H(headers)

	NewClosed().H(headers)
	NewClosed().ES(data)
	NewClosed().R(rst)
}

func TestExtend(t *testing.T) {
	NewIdle().R(rst)
}
