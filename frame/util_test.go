package frame

import (
	"bytes"
	assert "github.com/Jxck/assertion"
	"testing"
)

func TestMustWrite(t *testing.T) {
	var u8 uint8 = 10
	buf := bytes.NewBuffer(make([]byte, 0))
	MustWrite(buf, &u8)
	assert.Equal(t, buf.Bytes()[0], u8)
}

func TestMustWriteError(t *testing.T) {
	defer func() {
		e := Recovery(recover())
		if e == nil {
			t.Fatal("error is nil")
		}
	}()
	var u8 uint8 = 10
	MustWrite(nil, &u8)
	t.Fatal("can't be here")
}

func TestMustRead(t *testing.T) {
	var u8 uint8
	buf := bytes.NewBuffer([]byte{10})
	MustRead(buf, &u8)
	assert.Equal(t, uint8(10), u8)
}

func TestMustReadError(t *testing.T) {
	defer func() {
		e := Recovery(recover())
		if e == nil {
			t.Fatal("error is nil")
		}
	}()
	var u8 uint8 = 10
	MustRead(nil, &u8)
	t.Fatal("can't be here")
}
