package http2

import (
	"bytes"
	"net/http"
)

type Response struct {
	Headers http.Header
	Body    *bytes.Buffer
}

func (res *Response) Write(wire []byte) (int, error) {
	return res.Body.Write(wire)
}

func (res *Response) Header() http.Header {
	return res.Headers
}

func (res *Response) WriteHeader(statusCode int) {
	// TODO: implement me
}
