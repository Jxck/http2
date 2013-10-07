package http2

import (
	"bufio"
	"fmt"
	. "github.com/jxck/color"
	"log"
	"net"
	"net/http"
)

const Version string = "HTTP-draft-06/2.0"
const MagicString string = "PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n"
const DefaultSettingsBase64 string = "AAAABAAAAGQAAAAHAAD__w"

func init() {
	log.SetFlags(log.Lshortfile)
}

type Transport struct {
	LastStreamId uint32
	url          *URL
	bw           *bufio.Writer
	br           *bufio.Reader
	conn         *Conn
	Upgrade      bool
}

func (transport *Transport) Connect(url string) {
	var conn net.Conn
	if transport.url.Scheme == "http" {
		conn, _ = net.Dial("tcp", transport.url.Host+":"+transport.url.Port) // err
	} else {
		log.Fatal("not support yet")
	}

	transport.bw = bufio.NewWriter(conn)
	transport.br = bufio.NewReader(conn)
	transport.conn = NewConn(conn)
}

func (transport *Transport) SendUpgrade() *Stream {
	upgrade := "" +
		"GET " + transport.url.Path + " HTTP/1.1\r\n" +
		"Host: " + transport.url.Host + "\r\n" +
		"Connection: Upgrade, HTTP2-Settings\r\n" +
		"Upgrade: " + Version + "\r\n" +
		"HTTP2-Settings: " + DefaultSettingsBase64 + "\r\n" +
		"Accept: */*\r\n" +
		"\r\n"

	transport.bw.WriteString(upgrade) // err
	transport.bw.Flush()              // err
	fmt.Println(Blue(upgrade))

	res, _ := http.ReadResponse(transport.br, &http.Request{Method: "GET"}) // err

	fmt.Println(Blue(ResponseString(res)))
	fmt.Println(Yellow("HTTP Upgrade Success :)"))
	stream := &Stream{
		Id:   1,
		Conn: transport.conn,
	}
	return stream
}

func (transport *Transport) SendMagic() {
	transport.bw.WriteString(MagicString) // err
	transport.bw.Flush()                  // err
	fmt.Println(Yellow("Send MagicOctet"))
}

func (transport *Transport) NewStream() *Stream {
	stream := &Stream{
		Id:   transport.LastStreamId, // TODO: transport.GetNextID()
		Conn: transport.conn,
	}
	transport.LastStreamId += 2
	return stream
}

func (transport *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	transport.url, _ = NewURL(req.URL.String())
	transport.Connect(req.URL.String())

	var stream *Stream
	if transport.Upgrade {
		stream = transport.SendUpgrade()
		transport.SendMagic()
		settings := map[SettingsId]uint32{
			SETTINGS_MAX_CONCURRENT_STREAMS: 100,
			SETTINGS_INITIAL_WINDOW_SIZE:    65535,
		}
		stream.Send(NewSettingsFrame(settings, 0)) // err
	} else {
		transport.SendMagic()
		stream = transport.NewStream()
		settings := map[SettingsId]uint32{
			SETTINGS_MAX_CONCURRENT_STREAMS: 100,
			SETTINGS_INITIAL_WINDOW_SIZE:    65535,
			SETTINGS_FLOW_CONTROL_OPTIONS:   1,
		}
		stream.Send(NewSettingsFrame(settings, 0)) // err
		stream.SendRequest(req)
	}

	res := stream.ReadResponse()
	return res, nil
}
