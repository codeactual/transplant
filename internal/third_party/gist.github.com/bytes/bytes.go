package bytes

import (
	std_bytes "bytes"
	"sync"
)

// SharedBuffer is a goroutine safe bytes.Buffer
//
// Origin:
//   https://gist.github.com/arkan/5924e155dbb4254b64614069ba0afd81
//   https://github.com/arkan
//
// Changes:
//   - buffer field is now a pointer
//   - Renamed to SharedBuffer
//   - Added NewSharedBuffer
type SharedBuffer struct {
	buffer *std_bytes.Buffer
	mutex  sync.Mutex
}

// Write appends the contents of p to the buffer, growing the buffer as needed. It returns
// the number of bytes written.
func (s *SharedBuffer) Write(p []byte) (n int, err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.buffer.Write(p)
}

// String returns the contents of the unread portion of the buffer
// as a string.  If the SharedBuffer is a nil pointer, it returns "<nil>".
func (s *SharedBuffer) String() string {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.buffer.String()
}

func (s *SharedBuffer) Unshared() *std_bytes.Buffer {
	return s.buffer
}

func NewSharedBuffer() *SharedBuffer {
	return &SharedBuffer{buffer: new(std_bytes.Buffer)}
}
