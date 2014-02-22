package http2

import (
	"crypto/tls"
	. "github.com/jxck/color"
	. "github.com/jxck/http2/frame"
	. "github.com/jxck/logger"
	"log"
	"net/http"
	"time"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

// Transport implements http.RoundTriper
// with RoundTrip(request) response
type Transport struct {
	URL      *URL
	Conn     *Conn
	CertPath string
	KeyPath  string
}

// connect tcp connection with host
func (transport *Transport) Connect() {
	address := transport.URL.Host + ":" + transport.URL.Port

	// loading key pair
	cert, err := tls.LoadX509KeyPair(transport.CertPath, transport.KeyPath)
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

	// send Magic Octet
	transport.Conn.WriteMagic()

	// send settings
	settingsFrame := NewSettingsFrame(0 /*flags*/, DefaultSettings, 0 /*stream id*/)
	transport.Conn.WriteFrame(settingsFrame) // err
}

// http.RoundTriper implementation
func (transport *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	transport.URL, _ = NewURL(req.URL.String()) // err

	// establish tcp connection and handshake
	transport.Connect()

	// create stream
	stream := transport.Conn.NewStream(<-NextClientStreamId)
	log.Println(stream)
	req = util.UpdateRequest(req, transport.URL)

	// send request header via HEADERS Frame
	var flags uint8 = END_STREAM + END_HEADERS
	frame := NewHeadersFrame(flags, stream.Id)
	frame.Headers = req.Header
	frame.HeaderBlock = transport.Conn.EncodeHeader(frame.Headers)
	frame.Length = uint16(len(frame.HeaderBlock))
	stream.Write(frame) // err

	time.Sleep(time.Second)
	//stream.SendRequest(req)

	//// receive response from stream
	//res := stream.RecvResponse() // err

	//// send GOAWAY
	//transport.Conn.SendGoAway(NO_ERROR)

	//return res, nil
	return nil, nil
}
