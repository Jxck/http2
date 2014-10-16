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

var (
	NextClientStreamID chan uint32 = util.NextID(1)
	NextServerStreamID chan uint32 = util.NextID(2)
)

func (u Util) NextID(id uint32) chan uint32 {
	idChan := make(chan uint32)
	go func() {
		for {
			if id >= 4294967295 || id < 0 { // 2^32-1 or invalid
				log.Println("stream id too big or invalid, return to 0")
				id = 0
			}
			idChan <- id
			id = id + 2
		}
	}()
	return idChan
}

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

func (u Util) UpgradeRequest(req *http.Request, url *URL) *http.Request {
	if req.ContentLength != 0 {
		req.Header.Add("content-length", fmt.Sprintf("%d", req.ContentLength))
	}

	// TODO: manage header duplicat
	req.Header.Add(":authority", url.Host)
	req.Header.Add(":method", req.Method)
	req.Header.Add(":path", url.Path)
	req.Header.Add(":scheme", url.Scheme)
	req.Header.Add("accept", "*/*")
	req.Header.Add("x-http2-version", VERSION)
	return req
}

func (u Util) RequestString(req *http.Request) string {
	str := fmt.Sprintf("%v %v %v", req.Method, req.URL, req.Proto)
	for name, value := range req.Header {
		str += fmt.Sprintf("\n%s: %s", name, strings.Join(value, ","))
	}
	return str
}

func (u Util) Indent(v interface{}) string {
	return "\t" + strings.Replace(fmt.Sprintf("%v", v), "\n", "\n\t\t\t\t", -1)
}
