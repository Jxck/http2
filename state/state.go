package state

import (
	"fmt"
	"log"
	//"github.com/jxck/http2/frame"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

type State interface {
	H() State
	PP() State
	ES() State
	R() State
	String() string
}

type BaseState struct {
}

func (s *BaseState) H() State {
	panic("PROTOCOL_ERROR: invalid state: H")
}

func (s *BaseState) PP() State {
	panic("PROTOCOL_ERROR: invalid state: PP")
}

func (s *BaseState) ES() State {
	panic("PROTOCOL_ERROR: invalid state: ES")
}

func (s *BaseState) R() State {
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

func (s *Idle) H() State {
	return NewOpen()
}

func (s *Idle) PP() State {
	return NewReserved()
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

func (s *Open) R() State {
	return NewClosed()
}

func (s *Open) ES() State {
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

func (s *Reserved) H() State {
	return NewHalfClosed()
}

func (s *Reserved) R() State {
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

func (s *HalfClosed) R() State {
	return NewClosed()
}

func (s *HalfClosed) ES() State {
	return NewClosed()
}

func (s HalfClosed) String() string {
	return "HalfClosed"
}
