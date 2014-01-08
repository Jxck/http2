package http2

import (
	"fmt"
	. "github.com/jxck/color"
	. "github.com/jxck/logger"
	"log"
	"net"
	"net/http"
)

const Version string = "HTTP-draft-09/2.0"
const MagicString string = "PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n"
const DefaultSettingsBase64 string = "AAAABAAAAGQAAAAHAAD__w"

func init() {
	log.SetFlags(log.Lshortfile)
}

type Transport struct {
	URL     *URL
	Conn    *Conn
	Upgrade bool
	FlowCtl bool
}

func (transport *Transport) Connect() {
	var conn net.Conn
	if transport.URL.Scheme == "http" {
		address := transport.URL.Host + ":" + transport.URL.Port
		conn, _ = net.Dial("tcp", address) // err
	} else {
		Error("not support yet")
	}

	transport.Conn = NewConn(conn)
}

func (transport *Transport) SendUpgrade() *Stream {
	upgrade := fmt.Sprintf(""+
		"GET %s HTTP/1.1\r\n"+
		"Host: %s\r\n"+
		"Connection: Upgrade, HTTP2-Settings\r\n"+
		"Upgrade: %s\r\n"+
		"HTTP2-Settings: %s\r\n"+
		"Accept: */*\r\n"+
		"\r\n",
		transport.URL.Path,
		transport.URL.Host,
		Version,
		DefaultSettingsBase64)

	transport.Conn.WriteString(upgrade)
	res := transport.Conn.ReadResponse()

	if res.StatusCode != 101 {
		Error("error")
	}
	Info(Yellow("HTTP Upgrade Success :)"))

	stream := transport.Conn.NewStream()
	return stream
}

func (transport *Transport) SendMagic() {
	transport.Conn.WriteString(MagicString) // err
}

func (transport *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	transport.URL, _ = NewURL(req.URL.String())
	transport.Connect()

	settings := map[SettingsId]uint32{
		SETTINGS_MAX_CONCURRENT_STREAMS: 100,
		SETTINGS_INITIAL_WINDOW_SIZE:    DEFAULT_WINDOW_SIZE,
	}

	var stream *Stream
	if transport.Upgrade {
		stream = transport.SendUpgrade()
		transport.SendMagic()
		transport.Conn.SendSettings(settings) // err
	} else {
		transport.SendMagic()
		if !transport.FlowCtl {
			settings[SETTINGS_FLOW_CONTROL_OPTIONS] = 1
		}
		transport.Conn.SendSettings(settings) // err
		req = UpdateRequest(req, transport.URL)
		stream = transport.Conn.NewStream()
		stream.SendRequest(req)
	}

	res := stream.RecvResponse() // err
	transport.Conn.SendGoAway(NO_ERROR)
	return res, nil
}
