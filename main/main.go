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
	log.Println(fh.Decode(conn)) // setting
	// b = make([]byte, 36)
	// conn.Read(b)
	// log.Println(b)
	// [MAX_CONCURRENT_STREAMS(4):100]
	// [INITIAL_WINDOW_SIZE(7):65535]

	log.Println(fh.Decode(conn)) // window update
	log.Println(fh.Decode(conn)) // headers
	log.Println(fh.Decode(conn)) // data
	log.Println(fh.Decode(conn)) // data
	log.Println(fh.Decode(conn)) // data

	// TODO: Send GOAWAY
}
