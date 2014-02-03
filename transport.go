package http2

import (
	"crypto/tls"
	. "github.com/jxck/color"
	. "github.com/jxck/http2/frame"
	. "github.com/jxck/logger"
	"log"
	"net/http"
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

// connect tcp connection with host
func (transport *Transport) Connect() {
	address := transport.URL.Host + ":" + transport.URL.Port

	// TODO: move to args
	certpath := "keys/cert.pem"
	keypath := "keys/key.pem"

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

	transport.Conn = NewConn(conn)
}

// send magic octet
func (transport *Transport) SendMagic() {
	transport.Conn.WriteMagic() // err
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
	transport.SendMagic()
	if !transport.FlowCtl {
		settings[SETTINGS_FLOW_CONTROL_OPTIONS] = 1
	}
	transport.Conn.SendSettings(settings) // err
	req = transport.URL.Update(req)
	stream = transport.Conn.NewStream()
	stream.SendRequest(req)

	// receive response from stream
	res := stream.RecvResponse() // err

	// send GOAWAY
	transport.Conn.SendGoAway(NO_ERROR)

	return res, nil
}
