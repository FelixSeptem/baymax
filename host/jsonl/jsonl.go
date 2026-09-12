// Package jsonl implements the strict local JSONL binding for host envelopes.
package jsonl

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
)

const DefaultMaxFrameBytes = 1 << 20

var (
	ErrEmptyFrame         = errors.New("jsonl: empty frame")
	ErrMalformedFrame     = errors.New("jsonl: malformed frame")
	ErrFrameTooLarge      = errors.New("jsonl: frame too large")
	ErrPartialFrame       = errors.New("jsonl: partial frame at EOF")
	ErrUnsupportedVersion = errors.New("jsonl: unsupported protocol version")
)

type ProfileNegotiation struct {
	Version string `json:"version"`
}

func NegotiateProfile(version string) error {
	if version != types.HostProtocolVersionV1 {
		return fmt.Errorf("%w %q", ErrUnsupportedVersion, version)
	}
	return nil
}

const HostProtocolVersionV1 = types.HostProtocolVersionV1

type Decoder struct {
	r   *bufio.Reader
	max int
}

type DecoderOption func(*Decoder) error

func WithMaxFrameBytes(max int) DecoderOption {
	return func(d *Decoder) error {
		if max <= 0 || max > 16*DefaultMaxFrameBytes {
			return fmt.Errorf("jsonl: max frame bytes must be between 1 and %d", 16*DefaultMaxFrameBytes)
		}
		d.max = max
		return nil
	}
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{r: bufio.NewReader(r), max: DefaultMaxFrameBytes}
}

// NewDecoderWithError is the validating constructor for callers that need to
// reject invalid frame bounds before serving a connection.
func NewDecoderWithError(r io.Reader, opts ...DecoderOption) (*Decoder, error) {
	d := &Decoder{r: bufio.NewReader(r), max: DefaultMaxFrameBytes}
	for _, opt := range opts {
		if opt != nil {
			if err := opt(d); err != nil {
				return nil, err
			}
		}
	}
	return d, nil
}

func (d *Decoder) Next() ([]byte, error) {
	if d == nil || d.r == nil {
		return nil, io.EOF
	}
	line := make([]byte, 0, min(d.max, 4096))
	for {
		fragment, err := d.r.ReadSlice('\n')
		limit := d.max
		if err == nil {
			limit++ // A complete frame includes its LF delimiter.
		}
		if len(line)+len(fragment) > limit {
			if errors.Is(err, bufio.ErrBufferFull) {
				d.discardFrameRemainder()
			}
			return nil, ErrFrameTooLarge
		}
		line = append(line, fragment...)
		switch {
		case err == nil:
			line = line[:len(line)-1]
			goto decoded
		case errors.Is(err, bufio.ErrBufferFull):
			continue
		case errors.Is(err, io.EOF) && len(line) > 0:
			return nil, ErrPartialFrame
		case err != nil:
			return nil, err
		}
	}

decoded:
	if len(line) == 0 {
		return nil, ErrEmptyFrame
	}
	if line[len(line)-1] == '\r' {
		return nil, ErrMalformedFrame
	}
	trimmed := bytes.TrimSpace(line)
	if len(trimmed) == 0 {
		return nil, ErrEmptyFrame
	}
	if trimmed[0] != '{' || trimmed[len(trimmed)-1] != '}' || !json.Valid(trimmed) {
		return nil, ErrMalformedFrame
	}
	return append([]byte(nil), trimmed...), nil
}

func (d *Decoder) discardFrameRemainder() {
	for {
		_, err := d.r.ReadSlice('\n')
		if err == nil || !errors.Is(err, bufio.ErrBufferFull) {
			return
		}
	}
}

type Writer struct {
	mu sync.Mutex
	w  io.Writer
}

func NewWriter(w io.Writer) *Writer { return &Writer{w: w} }

func (w *Writer) WriteFrame(frame []byte) error {
	return w.WriteFrameContext(context.Background(), frame)
}

// WriteFrameContext writes one complete frame under the caller's deadline.
// Deadline support is optional and delegated to transports that expose it.
func (w *Writer) WriteFrameContext(ctx context.Context, frame []byte) (resultErr error) {
	if w == nil || w.w == nil {
		return io.ErrClosedPipe
	}
	trimmed := bytes.TrimSpace(frame)
	if len(trimmed) == 0 || bytes.IndexByte(frame, '\n') >= 0 || bytes.IndexByte(frame, '\r') >= 0 || trimmed[0] != '{' || trimmed[len(trimmed)-1] != '}' || !json.Valid(trimmed) {
		return ErrMalformedFrame
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if setter, ok := w.w.(interface{ SetWriteDeadline(time.Time) error }); ok {
		deadline, hasDeadline := ctx.Deadline()
		if err := setter.SetWriteDeadline(deadline); err != nil {
			return err
		}
		if hasDeadline {
			defer func() {
				if err := setter.SetWriteDeadline(time.Time{}); resultErr == nil && err != nil {
					resultErr = err
				}
			}()
		}
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	data := append(append([]byte(nil), frame...), '\n')
	for len(data) > 0 {
		n, err := w.w.Write(data)
		if err != nil {
			return err
		}
		if n <= 0 {
			return io.ErrShortWrite
		}
		data = data[n:]
	}
	if flusher, ok := w.w.(interface{ Flush() error }); ok {
		if err := flusher.Flush(); err != nil {
			return err
		}
	}
	return nil
}
