package crawl

import (
	"io"
	"strings"
)

// rot13Reader adds byte-level rot13 transformation to an io.Reader
type rot13Reader struct {
	r io.Reader
}

func rot13(r byte) byte {
	switch {
	case r >= 'A' && r <= 'Z':
		return 'A' + (r-'A'+13)%26
	case r >= 'a' && r <= 'z':
		return 'a' + (r-'a'+13)%26
	default:
		return r
	}
}

// Read() reads bytes from underliying r13.r
// and applies rot13 transformation
func (r13 rot13Reader) Read(buf []byte) (int, error) {
	// fill buffer
	n, err := r13.r.Read(buf)
	if err == nil {
		// apply rot13 on bytes (not runes - no UTF8 decoding required !)
		for i := range n {
			buf[i] = rot13(buf[i])
		}
	}
	return n, err
}

// Rot13 is a helper function to avoid the hassle of
// creating a strings.NewReader() in simple cases
func Rot13(in string) (out string, err error) {
	r := rot13Reader{strings.NewReader(in)}

	if out, err := io.ReadAll(r); err == nil {
		return string(out), nil
	}
	return in, err
}
