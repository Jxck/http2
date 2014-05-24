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
	H(*HeadersFrame) State
	ES(Frame) State
	R(*RstStreamFrame) State
	String() string
}

type BaseState struct {
}

func (s *BaseState) H(headers *HeadersFrame) State {
	panic("PROTOCOL_ERROR: invalid state: H")
}

func (s *BaseState) ES(Frame) State {
	panic("PROTOCOL_ERROR: invalid state: ES")
}

func (s *BaseState) R(*RstStreamFrame) State {
	panic("PROTOCOL_ERROR: invalid state: R")
}

/**
 * Idle
 */
type Idle struct {
	BaseState
}

func NewIdle() State {
	s := new(Idle)
	fmt.Println(s)
	return s
}

func (s Idle) String() string {
	return "Idle"
}

func (s *Idle) H(headers *HeadersFrame) State {
	return NewOpen()
}

/**
 * Open
 */
type Open struct {
	BaseState
}

func NewOpen() State {
	s := new(Open)
	fmt.Println(s)
	return s
}

func (s Open) String() string {
	return "Open"
}

func (s *Open) R(*RstStreamFrame) State {
	return NewClosed()
}

func (s *Open) ES(Frame) State {
	return NewHalfClosed()
}

/**
 * Closed
 */
type Closed struct {
	BaseState
}

func NewClosed() State {
	s := new(Closed)
	fmt.Println(s)
	return s
}

func (s Closed) String() string {
	return "Closed"
}

/**
 * Reserved
 */
type Reserved struct {
	BaseState
}

func NewReserved() State {
	s := new(Reserved)
	fmt.Println(s)
	return s
}

func (s *Reserved) H(headers *HeadersFrame) State {
	return NewHalfClosed()
}

func (s *Reserved) R(*RstStreamFrame) State {
	return NewClosed()
}

func (s Reserved) String() string {
	return "Reserved"
}

/**
 * HalfClosed
 */
type HalfClosed struct {
	BaseState
}

func NewHalfClosed() State {
	s := new(HalfClosed)
	fmt.Println(s)
	return s
}

func (s *HalfClosed) R(*RstStreamFrame) State {
	return NewClosed()
}

func (s *HalfClosed) ES(Frame) State {
	return NewClosed()
}

func (s HalfClosed) String() string {
	return "HalfClosed"
}
