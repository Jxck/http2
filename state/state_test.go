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

	idle = NewIDLE()
	idle.H().R()

	idle = NewIDLE()
	idle.H().ES().ES()

	idle = NewIDLE()
	idle.H().ES().R()

	idle = NewIDLE()
	idle.PP().H().ES()

	idle = NewIDLE()
	idle.PP().H().R()

	idle = NewIDLE()
	idle.PP().R()
}

func TestSuccess(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatal(r)
		}
	}()

	NewIDLE().H()
	NewIDLE().PP()

	NewOPEN().ES()
	NewOPEN().R()

	NewRESERVED().H()
	NewRESERVED().R()

	NewHALFCLOSED().ES()
	NewHALFCLOSED().R()
}

func TestFail(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Log(r)
		}
	}()

	NewIDLE().ES()
	NewIDLE().R()

	NewOPEN().H()
	NewOPEN().PP()

	NewRESERVED().PP()
	NewRESERVED().ES()

	NewHALFCLOSED().H()
	NewHALFCLOSED().PP()

	NewCLOSED().H()
	NewCLOSED().PP()
	NewCLOSED().ES()
	NewCLOSED().R()
}
