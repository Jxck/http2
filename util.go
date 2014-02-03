package http2

import (
	"fmt"
	"log"
	"net/http"
	"strings"
)

var util = Util{}

func init() {
	log.SetFlags(log.Lshortfile)
}

// Must Header with prefix
var MustHeader = map[string]string{
	":authority": "authority",
	":method":    "method",
	":path":      "path",
	":scheme":    "scheme",
	":status":    "status",
	// invert
	"Authority": ":authority",
	"Method":    ":method",
	"Path":      ":path",
	"Scheme":    ":scheme",
	"Status":    ":status",
}

type Util struct{}

func (u Util) AddPrefix(header http.Header) http.Header {
	for key, values := range header {
		name, ok := MustHeader[key]
		if ok {
			header.Del(key)
			for _, value := range values {
				header.Add(name, value)
			}
		}
	}
	return header
}

func (u Util) RemovePrefix(header http.Header) http.Header {
	for key, values := range header {
		name, ok := MustHeader[key]
		if ok {
			header.Del(key)
			for _, value := range values {
				header.Add(name, value)
			}
		}
	}
	return header
}

func (u Util) UpdateRequest(req *http.Request, url *URL) *http.Request {
	if req.ContentLength != 0 {
		req.Header.Add("content-length", fmt.Sprintf("%d", req.ContentLength))
	}

	// TODO: manage header duplicat
	req.Header.Add(":authority", url.Host)
	req.Header.Add(":method", req.Method)
	req.Header.Add(":path", url.Path)
	req.Header.Add(":scheme", url.Scheme)
	req.Header.Add("accept", "*/*")
	req.Header.Add("x-http2-version", Version)
	return req
}

func (u Util) RequestString(req *http.Request) string {
	str := fmt.Sprintf("%v %v %v", req.Method, req.URL, req.Proto)
	for k, v := range req.Header {
		str += fmt.Sprintf("\n%v: %v", k, v[0])
	}
	return str
}

func (u Util) ResponseString(res *http.Response) string {
	str := fmt.Sprintf("%v %v", res.Proto, res.Status)
	for k, v := range res.Header {
		str += fmt.Sprintf("\n%v: %v", k, v[0])
	}
	return str
}

func (u Util) Indent(v interface{}) string {
	return strings.Replace(fmt.Sprintf("%v", v), "\n", "\n\t\t\t\t", -1)
}
