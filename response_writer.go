package http2

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
)

type ResponseWriter struct {
	status int
	header http.Header
	body   *bytes.Buffer
}

func NewResponseWriter() *ResponseWriter {
	return &ResponseWriter{
		status: 0,
		header: make(http.Header),
		body:   bytes.NewBuffer([]byte{}),
	}
}

func (r *ResponseWriter) Header() http.Header {
	return r.header
}

func (r *ResponseWriter) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.WriteHeader(http.StatusOK)
	}
	return r.body.Write(b)
}

func (r *ResponseWriter) WriteHeader(status int) {
	r.status = status
}

func (r ResponseWriter) String() (str string) {
	str += fmt.Sprintf("HTTP/1.1 %d %s", r.status, http.StatusText(r.status))
	for name, value := range r.header {
		if strings.HasPrefix(name, ":") {
			continue
		}
		str += fmt.Sprintf("\n%s: %s", name, strings.Join(value, ","))
	}
	return str
}
