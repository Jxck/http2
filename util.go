package http2

import (
	"fmt"
	"log"
	"net/http"
	"strings"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

func RequestString(req *http.Request) string {
	str := fmt.Sprintf("%v %v %v", req.Method, req.URL, req.Proto)
	for k, v := range req.Header {
		str += fmt.Sprintf("\n%v: %v", k, v[0])
	}
	return str
}

func ResponseString(res *http.Response) string {
	str := fmt.Sprintf("%v %v", res.Proto, res.Status)
	for k, v := range res.Header {
		str += fmt.Sprintf("\n%v: %v", k, v[0])
	}
	return str
}

func Indent(v interface{}) string {
	return strings.Replace(fmt.Sprintf("%v", v), "\n", "\n\t\t\t\t", -1)
}
