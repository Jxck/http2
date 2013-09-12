package main

import (
	"encoding/base64"
	"log"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

func main() {
	// [0 8 4 0, 0 0 0 0, 0 0 0 4, 127 255 255 255]
	buf := []byte{
		0, 8, 4, 0,
		// 00000000 00001000,00000100,00000000
		0, 0, 0, 0,
		// 00000000 00000000 00000000 00000000
		0, 0, 0, 4,
		// 00000000 00000000 00000000 00000100
		0, 0, 0, 0xc4,
		// 00000000 00000000 00000000 11000100
	}
	str := base64.StdEncoding.EncodeToString(buf)

	log.Printf("%v", str)
}
