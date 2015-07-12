package http2

import (
	"fmt"
	. "github.com/Jxck/color"
	"github.com/Jxck/hpack"
	. "github.com/Jxck/http2/frame"
	. "github.com/Jxck/logger"
	"io"
	"log"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

type Conn struct {
	RW           io.ReadWriter
	HpackContext *hpack.Context
	LastStreamID uint32
	Window       *Window
	Settings     map[SettingsID]int32
	PeerSettings map[SettingsID]int32
	Streams      map[uint32]*Stream
	WriteChan    chan Frame
	CallBack     func(stream *Stream)
}

func NewConn(rw io.ReadWriter) *Conn {
	conn := &Conn{
		RW:           rw,
		HpackContext: hpack.NewContext(uint32(DEFAULT_HEADER_TABLE_SIZE)),
		Settings:     DefaultSettings,
		PeerSettings: DefaultSettings,
		Window:       NewWindowDefault(),
		Streams:      make(map[uint32]*Stream),
		WriteChan:    make(chan Frame),
	}
	return conn
}

func (conn *Conn) NewStream(streamid uint32) *Stream {
	stream := NewStream(
		streamid,
		conn.WriteChan,
		conn.Settings,
		conn.PeerSettings,
		conn.HpackContext,
		conn.CallBack,
	)
	Debug("adding new stream (id=%d) total (%d)", stream.ID, len(conn.Streams))
	return stream
}

func (conn *Conn) HandleSettings(settingsFrame *SettingsFrame) {
	if settingsFrame.Flags == ACK {
		// receive ACK
		Trace("receive SETTINGS ACK")
		return
	}

	if settingsFrame.Flags != UNSET {
		Error("unknown flag of SETTINGS Frame %v", settingsFrame.Flags)
		return
	}

	// received SETTINGS Frame
	settings := settingsFrame.Settings

	defaultSettings := map[SettingsID]int32{
		SETTINGS_HEADER_TABLE_SIZE:      DEFAULT_HEADER_TABLE_SIZE,
		SETTINGS_ENABLE_PUSH:            DEFAULT_ENABLE_PUSH,
		SETTINGS_MAX_CONCURRENT_STREAMS: DEFAULT_MAX_CONCURRENT_STREAMS,
		SETTINGS_INITIAL_WINDOW_SIZE:    DEFAULT_INITIAL_WINDOW_SIZE,
		SETTINGS_MAX_FRAME_SIZE:         DEFAULT_MAX_FRAME_SIZE,
		SETTINGS_MAX_HEADER_LIST_SIZE:   DEFAULT_MAX_HEADER_LIST_SIZE,
	}

	// merge with default
	for k, v := range settings {
		defaultSettings[k] = v
	}

	Trace("merged settigns ============")
	for k, v := range defaultSettings {
		Trace("%v:%v", k, v)
	}
	Trace("merged settigns ============")

	// save settings to conn
	conn.Settings = defaultSettings

	// SETTINGS_INITIAL_WINDOW_SIZE
	initialWindowSize, ok := settings[SETTINGS_INITIAL_WINDOW_SIZE]
	if ok {

		if initialWindowSize > 2147483647 { // validate < 2^31-1
			Error("FLOW_CONTROL_ERROR (%s)", "SETTINGS_INITIAL_WINDOW_SIZE too large")
			return
		}

		conn.Window.PeerCurrentSize -= conn.Window.InitialSize
		conn.Window.PeerCurrentSize += initialWindowSize
		conn.Window.InitialSize = initialWindowSize
		conn.PeerSettings[SETTINGS_INITIAL_WINDOW_SIZE] = initialWindowSize

		for _, stream := range conn.Streams {
			log.Println("apply settings to stream", stream)
			stream.Window.PeerCurrentSize -= stream.Window.InitialSize
			stream.Window.PeerCurrentSize += initialWindowSize
			stream.Window.InitialSize = initialWindowSize
			stream.PeerSettings[SETTINGS_INITIAL_WINDOW_SIZE] = initialWindowSize
		}
	}

	// send ACK
	ack := NewSettingsFrame(ACK, 0, NilSettings)
	conn.WriteChan <- ack
}

func (conn *Conn) ReadLoop() {
	Debug("start conn.ReadLoop()")
	for {
		// コネクションからフレームを読み込む
		frame, err := ReadFrame(conn.RW, conn.Settings)
		if err != nil {
			Error("%v", err)
			h2Error, ok := err.(*H2Error)
			Debug("%v", ok)
			if ok {
				conn.GoAway(0, h2Error)
			}
			break
		}
		if frame != nil {
			Notice("%v %v", Green("recv"), util.Indent(frame.String()))
		}

		streamID := frame.Header().StreamID
		types := frame.Header().Type

		// CONNECTION LEVEL
		if streamID == 0 {
			if types == DataFrameType ||
				types == HeadersFrameType ||
				types == PriorityFrameType ||
				types == RstStreamFrameType ||
				types == PushPromiseFrameType ||
				types == ContinuationFrameType {

				msg := fmt.Sprintf("%s FRAME for Stream ID 0", types)
				Error("%v", msg)
				conn.GoAway(0, &H2Error{PROTOCOL_ERROR, msg})
			}

			// SETTINGS frame を受け取った場合
			if types == SettingsFrameType {
				settingsFrame, ok := frame.(*SettingsFrame)
				if !ok {
					Error("invalid settings frame %v", frame)
					return
				}
				conn.HandleSettings(settingsFrame)
			}

			// Connection Level Window Update
			if types == WindowUpdateFrameType {
				windowUpdateFrame, ok := frame.(*WindowUpdateFrame)
				if !ok {
					Error("invalid window update frame %v", frame)
					return
				}
				conn.Window.PeerCurrentSize += int32(windowUpdateFrame.WindowSizeIncrement)
			}

			// handle GOAWAY with close connection
			if types == GoAwayFrameType {
				Debug("stop conn.ReadLoop() by GOAWAY")
				break
			}
		}

		// STREAM LEVEL
		if streamID > 0 {
			if types == SettingsFrameType ||
				types == PingFrameType ||
				types == GoAwayFrameType {

				msg := fmt.Sprintf("%s FRAME for Stream ID not 0", types)
				Error("%v", msg)
				conn.GoAway(0, &H2Error{PROTOCOL_ERROR, msg})
			}

			// DATA frame なら winodw update
			if types == DataFrameType {
				length := int32(frame.Header().Length)
				conn.WindowUpdate(length)
			}

			// 新しいストリーム ID なら対応するストリームを生成
			stream, ok := conn.Streams[streamID]
			if !ok {
				// create stream with streamID
				stream = conn.NewStream(streamID)
				conn.Streams[streamID] = stream

				// update last stream id
				if streamID > conn.LastStreamID {
					conn.LastStreamID = streamID
				}
			}

			// stream の state を変える
			err = stream.ChangeState(frame, RECV)
			if err != nil {
				Error("%v", err)
				h2Error, ok := err.(*H2Error)
				if ok {
					conn.GoAway(0, h2Error)
				}
				break
			}

			// stream が close ならリストから消す
			if stream.State == CLOSED {
				Info("remove stream(%d) from conn.Streams[]", streamID)
				conn.Streams[streamID] = nil
			}

			// ストリームにフレームを渡す
			stream.ReadChan <- frame
		}
	}

	Debug("stop the readloop")
}

func (conn *Conn) WriteLoop() (err error) {
	Debug("start conn.WriteLoop()")
	for frame := range conn.WriteChan {
		Notice("%v %v", Red("send"), util.Indent(frame.String()))

		// TODO: ここで WindowSize を見る
		err = frame.Write(conn.RW)
		if err != nil {
			Error("%v", err)
			return err
		}
	}
	return
}

func (conn *Conn) GoAway(streamId uint32, h2Error *H2Error) {
	Debug("connection close with GO_AWAY(%v)", h2Error)
	errorCode := h2Error.ErrorCode
	additionalDebugData := []byte(h2Error.AdditiolanDebugData)
	goaway := NewGoAwayFrame(streamId, conn.LastStreamID, errorCode, additionalDebugData)
	conn.WriteChan <- goaway
}

func (conn *Conn) WindowUpdate(length int32) {
	Debug("connection window update %d byte", length)

	conn.Window.CurrentSize = conn.Window.CurrentSize - length

	// この値を下回ったら WindowUpdate を送る
	if conn.Window.CurrentSize < conn.Window.Threshold {
		update := conn.Window.InitialSize - conn.Window.CurrentSize
		conn.WriteChan <- NewWindowUpdateFrame(0, uint32(update))
		conn.Window.CurrentSize = conn.Window.CurrentSize + update
	}
}

func (conn *Conn) WriteMagic() (err error) {
	_, err = conn.RW.Write([]byte(CONNECTION_PREFACE))
	if err != nil {
		return err
	}
	Info("%v %q", Red("send"), CONNECTION_PREFACE)
	return
}

func (conn *Conn) ReadMagic() (err error) {
	magic := make([]byte, len(CONNECTION_PREFACE))
	_, err = conn.RW.Read(magic)
	if err != nil {
		return err
	}
	if string(magic) != CONNECTION_PREFACE {
		Info("Invalid Magic String: %q", string(magic))
		return fmt.Errorf("Invalid Magic String")
	}
	Info("%v %q", Green("recv"), string(magic))
	return
}

func (conn *Conn) Close() {
	Info("close all conn.Streams")
	for i, stream := range conn.Streams {
		if stream != nil {
			Debug("close stream(%d)", i)
			stream.Close()
		}
	}
	Info("close conn.WriteChan")
	close(conn.WriteChan)
}
