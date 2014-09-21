package frame

import (
	"encoding/binary"
	"fmt"
	"io"
)

func MustWrite(w io.Writer, data interface{}) {
	err := binary.Write(w, binary.BigEndian, data)
	if err != nil {
		panic(err)
	}
}

func MustRead(r io.Reader, data interface{}) {
	err := binary.Read(r, binary.BigEndian, data)
	if err != nil {
		panic(err)
	}
}

func Recovery(r interface{}) error {
	if r == nil {
		return nil
	}

	switch e := r.(type) {
	case error:
		return e
	default:
		return fmt.Errorf("%v", e)
	}
}
