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
	URL          *URL
	Bw           *bufio.Writer
	Br           *bufio.Reader
	Conn         *Conn
	Upgrade      bool
}

func (transport *Transport) Connect(url string) {
	var conn net.Conn
	if transport.URL.Scheme == "http" {
		conn, _ = net.Dial("tcp", transport.URL.Host+":"+transport.URL.Port) // err
	} else {
		log.Fatal("not support yet")
	}

	transport.Bw = bufio.NewWriter(conn)
	transport.Br = bufio.NewReader(conn)
	transport.Conn = NewConn(conn)
}

func (transport *Transport) SendUpgrade() *Stream {
	upgrade := "" +
		"GET " + transport.URL.Path + " HTTP/1.1\r\n" +
		"Host: " + transport.URL.Host + "\r\n" +
		"Connection: Upgrade, HTTP2-Settings\r\n" +
		"Upgrade: " + Version + "\r\n" +
		"HTTP2-Settings: " + DefaultSettingsBase64 + "\r\n" +
		"Accept: */*\r\n" +
		"\r\n"

	transport.Bw.WriteString(upgrade) // err
	transport.Bw.Flush()              // err
	fmt.Println(Blue(upgrade))

	res, _ := http.ReadResponse(transport.Br, &http.Request{Method: "GET"}) // err

	fmt.Println(Blue(ResponseString(res)))
	fmt.Println(Yellow("HTTP Upgrade Success :)"))
	stream := &Stream{
		Id:   1,
		Conn: transport.Conn,
	}
	return stream
}

func (transport *Transport) SendMagic() {
	transport.Bw.WriteString(MagicString) // err
	transport.Bw.Flush()                  // err
	fmt.Println(Yellow("Send MagicOctet"))
}

func (transport *Transport) NewStream() *Stream {
	stream := &Stream{
		Id:   transport.LastStreamId, // TODO: transport.GetNextID()
		Conn: transport.Conn,
	}
	transport.LastStreamId += 2
	return stream
}

func (transport *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	transport.URL, _ = NewURL(req.URL.String())
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
