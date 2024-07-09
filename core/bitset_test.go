package core

import (
	"testing"
)

func TestBitstring(t *testing.T) {
	b := NewBitset(10)
	if b.Contains(0) {
		t.Errorf("Expected bit 0 to be unset")
	}
	b.Insert(0)
	if !b.Contains(0) {
		t.Errorf("Expected bit 0 to be set")
	}
	if b.Contains(1) {
		t.Errorf("Expected bit 1 to be unset")
	}
	b.Insert(1)
	if !b.Contains(1) {
		t.Errorf("Expected bit 1 to be set")
	}
}
