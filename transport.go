package http2

import (
	"crypto/tls"
	"fmt"
	. "github.com/jxck/color"
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

/**
 * Transport implements http.RoundTriper
 * with RoundTrip(request) response
 */
type Transport struct {
	URL     *URL
	Conn    *Conn
	Upgrade bool
	FlowCtl bool
}

// NPN Dial
func DialNPN(address, certpath, keypath string) *tls.Conn {
	// 証明書の設定
	cert, err := tls.LoadX509KeyPair(certpath, keypath)
	if err != nil {
		log.Fatal(err)
	}

	// TLS の設定(証明書検証無)
	config := tls.Config{
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true,
		NextProtos:         []string{"http2.0/draft-09"},
	}
	conn, err := tls.Dial("tcp", address, &config)
	if err != nil {
		log.Fatal(err)
	}

	// 接続確認
	state := conn.ConnectionState()
	log.Println("handshake: ", state.HandshakeComplete)
	log.Println("protocol: ", state.NegotiatedProtocol)

	return conn
}

// connect tcp connection with host
func (transport *Transport) Connect() {
	var conn net.Conn
	address := transport.URL.Host + ":" + transport.URL.Port
	if transport.URL.Scheme == "http" {
		conn, _ = net.Dial("tcp", address) // err
	} else {
		// TODO: move to arg
		certpath, keypath := "keys/cert.pem", "keys/key.pem"
		conn = DialNPN(address, certpath, keypath)
	}
	transport.Conn = NewConn(conn)
}

// send http upgrade header
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

// send magic octet
func (transport *Transport) SendMagic() {
	transport.Conn.WriteString(MagicString) // err
}

// http.RoundTriper implementation
func (transport *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	transport.URL, _ = NewURL(req.URL.String()) // err

	// establish tcp connection
	transport.Connect()

	// Default Settings
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

	//  send GOAWAY
	transport.Conn.SendGoAway(NO_ERROR)

	return res, nil
}
