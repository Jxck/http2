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

func UpdateRequest(req *http.Request, url *URL) *http.Request {
	req.Header.Add("host", url.Host)
	req.Header.Add("method", req.Method)
	req.Header.Add("path", url.Path)
	req.Header.Add("scheme", url.Scheme)
	req.Header.Add("accept", "*/*")

	if req.ContentLength != 0 {
		req.Header.Add("content-length", fmt.Sprintf("%d", req.ContentLength))
	}
	req.Header.Add("x-http2-version", Version)
	return req
}
