package state

import (
	"testing"
)

func TestStory(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatal(r)
		}
	}()

	var idle State

	idle = NewIdle()
	idle.H().R()

	idle = NewIdle()
	idle.H().ES().ES()

	idle = NewIdle()
	idle.H().ES().R()

	idle = NewIdle()
	idle.PP().H().ES()

	idle = NewIdle()
	idle.PP().H().R()

	idle = NewIdle()
	idle.PP().R()
}

func TestSuccess(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatal(r)
		}
	}()

	NewIdle().H()
	NewIdle().PP()

	NewOpen().ES()
	NewOpen().R()

	NewReserved().H()
	NewReserved().R()

	NewHalfClosed().ES()
	NewHalfClosed().R()
}

func TestFail(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Log(r)
		}
	}()

	NewIdle().ES()
	NewIdle().R()

	NewOpen().H()
	NewOpen().PP()

	NewReserved().PP()
	NewReserved().ES()

	NewHalfClosed().H()
	NewHalfClosed().PP()

	NewClosed().H()
	NewClosed().PP()
	NewClosed().ES()
	NewClosed().R()
}
