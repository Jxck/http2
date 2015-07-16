package main

import (
	"bufio"
	"bytes"
	"fmt"
	. "github.com/Jxck/color"
	"log"
	"os"
	"regexp"
	"runtime/pprof"
	"strconv"
	"strings"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

func main() {
	handleConn()
}

type Stream struct {
	ID       int
	ReadChan chan *Frame
}

func (s *Stream) ReadLoop() {
	req := ""
	for {
		select {
		case frame := <-s.ReadChan:
			if frame.Type == "headers" {
				req += "header"
			}
			if frame.Type == "data" {
				req += "body"
			}

			if len(frame.Flag) > 0 && frame.Flag[0] == "endstream" {
				res := req
				log.Println(res)
			}
		}
	}
}

func NewStream(id int) *Stream {
	stream := &Stream{id, make(chan *Frame)}
	go stream.ReadLoop()
	return stream
}

type Conn struct {
	reader  *bufio.Reader
	writer  *bufio.Writer
	Streams map[int]*Stream
}

func NewConn() *Conn {
	return &Conn{
		reader:  bufio.NewReader(os.Stdin),
		writer:  bufio.NewWriter(os.Stdout),
		Streams: make(map[int]*Stream),
	}
}

type Frame struct {
	ID   int
	Type string
	Flag []string
}

func (conn *Conn) ReadFrame() (*Frame, error) {
	line, err := conn.reader.ReadString('\n')
	if line == "" {
		return nil, nil
	}

	header := strings.Split(strings.TrimSpace(line), " ")
	id, _ := strconv.Atoi(header[0])
	types := header[1]

	flag := header[2:]
	frame := &Frame{id, types, flag}

	fmt.Printf("< %+v\n", frame)
	return frame, err
}

func (conn *Conn) WriteFrame(frame *Frame) (err error) {
	fmt.Printf("\t\t\t> %#v\n", frame)
	return err
}

func (conn *Conn) GetStream(id int) *Stream {
	stream, ok := conn.Streams[id]
	if !ok {
		stream = NewStream(id)
		conn.Streams[id] = stream
	}
	return stream
}

func handleConn() {
	Conn := NewConn()

	var frameChan = make(chan *Frame)
	go func() {
		for {
			frame, err := Conn.ReadFrame()
			if frame == nil {
				continue
			}
			if err != nil {
				break
			}
			frameChan <- frame
		}
	}()

	for {
		select {
		case frame := <-frameChan:
			if frame.ID > 0 {
				stream := Conn.GetStream(frame.ID)
				stream.ReadChan <- frame
			}
		}

		// Conn.WriteFrame(frame)
	}
}

func dump() {
	stack := new(bytes.Buffer)
	pprof.Lookup("goroutine").WriteTo(stack, 10)
	fmt.Println(Brown(stack))
	re := regexp.MustCompile(`goroutine (\d+)(.+)\n(.+)`)
	matches := re.FindAllSubmatch(stack.Bytes(), -1)

	for _, match := range matches {
		str := string(match[3])
		if false && strings.HasPrefix(str, "runtime") {
			continue
		}
		// fmt.Printf(Red("go(%s)\t%s  %s\n"), match[1], match[2], str)
	}
}
