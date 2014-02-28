package http2

import (
	"bytes"
)

type Body struct {
	bytes.Buffer
}

func (b *Body) Close() error {
	return nil
}
