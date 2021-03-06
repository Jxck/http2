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

func (u Util) UpgradeRequest(req *http.Request, url *URL) *http.Request {
	// TODO: manage header duplicat
	req.Header.Add(":authority", url.Host)
	req.Header.Add(":method", req.Method)
	req.Header.Add(":path", url.Path)
	req.Header.Add(":scheme", url.Scheme)
	return req
}

func (u Util) RequestString(req *http.Request) string {
	str := fmt.Sprintf("%v %v %v", req.Method, req.URL, req.Proto)
	for name, value := range req.Header {
		str += fmt.Sprintf("\n%s: %s", name, strings.Join(value, ","))
	}
	return str
}

func (u Util) ResponseString(res *http.Response) string {
	str := fmt.Sprintf("%v %v", res.Proto, res.Status)
	for name, value := range res.Header {
		str += fmt.Sprintf("\n%s: %s", name, strings.Join(value, ","))
	}
	return str
}

func (u Util) Indent(v interface{}) string {
	return strings.Replace(fmt.Sprintf("%v", v), "\n", "\n\t\t\t\t", -1)
}
