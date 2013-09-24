package main

import (
	"github.com/jxck/hpack"
	"log"
	"net"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

func main() {
	conn, _ := net.Dial("tcp", "106.186.112.116:80") // err

	conn.Write([]byte("GET / HTTP/1.1\r\n"))                         // err
	conn.Write([]byte("Host: 106.186.112.116:80\r\n"))               // err
	conn.Write([]byte("Connection: Upgrade, HTTP2-Settings\r\n"))    // err
	conn.Write([]byte("Upgrade: HTTP-draft-06/2.0\r\n"))             // err
	conn.Write([]byte("HTTP2-Settings: AAAABAAAAGQAAAAHAAD__w\r\n")) // err
	conn.Write([]byte("Accept: */*\r\n"))                            // err
	conn.Write([]byte("\r\n"))                                       // err

	b := make([]byte, 76)
	conn.Read(b)
	// Upgrade Success :)

	conn.Write([]byte("PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n")) // err

	//0 16 4 0
	//0  0 0 0
	//0  0 0 4
	//     100
	//0  0 0 7
	//   65535
	// write setting frame
	conn.Write([]byte{0x0, 0x10, 0x4, 0x0,
		0x0, 0x0, 0x0, 0x0,
		0x0, 0x0, 0x0, 0x4,
		0x0, 0x0, 0x0, 0x64,
		0x0, 0x0, 0x0, 0x7,
		0x0, 0x0, 0xFF,
	}) // err

	// fh := &http2.FrameHeader{}
	// fh.Decode(conn)
	b = make([]byte, 36)
	conn.Read(b)
	// log.Println(b)
	/*
	   ====
	   0 16 4 0 (length=16, type=4, flag=0) Settings Frame
	   0 0 0 0 (R=0, StreamId=0)

	   0 0 0 4 (Reserved=0, SettingId=4)
	   0 0 0 100 SETTING_MAX_CONCURRENT_STREAM=100

	   0 0 0 7
	   0 0 255 255 SETTING_INITIAL_WINDOW_SIZE=65535
	   ====
	   0 4 9 0 (length=4, type=9, flag=0) Window Update Frame
	   0 0 0 0 (StreamId=0)
	   59 154 202 7
	   00111011, 10011010, 11001010, 00000111=1000000007
	   ====
	*/

	// [MAX_CONCURRENT_STREAMS(4):100]
	// [INITIAL_WINDOW_SIZE(7):65535]

	// read next header
	b = make([]byte, 8)
	n, err := conn.Read(b)
	log.Println(n, err, b)

	// read next payload
	length := b[1]
	b = make([]byte, length)
	n, err = conn.Read(b)
	log.Println(length, n, err)

	// unpack
	server := hpack.NewResponseContext()
	server.Decode(b)

	var data string
	var l, m uint64

	for i := 0; i < 4; i++ {
		// read next header
		b = make([]byte, 8)
		n, err = conn.Read(b)
		log.Println(n, err)

		if b[3] == 1 {
			log.Println("last")
			break
		}

		l = uint64(b[0])
		l <<= 8
		l += uint64(b[1])
		log.Println("length of data", l)

		// read Data
		b = make([]byte, l)
		n, err = conn.Read(b)
		log.Println(n, err)
		data += string(b)
		m = uint64(n)

		for m < l {
			l -= m
			b = make([]byte, l)
			n, err = conn.Read(b)
			log.Println(n, err)
			data += string(b)
			m = uint64(n)
		}
	}
	// log.Println(data)

	// TODO: Send GOAWAY
}
