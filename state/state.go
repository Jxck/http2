package state

import (
	"fmt"
	. "github.com/jxck/http2/frame"
	"log"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

type State interface {
	H(*HeadersFrame) (State, error)
	ES(Frame) (State, error)
	R(*RstStreamFrame) (State, error)
	String() string
}

/**
* Idle
*/
type Idle struct {
}

func NewIdle() State {
	s := new(Idle)
	fmt.Println(s)
	return s
}

func (s Idle) String() string {
	return "Idle"
}

func (s *Idle) H(frame *HeadersFrame) (State, error) {
	return NewOpen(), nil
}

func (s *Idle) ES(frame Frame) (State, error) {
	return nil, fmt.Errorf("illegal state %s at %s", frame, s)
}

func (s *Idle) R(frame *RstStreamFrame) (State, error) {
	return nil, fmt.Errorf("illegal state %s at %s", frame, s)
}

/**
* Open
*/
type Open struct {
}

func NewOpen() State {
	s := new(Open)
	fmt.Println(s)
	return s
}

func (s Open) String() string {
	return "Open"
}

func (s *Open) H(frame *HeadersFrame) (State, error) {
	flags := frame.Header().Flags
	if flags&END_STREAM == END_STREAM {
		return NewClosed(), nil
	}
	return nil, fmt.Errorf("illegal state %s at %s", frame, s)
}

func (s *Open) ES(frame Frame) (State, error) {
	return NewHalfClosed(), nil
}

func (s *Open) R(frame *RstStreamFrame) (State, error) {
	return NewClosed(), nil
}

/**
* Closed
*/
type Closed struct {
}

func NewClosed() State {
	s := new(Closed)
	fmt.Println(s)
	return s
}

func (s Closed) String() string {
	return "Closed"
}

func (s *Closed) H(frame *HeadersFrame) (State, error) {
	return nil, fmt.Errorf("illegal state %s at %s", frame, s)
}

func (s *Closed) ES(frame Frame) (State, error) {
	return nil, fmt.Errorf("illegal state %s at %s", frame, s)
}

func (s *Closed) R(frame *RstStreamFrame) (State, error) {
	return nil, fmt.Errorf("illegal state %s at %s", frame, s)
}

/**
* HalfClosed
*/
type HalfClosed struct {
}

func NewHalfClosed() State {
	s := new(HalfClosed)
	fmt.Println(s)
	return s
}

func (s *HalfClosed) H(frame *HeadersFrame) (State, error) {
	return nil, fmt.Errorf("illegal state %s at %s", frame, s)
}

func (s *HalfClosed) R(frame *RstStreamFrame) (State, error) {
	return NewClosed(), nil
}

func (s *HalfClosed) ES(frame Frame) (State, error) {
	return NewClosed(), nil
}

func (s HalfClosed) String() string {
	return "HalfClosed"
}
