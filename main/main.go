package main

import (
	"github.com/jxck/http2"
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

	fh := &http2.FrameHeader{}
	fh.Decode(conn) // setting
	// b = make([]byte, 36)
	// conn.Read(b)
	// log.Println(b)
	// [MAX_CONCURRENT_STREAMS(4):100]
	// [INITIAL_WINDOW_SIZE(7):65535]

	fh.Decode(conn) // window update
	fh.Decode(conn) // headers

	var data string
	var l, m uint64

	for i := 0; i < 4; i++ {
		// read next header
		b = make([]byte, 8)
		n, err := conn.Read(b)
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
