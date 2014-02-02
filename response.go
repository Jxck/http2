package http2

import (
	"bytes"
	"net/http"
)

type ResponseWriter struct {
	status int
	header http.Header
	body   *bytes.Buffer
}

func NewResponseWriter() *ResponseWriter {
	return &ResponseWriter{
		status: 0,
		header: make(http.Header, 0),
		body:   bytes.NewBuffer([]byte{}),
	}
}

func (r *ResponseWriter) Header() http.Header {
	return r.header
}

func (r *ResponseWriter) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.body.Write(b)
}

func (r *ResponseWriter) WriteHeader(status int) {
	r.status = status
}
