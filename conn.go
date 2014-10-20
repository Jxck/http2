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
	WindowSize   uint32
	Settings     map[SettingsID]uint32
	Streams      map[uint32]*Stream
	WriteChan    chan Frame
	CallBack     func(stream *Stream)
}

func NewConn(rw io.ReadWriter) *Conn {
	conn := &Conn{
		RW:           rw,
		HpackContext: hpack.NewContext(DEFAULT_HEADER_TABLE_SIZE),
		Settings:     DefaultSettings,
		WindowSize:   DefaultSettings[SETTINGS_INITIAL_WINDOW_SIZE],
		Streams:      make(map[uint32]*Stream),
		WriteChan:    make(chan Frame),
	}
	return conn
}

func (conn *Conn) NewStream(streamid uint32) *Stream {
	stream := NewStream(
		streamid,
		conn.WriteChan,
		DefaultSettings[SETTINGS_INITIAL_WINDOW_SIZE],
		conn.HpackContext,
		conn.CallBack,
	)
	Debug("adding new stream (id=%d) total (%d)", stream.ID, len(conn.Streams))
	return stream
}

func (conn *Conn) ReadLoop() {
	Debug("start conn.ReadLoop()")
	for {
		// コネクションからフレームを読み込む
		frame, err := ReadFrame(conn.RW)
		if err != nil {
			if err == io.EOF {
				Error("%v", err)
				break
			}
			Fatal("%v", err)
		}
		if frame != nil {
			Notice("%v %v", Green("recv"), util.Indent(frame.String()))
		}

		// 新しいストリーム ID なら対応するストリームを生成
		streamID := frame.Header().StreamID
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
		}

		// DATA frame なら winodw update
		if frame.Header().Type == DataFrameType {
			conn.WindowUpdate(frame.Header().Length)
		}

		// stream が close ならリストから消す
		if stream.State == CLOSED {
			Info("remove stream(%d) from conn.Streams[]", streamID)
			conn.Streams[streamID] = nil
		}

		// handle GOAWAY with close connection
		if frame.Header().Type == GoAwayFrameType {
			Debug("stop conn.ReadLoop() by GOAWAY")
			conn.Close()
			break
		}

		// ストリームにフレームを渡す
		stream.ReadChan <- frame
	}
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

func (conn *Conn) WindowUpdate(length uint32) {
	Debug("connection window update %d byte", length)

	total := conn.WindowSize

	total = total - length
	if total < WINDOW_UPDATE_THRESHOLD {
		// この値を下回ったら WindowUpdate を送る
		update := conn.WindowSize - total
		conn.WriteChan <- NewWindowUpdateFrame(0, update)
	} else {
		conn.WindowSize = total
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
		Error("Invalid Magic String")
		return fmt.Errorf("Invalid Magic String")
	}
	Info("%v %q", Red("recv"), string(magic))
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
