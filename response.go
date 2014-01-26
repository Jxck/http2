package http2

import (
	"bytes"
	"net/http"
)

type ResponseWriter struct {
	Headers *http.Header
	Body    *bytes.Buffer
}

func NewResponseWriter() ResponseWriter {
	buf := make([]byte, 1000)
	return ResponseWriter{
		Headers: new(http.Header),
		Body:    bytes.NewBuffer(buf),
	}
}

func (res ResponseWriter) Write(wire []byte) (int, error) {
	return res.Body.Write(wire)
}

func (res ResponseWriter) Header() http.Header {
	return *res.Headers
}

func (res ResponseWriter) WriteHeader(statusCode int) {
	// TODO: implement me
}
