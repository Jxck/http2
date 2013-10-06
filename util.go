package http2

import (
	"fmt"
	"log"
	"net/http"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

func ResponseString(res *http.Response) string {
	str := fmt.Sprintf("%v %v", res.Proto, res.Status)
	for k, v := range res.Header {
		str += fmt.Sprintf("\n%v: %v", k, v[0])
	}
	return str
}

func NewHeader(host, path string) http.Header {
	header := http.Header{}
	header.Add("host", host)
	header.Add("method", "GET")
	header.Add("path", path)
	header.Add("scheme", "http")
	header.Add("accept", "*/*")
	header.Add("x-http2-version", Version)
	return header
}
