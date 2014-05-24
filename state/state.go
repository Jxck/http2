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
 * IDLE
 */
type IDLE struct {
	BaseState
}

func NewIDLE() State {
	s := new(IDLE)
	fmt.Println(s)
	return s
}

func (s IDLE) String() string {
	return "IDLE"
}

func (s *IDLE) H() State {
	return NewOPEN()
}

func (s *IDLE) PP() State {
	return NewRESERVED()
}

/**
 * OPEN
 */
type OPEN struct {
	BaseState
}

func NewOPEN() State {
	s := new(OPEN)
	fmt.Println(s)
	return s
}

func (s OPEN) String() string {
	return "OPEN"
}

func (s *OPEN) R() State {
	return NewCLOSED()
}

func (s *OPEN) ES() State {
	return NewHALFCLOSED()
}

/**
 * CLOSED
 */
type CLOSED struct {
	BaseState
}

func NewCLOSED() State {
	s := new(CLOSED)
	fmt.Println(s)
	return s
}

func (s CLOSED) String() string {
	return "CLOSED"
}

/**
 * RESERVED
 */
type RESERVED struct {
	BaseState
}

func NewRESERVED() State {
	s := new(RESERVED)
	fmt.Println(s)
	return s
}

func (s *RESERVED) H() State {
	return NewHALFCLOSED()
}

func (s *RESERVED) R() State {
	return NewCLOSED()
}

func (s RESERVED) String() string {
	return "RESERVED"
}

/**
 * HALFCLOSED
 */
type HALFCLOSED struct {
	BaseState
}

func NewHALFCLOSED() State {
	s := new(HALFCLOSED)
	fmt.Println(s)
	return s
}

func (s *HALFCLOSED) R() State {
	return NewCLOSED()
}

func (s *HALFCLOSED) ES() State {
	return NewCLOSED()
}

func (s HALFCLOSED) String() string {
	return "HALFCLOSED"
}
