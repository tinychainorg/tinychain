package core

import (
	"testing"
)

func TestBitstring(t *testing.T) {
	b := NewBitstring(10)
	if b.IsSet(0) {
		t.Errorf("Expected bit 0 to be unset")
	}
	b.SetBit(0)
	if !b.IsSet(0) {
		t.Errorf("Expected bit 0 to be set")
	}
	if b.IsSet(1) {
		t.Errorf("Expected bit 1 to be unset")
	}
	b.SetBit(1)
	if !b.IsSet(1) {
		t.Errorf("Expected bit 1 to be set")
	}
}