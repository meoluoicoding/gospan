package input

import "testing"

func TestBufferBounded(t *testing.T) {
	b := NewBuffer(3)
	for _, r := range "abcdef" {
		b.Push(r)
	}
	if got := b.String(); got != "def" {
		t.Errorf("String() = %q, want def", got)
	}
	if b.Len() != 3 {
		t.Errorf("Len() = %d, want 3", b.Len())
	}
}

func TestBufferResetAndBackspace(t *testing.T) {
	b := NewBuffer(8)
	b.Push('a')
	b.Push('b')
	b.Backspace()
	if b.String() != "a" {
		t.Errorf("after Backspace = %q, want a", b.String())
	}
	b.Reset()
	if b.Len() != 0 {
		t.Errorf("after Reset Len = %d, want 0", b.Len())
	}
}

func TestBufferRemoveAt(t *testing.T) {
	b := NewBuffer(8)
	for _, r := range "abcd" {
		b.Push(r)
	}
	b.RemoveAt(1)
	if b.String() != "acd" {
		t.Errorf("RemoveAt(1) = %q, want acd", b.String())
	}
	b.RemoveAt(99) // out of range: ignored
	if b.String() != "acd" {
		t.Errorf("out-of-range RemoveAt changed buffer: %q", b.String())
	}
}

func TestBufferUnicode(t *testing.T) {
	b := NewBuffer(4)
	for _, r := range "được" {
		b.Push(r)
	}
	if b.String() != "được" {
		t.Errorf("String() = %q, want được", b.String())
	}
	if b.Len() != 4 {
		t.Errorf("Len() = %d, want 4", b.Len())
	}
}
