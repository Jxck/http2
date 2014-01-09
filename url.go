package http2

import (
	"fmt"
	"log"
	neturl "net/url"
	"strings"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

// Exted net/url with adding Port
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
