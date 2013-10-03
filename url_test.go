package http2

import (
	"fmt"
	"testing"
)

func TestNewURL(t *testing.T) {
	var cases = []struct {
		rawurl, scheme, host, port string
	}{
		{"http://go.com", "http", "go.com", "80"},
		{"http://go.com:80", "http", "go.com", "80"},
		{"http://go.com:8080", "http", "go.com", "8080"},
		{"https://go.com", "https", "go.com", "443"},
		{"https://go.com:443", "https", "go.com", "443"},
		{"https://go.com:8443", "https", "go.com", "8443"},
		{"http://123.123.123.123", "http", "123.123.123.123", "80"},
		{"http://123.123.123.123:80", "http", "123.123.123.123", "80"},
		{"http://123.123.123.123:8080", "http", "123.123.123.123", "8080"},
		{"https://123.123.123.123", "https", "123.123.123.123", "443"},
		{"https://123.123.123.123:443", "https", "123.123.123.123", "443"},
		{"https://123.123.123.123:8443", "https", "123.123.123.123", "8443"},
	}

	for _, s := range cases {
		url, err := NewURL(s.rawurl)
		if err != nil {
			t.Errorf("got %v", err)
		}
		if s.scheme != url.Scheme {
			t.Errorf("got %v\twant %v", url.Scheme, s.scheme)
		}
		if s.host != url.Host {
			t.Errorf("got %v\twant %v", url.Host, s.host)
		}
		if s.port != url.Port {
			t.Errorf("got %v\twant %v", url.Port, s.port)
		}
	}

	rowurl := "http://go.com:80:80"
	u, err := NewURL(rowurl)
	if u != nil || err == nil {
		t.Log(u)
		t.Log(err)
		t.Errorf("NewURL(%q) should error", rowurl)
	}
	if err.Error() != fmt.Sprintf("parse error %q in HOST:PORT", "go.com:80:80") {
		t.Errorf("NewURL(%q) should error", rowurl)
	}
}
