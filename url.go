package http2

import (
	"fmt"
	"log"
	urllib "net/url"
	"strings"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

type URL struct {
	*urllib.URL
	Port string
}

func (url *URL) SplitHostPort() (err error) {
	splitted := strings.Split(url.Host, ":")
	switch {
	case len(splitted) > 2:
		return fmt.Errorf("parse error %q in HOST:PORT", url.Host)
	case len(splitted) > 1:
		url.Host, url.Port = splitted[0], splitted[1]
		return
	case url.Scheme == "https":
		url.Host, url.Port = splitted[0], "443"
		return
	case url.Scheme == "http":
		url.Host, url.Port = splitted[0], "80"
		return
	}
	return
}

func NewURL(rawurl string) (*URL, error) {
	u, err := urllib.Parse(rawurl)
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
