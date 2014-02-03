package http2

import (
	"fmt"
	"log"
	"net/http"
	neturl "net/url"
	"strings"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

// Exted net/url with adding Port
// because tls.Dial needs port number
type URL struct {
	*neturl.URL
	Port string
}

func NewURL(rawurl string) (*URL, error) {
	u, err := neturl.Parse(rawurl)
	if err != nil {
		return nil, err
	}
	url := &URL{
		u,
		"",
	}
	err = url.SplitHostPort()
	if err != nil {
		return nil, err
	}
	return url, nil
}

// TODO: net#SplitHostPort
func (url *URL) SplitHostPort() (err error) {
	splitted := strings.Split(url.Host, ":")
	switch {
	case len(splitted) > 2:
		err = fmt.Errorf("parse error %q in HOST:PORT", url.Host)
	case len(splitted) > 1:
		url.Host, url.Port = splitted[0], splitted[1]
	case url.Scheme == "https":
		url.Host, url.Port = splitted[0], "443"
	case url.Scheme == "http":
		url.Host, url.Port = splitted[0], "80"
	}
	return
}

func (url *URL) Update(req *http.Request) *http.Request {
	if url.Path == "" {
		url.Path = "/"
	}
	if req.ContentLength != 0 {
		req.Header.Add("content-length", fmt.Sprintf("%d", req.ContentLength))
	}

	req.Header.Add(":authority", url.Host)
	req.Header.Add(":method", req.Method)
	req.Header.Add(":path", url.Path)
	req.Header.Add(":scheme", url.Scheme)
	req.Header.Add("accept", "*/*")
	req.Header.Add("x-http2-version", Version)
	return req
}
