package http2

import (
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
	actual = AddPrefix(actual)

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
	expected = AddPrefix(actual)

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("\ngot  %v\nwant %v", actual, expected)
	}
}
