package http2

import (
	"bytes"
	"fmt"
	assert "github.com/Jxck/assertion"
	"net/http"
	"reflect"
	"testing"
)

func TestAddPrefix(t *testing.T) {
	actual := make(http.Header)
	actual.Add("method", "get")
	actual.Add("scheme", "http")
	actual.Add("host", "www.example.com")
	actual.Add("path", "/index.html")
	actual.Add("cookie", "a")
	actual.Add("cookie", "b")
	actual.Add("x-http2-version", "draft-09")
	actual = util.AddPrefix(actual)

	expected := make(http.Header)
	expected.Add(":method", "get")
	expected.Add(":scheme", "http")
	expected.Add("host", "www.example.com")
	expected.Add(":path", "/index.html")
	expected.Add("cookie", "a")
	expected.Add("cookie", "b")
	expected.Add("x-http2-version", "draft-09")

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("\ngot  %v\nwant %v", actual, expected)
	}
}

func TestRemovePrefix(t *testing.T) {
	actual := make(http.Header)
	actual.Add(":method", "get")
	actual.Add(":scheme", "http")
	actual.Add("host", "www.example.com")
	actual.Add(":path", "/index.html")
	actual.Add("cookie", "a")
	actual.Add("cookie", "b")
	actual.Add("x-http2-version", "draft-09")

	expected := make(http.Header)
	expected.Add("method", "get")
	expected.Add("scheme", "http")
	expected.Add("host", "www.example.com")
	expected.Add("path", "/index.html")
	expected.Add("cookie", "a")
	expected.Add("cookie", "b")
	expected.Add("x-http2-version", "draft-09")
	expected = util.AddPrefix(actual)

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("\ngot  %v\nwant %v", actual, expected)
	}
}

func TestMustWrite(t *testing.T) {
	var u8 uint8 = 10
	buf := bytes.NewBuffer(make([]byte, 0))
	MustWrite(buf, &u8)
	assert.Equal(t, buf.Bytes()[0], u8)
}

func TestMustWriteError(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			switch e := r.(type) {
			case error:
				t.Log(e)
			default:
				t.Log(fmt.Errorf("%v", e))
			}
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
		if r := recover(); r != nil {
			switch e := r.(type) {
			case error:
				t.Log(e)
			default:
				t.Log(fmt.Errorf("%v", e))
			}
		}
	}()
	var u8 uint8 = 10
	MustRead(nil, &u8)
	t.Fatal("can't be here")
}
