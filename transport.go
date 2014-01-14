package http2

import (
	"crypto/tls"
	"fmt"
	. "github.com/jxck/color"
	. "github.com/jxck/http2/frame"
	. "github.com/jxck/logger"
	"log"
	"net"
	"net/http"
)

const (
	Version               = "HTTP-draft-09/2.0"
	MagicString           = "PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n"
	DefaultSettingsBase64 = "AAAABAAAAGQAAAAHAAD__w"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

// Transport implements http.RoundTriper
// with RoundTrip(request) response
type Transport struct {
	URL     *URL
	Conn    *Conn
	Upgrade bool
	FlowCtl bool
}

// dial tls connection with NPN
func DialNPN(address, certpath, keypath string) *tls.Conn {
	// loading key pair
	cert, err := tls.LoadX509KeyPair(certpath, keypath)
	if err != nil {
		log.Fatal(err)
	}

	// setting TLS config
	config := tls.Config{
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true,
		NextProtos:         []string{Version},
	}
	conn, err := tls.Dial("tcp", address, &config)
	if err != nil {
		log.Fatal(err)
	}

	// check connection state
	state := conn.ConnectionState()
	Info("%v %v", Yellow("handshake"), state.HandshakeComplete)
	Info("%v %v", Yellow("protocol"), state.NegotiatedProtocol)

	return conn
}

// connect tcp connection with host
func (transport *Transport) Connect() {
	var conn net.Conn
	address := transport.URL.Host + ":" + transport.URL.Port
	if transport.URL.Scheme == "http" {
		conn, _ = net.Dial("tcp", address) // err
	} else {
		conn = DialNPN(address, DEFAULT_CERT, DEFAULT_KEY)
	}
	transport.Conn = NewConn(conn)
}

// send http upgrade header
func (transport *Transport) SendUpgrade() *Stream {
	// HTTP/1.1 Upgrade Header
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
		Error(Red("faild to Upgrade :("))
	}
	Info(Yellow("HTTP Upgrade Success :)"))

	stream := transport.Conn.NewStream()
	return stream
}

// send magic octet
func (transport *Transport) SendMagic() {
	transport.Conn.WriteString(MagicString) // err
}

// http.RoundTriper implementation
func (transport *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	transport.URL, _ = NewURL(req.URL.String()) // err

	// establish tcp connection
	transport.Connect()

	// default Settings
	settings := map[SettingsId]uint32{
		SETTINGS_MAX_CONCURRENT_STREAMS: 100,
		SETTINGS_INITIAL_WINDOW_SIZE:    DEFAULT_WINDOW_SIZE,
	}

	var stream *Stream // create stream
	if transport.Upgrade {
		// using http upgrade
		stream = transport.SendUpgrade()
		transport.SendMagic()
		transport.Conn.SendSettings(settings) // err
	} else {
		// using NPN
		transport.SendMagic()
		if !transport.FlowCtl {
			settings[SETTINGS_FLOW_CONTROL_OPTIONS] = 1
		}
		transport.Conn.SendSettings(settings) // err
		req = UpdateRequest(req, transport.URL)
		stream = transport.Conn.NewStream()
		stream.SendRequest(req)
	}

	// receive response from stream
	res := stream.RecvResponse() // err

	// send GOAWAY
	transport.Conn.SendGoAway(NO_ERROR)

	return res, nil
}
