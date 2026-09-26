// Package boundedio bounds bytes consumed from untrusted upstream streams.
package boundedio

import (
	"errors"
	"io"
)

// JSONResponseLimit bounds decoded Bungie API and OAuth response bodies.
const JSONResponseLimit int64 = 32 << 20

var ErrTooLarge = errors.New("upstream body exceeds byte limit")

// ReadAll returns the entire stream or an error, never a truncated success.
// The caller retains ownership of r. The one-byte overflow probe is not returned.
func ReadAll(r io.Reader, limit int64) ([]byte, error) {
	if limit <= 0 {
		return nil, errors.New("upstream byte limit must be positive")
	}
	body, err := io.ReadAll(io.LimitReader(r, limit))
	if err == nil && int64(len(body)) == limit {
		err = checkEOF(r)
	}
	if err != nil {
		return nil, err
	}
	return body, nil
}

// Copy writes at most limit bytes and checks for excess without writing the
// overflow byte. Callers must discard partial output on failure and close both
// streams themselves; a successful copy does not imply a successful file close.
func Copy(dst io.Writer, src io.Reader, limit int64) error {
	if limit <= 0 {
		return errors.New("upstream byte limit must be positive")
	}
	n, err := io.Copy(dst, io.LimitReader(src, limit))
	if err == nil && n == limit {
		err = checkEOF(src)
	}
	return err
}

func checkEOF(r io.Reader) error {
	var probe [1]byte
	n, err := io.ReadFull(r, probe[:])
	if n != 0 {
		return ErrTooLarge
	}
	if errors.Is(err, io.EOF) {
		return nil
	}
	return err
}
