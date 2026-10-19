// Package input defines the platform-independent input buffer used by the
// matching engine. It stores recently typed runes so the engine can look for
// triggers at the end of the stream without retaining unbounded history.
package input

// Buffer is a bounded ring of runes representing the most recently typed input.
// It is not safe for concurrent use; the engine serialises access.
type Buffer struct {
	runes []rune
	size  int
}

// NewBuffer returns a Buffer that retains at most size runes. A non-positive
// size selects a small default.
func NewBuffer(size int) *Buffer {
	if size <= 0 {
		size = 256
	}
	return &Buffer{
		runes: make([]rune, 0, size),
		size:  size,
	}
}

// Push appends r, dropping the oldest rune when the buffer is full.
func (b *Buffer) Push(r rune) {
	if len(b.runes) == b.size {
		copy(b.runes, b.runes[1:])
		b.runes[len(b.runes)-1] = r
		return
	}
	b.runes = append(b.runes, r)
}

// Runes returns the underlying runes. Callers must not modify the slice.
func (b *Buffer) Runes() []rune {
	return b.runes
}

// String returns the buffered input as a string.
func (b *Buffer) String() string {
	return string(b.runes)
}

// Len returns the number of buffered runes.
func (b *Buffer) Len() int {
	return len(b.runes)
}

// Cap returns the maximum number of runes the buffer retains.
func (b *Buffer) Cap() int {
	return b.size
}

// Reset discards all buffered runes. It retains the allocated capacity.
func (b *Buffer) Reset() {
	b.runes = b.runes[:0]
}

// Backspace removes the last rune, if any.
func (b *Buffer) Backspace() {
	if len(b.runes) > 0 {
		b.runes = b.runes[:len(b.runes)-1]
	}
}

// RemoveAt deletes the rune at index i, shifting later runes left. Indices
// outside the buffer are ignored.
func (b *Buffer) RemoveAt(i int) {
	if i < 0 || i >= len(b.runes) {
		return
	}
	copy(b.runes[i:], b.runes[i+1:])
	b.runes = b.runes[:len(b.runes)-1]
}
