package jsonl

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
)

func TestDecoderUsesLFOnlyAndPreservesUnicodeSeparators(t *testing.T) {
	input := "{\"text\":\"a\u2028b\u2029c\"}\n{\"ok\":true}\n"
	d := NewDecoder(strings.NewReader(input))
	first, err := d.Next()
	var decoded struct {
		Text string `json:"text"`
	}
	if err != nil || json.Unmarshal(first, &decoded) != nil || decoded.Text != "a\u2028b\u2029c" {
		t.Fatalf("first=%q err=%v", first, err)
	}
	second, err := d.Next()
	if err != nil || string(second) != `{"ok":true}` {
		t.Fatalf("second=%q err=%v", second, err)
	}
}

func TestDecoderCRLFPolicyAndPartialEOF(t *testing.T) {
	d := NewDecoder(strings.NewReader("{\"ok\":true}\r\n"))
	if _, err := d.Next(); !errors.Is(err, ErrMalformedFrame) {
		t.Fatalf("CRLF err=%v want %v", err, ErrMalformedFrame)
	}
	d = NewDecoder(strings.NewReader(`{"ok":true}`))
	if _, err := d.Next(); !errors.Is(err, ErrPartialFrame) {
		t.Fatalf("partial err=%v want %v", err, ErrPartialFrame)
	}
}

func TestDecoderCleanEOF(t *testing.T) {
	if _, err := NewDecoder(strings.NewReader("")).Next(); !errors.Is(err, io.EOF) {
		t.Fatalf("empty input err=%v want EOF", err)
	}
	d := NewDecoder(strings.NewReader("{}\n"))
	if _, err := d.Next(); err != nil {
		t.Fatal(err)
	}
	if _, err := d.Next(); !errors.Is(err, io.EOF) {
		t.Fatalf("after complete frame err=%v want EOF", err)
	}
}

func TestNegotiateProfileAcceptsOnlyCanonicalVersion(t *testing.T) {
	if err := NegotiateProfile(types.HostProtocolVersionV1); err != nil {
		t.Fatal(err)
	}
	for _, version := range []string{"", "embedded_host_protocol.v2"} {
		if err := NegotiateProfile(version); err == nil {
			t.Fatalf("version %q was accepted", version)
		}
	}
}

func TestDecoderClassifiesOverLimitFrameWithoutLF(t *testing.T) {
	d, err := NewDecoderWithError(strings.NewReader("12345"), WithMaxFrameBytes(4))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.Next(); !errors.Is(err, ErrFrameTooLarge) {
		t.Fatalf("err=%v want %v", err, ErrFrameTooLarge)
	}
}

func TestDecoderMaximumBoundaryAndValidatedOverride(t *testing.T) {
	const frame = `{"a":1}`
	d, err := NewDecoderWithError(strings.NewReader(frame+"\n"), WithMaxFrameBytes(len(frame)))
	if err != nil {
		t.Fatal(err)
	}
	if got, err := d.Next(); err != nil || string(got) != frame {
		t.Fatalf("boundary frame=%q err=%v", got, err)
	}
	if _, err := NewDecoderWithError(strings.NewReader(""), WithMaxFrameBytes(0)); err == nil {
		t.Fatal("invalid override accepted")
	}
	for _, max := range []int{-1, 16 * DefaultMaxFrameBytes, 16*DefaultMaxFrameBytes + 1} {
		d, err := NewDecoderWithError(strings.NewReader("{}\n"), WithMaxFrameBytes(max))
		if max == 16*DefaultMaxFrameBytes {
			if err != nil || d == nil {
				t.Fatalf("hard ceiling max=%d decoder=%v err=%v", max, d, err)
			}
			continue
		}
		if err == nil {
			t.Fatalf("invalid max=%d accepted", max)
		}
	}
}

func TestDecoderRejectsEmptyMalformedAndOversizedFrames(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input string
		max   int
		want  error
	}{
		{"empty", "\n", 10, ErrEmptyFrame},
		{"malformed", "not-json\n", 10, ErrMalformedFrame},
		{"not-object", "true\n", 10, ErrMalformedFrame},
		{"array", "[]\n", 10, ErrMalformedFrame},
		{"multiple-values", "{}{}\n", 10, ErrMalformedFrame},
		{"oversized", "12345\n", 4, ErrFrameTooLarge},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d, constructErr := NewDecoderWithError(strings.NewReader(tc.input), WithMaxFrameBytes(tc.max))
			if constructErr != nil {
				t.Fatal(constructErr)
			}
			_, err := d.Next()
			if !errors.Is(err, tc.want) {
				t.Fatalf("err=%v want %v", err, tc.want)
			}
		})
	}
}

func TestDecoderDiscardsRemainderOfOversizedFrameBeforeReadingNextFrame(t *testing.T) {
	// Fill the reader's internal buffer before placing a short, valid-looking
	// JSON suffix in the same oversized frame. That suffix must never be
	// interpreted as a standalone command on the next call.
	input := strings.Repeat("x", 4096) + `{"x":1}` + "\n" + `{"y":2}` + "\n"
	d, err := NewDecoderWithError(strings.NewReader(input), WithMaxFrameBytes(8))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.Next(); !errors.Is(err, ErrFrameTooLarge) {
		t.Fatalf("oversized err=%v want %v", err, ErrFrameTooLarge)
	}
	frame, err := d.Next()
	if err != nil {
		t.Fatalf("next frame: %v", err)
	}
	if got := string(frame); got != `{"y":2}` {
		t.Fatalf("next frame=%q, want only the frame after the oversized delimiter", got)
	}
}

type flushErrorWriter struct{ bytes.Buffer }

func (w *flushErrorWriter) Flush() error { return errors.New("flush failed") }

func TestWriterFlushErrorAndConcurrentAtomicity(t *testing.T) {
	bad := NewWriter(&flushErrorWriter{})
	if err := bad.WriteFrame([]byte(`{"a":1}`)); err == nil || !strings.Contains(err.Error(), "flush failed") {
		t.Fatalf("flush err=%v", err)
	}

	var buf bytes.Buffer
	w := NewWriter(&buf)
	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := w.WriteFrameContext(context.Background(), []byte(`{"frame":true}`)); err != nil {
				t.Errorf("write: %v", err)
			}
		}()
	}
	wg.Wait()
	lines := strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n")
	if len(lines) != 32 {
		t.Fatalf("lines=%d", len(lines))
	}
	for _, line := range lines {
		if line != `{"frame":true}` {
			t.Fatalf("interleaved line=%q", line)
		}
	}
}

func TestWriterSerializesHeterogeneousProtocolFrames(t *testing.T) {
	frames := []any{
		types.HostCommandResponse{Version: types.HostProtocolVersionV1, MessageID: "response-1", Kind: types.HostEnvelopeKindCommandResponse, Time: time.Now().UTC(), RequestID: "request-1", Status: types.HostAdmissionStatusAccepted},
		types.HostRuntimeEventEnvelope{Version: types.HostProtocolVersionV1, MessageID: "event-1", Kind: types.HostEnvelopeKindRuntimeEvent, Time: time.Now().UTC(), RequestID: "request-1", Event: types.EventEnvelope{EventID: "event-1", RunID: "run-1", Source: types.ProtocolSourceRunner, Kind: types.ProtocolEventKindProgress, Time: time.Now().UTC()}},
		types.HostRequestEnvelope{Version: types.HostProtocolVersionV1, MessageID: "host-request-1", Kind: types.HostEnvelopeKindHostRequest, Time: time.Now().UTC(), RequestID: "clarification-1", RequestType: "clarification"},
		types.HostResponseEnvelope{Version: types.HostProtocolVersionV1, MessageID: "host-response-1", Kind: types.HostEnvelopeKindHostResponse, Time: time.Now().UTC(), RequestID: "clarification-1", Accepted: true},
	}
	var output bytes.Buffer
	writer := NewWriter(&output)
	var group sync.WaitGroup
	for _, frame := range frames {
		encoded, err := json.Marshal(frame)
		if err != nil {
			t.Fatal(err)
		}
		group.Add(1)
		go func() {
			defer group.Done()
			if err := writer.WriteFrame(encoded); err != nil {
				t.Errorf("write: %v", err)
			}
		}()
	}
	group.Wait()
	lines := bytes.Split(bytes.TrimSuffix(output.Bytes(), []byte{'\n'}), []byte{'\n'})
	if len(lines) != len(frames) {
		t.Fatalf("frames=%d want %d", len(lines), len(frames))
	}
	seen := make(map[types.HostEnvelopeKind]int, len(frames))
	for _, line := range lines {
		var envelope struct {
			Kind types.HostEnvelopeKind `json:"kind"`
		}
		if err := json.Unmarshal(line, &envelope); err != nil {
			t.Fatalf("interleaved frame %q: %v", line, err)
		}
		seen[envelope.Kind]++
	}
	for _, kind := range []types.HostEnvelopeKind{types.HostEnvelopeKindCommandResponse, types.HostEnvelopeKindRuntimeEvent, types.HostEnvelopeKindHostRequest, types.HostEnvelopeKindHostResponse} {
		if seen[kind] != 1 {
			t.Fatalf("kind %q count=%d want 1", kind, seen[kind])
		}
	}
}

type deadlineWriter struct {
	deadlines []time.Time
	bytes.Buffer
}

func (w *deadlineWriter) SetWriteDeadline(deadline time.Time) error {
	w.deadlines = append(w.deadlines, deadline)
	return nil
}

func TestWriterAppliesContextDeadline(t *testing.T) {
	raw := &deadlineWriter{}
	w := NewWriter(raw)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := w.WriteFrameContext(ctx, []byte(`{"ok":true}`)); err != nil {
		t.Fatal(err)
	}
	if len(raw.deadlines) != 2 || raw.deadlines[0].IsZero() || !raw.deadlines[1].IsZero() {
		t.Fatalf("deadlines=%v, want applied deadline followed by reset", raw.deadlines)
	}
	if err := w.WriteFrameContext(context.Background(), []byte(`{"next":true}`)); err != nil {
		t.Fatal(err)
	}
	if len(raw.deadlines) != 3 || !raw.deadlines[2].IsZero() {
		t.Fatalf("deadlines=%v, background write must clear stale deadline", raw.deadlines)
	}
	if raw.deadlines[0].IsZero() {
		t.Fatal("write deadline was not applied")
	}
}

func TestWriterRejectsCarriageReturnFraming(t *testing.T) {
	var buf bytes.Buffer
	if err := NewWriter(&buf).WriteFrame([]byte("{\"ok\":true}\r")); !errors.Is(err, ErrMalformedFrame) {
		t.Fatalf("err=%v want %v", err, ErrMalformedFrame)
	}
}

type shortWriter struct {
	buf bytes.Buffer
	max int
}

func (w *shortWriter) Write(p []byte) (int, error) {
	if len(p) > w.max {
		p = p[:w.max]
	}
	return w.buf.Write(p)
}

func TestWriterHandlesPartialWrites(t *testing.T) {
	raw := &shortWriter{max: 2}
	w := NewWriter(raw)
	if err := w.WriteFrame([]byte(`{"ok":true}`)); err != nil {
		t.Fatal(err)
	}
	if got := raw.buf.String(); got != "{\"ok\":true}\n" {
		t.Fatalf("partial output=%q", got)
	}
}

func TestWriterSerializesCompleteFrames(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf)
	if err := w.WriteFrame([]byte(`{"a":1}`)); err != nil {
		t.Fatal(err)
	}
	if got := buf.String(); got != "{\"a\":1}\n" {
		t.Fatalf("got %q", got)
	}
	if err := w.WriteFrame([]byte("bad\nframe")); !errors.Is(err, ErrMalformedFrame) {
		t.Fatalf("err=%v", err)
	}
	if _, err := io.WriteString(&buf, ""); err != nil {
		t.Fatal(err)
	}
}
