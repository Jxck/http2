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

	var idle State

	idle = NewIdle()
	idle.H(headers).R(rst)

	idle = NewIdle()
	idle.H(headers).ES(data).ES(data)

	idle = NewIdle()
	idle.H(headers).ES(data).R(rst)
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

	NewReserved().H(headers)
	NewReserved().R(rst)

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

	NewReserved().ES(data)

	NewHalfClosed().H(headers)

	NewClosed().H(headers)
	NewClosed().ES(data)
	NewClosed().R(rst)
}
