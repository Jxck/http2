package http2

import (
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
	Conn         *Conn
	Upgrade      bool
}

func (transport *Transport) Connect() {
	var conn net.Conn
	if transport.URL.Scheme == "http" {
		address := transport.URL.Host + ":" + transport.URL.Port
		conn, _ = net.Dial("tcp", address) // err
	} else {
		log.Fatal("not support yet")
	}

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

	transport.Conn.WriteString(upgrade)
	res := transport.Conn.ReadResponse()

	if res.StatusCode != 101 {
		log.Fatal("error")
	}
	fmt.Println(Yellow("HTTP Upgrade Success :)"))

	stream := transport.NewStream()
	return stream
}

func (transport *Transport) SendMagic() {
	transport.Conn.WriteString(MagicString) // err
	fmt.Println(Yellow("Send MagicOctet"))
}

func (transport *Transport) NewStream() *Stream {
	if transport.LastStreamId == 0 {
		transport.LastStreamId = 1
	} else {
		transport.LastStreamId += 2
	}
	stream := &Stream{
		Id:   transport.LastStreamId, // TODO: transport.GetNextID()
		Conn: transport.Conn,
	}
	return stream
}

func (transport *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	transport.URL, _ = NewURL(req.URL.String())
	transport.Connect()

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

	res := stream.ReadResponse() // err
	return res, nil
}
