package jsonl

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/host"
)

func TestServerRequiresNegotiationBeforeMutation(t *testing.T) {
	control := &countingControl{}
	command := validActionCommand()
	input := bytes.NewBuffer(append(mustJSON(t, command), '\n'))
	var protocol bytes.Buffer
	var diagnostics bytes.Buffer
	server, err := NewServer(host.New(nil, host.WithSourceControl(control)), ServerConfig{
		Input:          io.NopCloser(input),
		ProtocolOutput: nopWriteCloser{Writer: &protocol},
		LogOutput:      &diagnostics,
	})
	if err != nil {
		t.Fatal(err)
	}

	err = server.Serve(context.Background())
	if !errors.Is(err, ErrNegotiationRequired) {
		t.Fatalf("Serve() error = %v, want %v", err, ErrNegotiationRequired)
	}
	if got := control.calls.Load(); got != 0 {
		t.Fatalf("source mutations = %d, want 0", got)
	}
	if protocol.Len() != 0 {
		t.Fatalf("protocol output = %q, want empty", protocol.String())
	}
	if !strings.Contains(diagnostics.String(), string(ExitNegotiation)) {
		t.Fatalf("diagnostics = %q, want protocol classification", diagnostics.String())
	}
}

func TestDefaultServerConfigReservesStdoutForProtocolAndStderrForLogs(t *testing.T) {
	config := DefaultServerConfig()
	if config.Input != os.Stdin {
		t.Fatalf("default input = %T, want os.Stdin", config.Input)
	}
	if config.ProtocolOutput != os.Stdout {
		t.Fatalf("default protocol output = %T, want os.Stdout", config.ProtocolOutput)
	}
	if config.LogOutput != os.Stderr {
		t.Fatalf("default log output = %T, want os.Stderr", config.LogOutput)
	}
	if config.DeliveryTimeout <= 0 {
		t.Fatalf("default delivery timeout = %v, want bounded", config.DeliveryTimeout)
	}
}

func TestFrameWriterRejectsNonObjectOutputWithoutPollutingProtocol(t *testing.T) {
	var protocol bytes.Buffer
	writer := NewFrameWriter(nopWriteCloser{Writer: &protocol})
	if err := writer.WriteFrame(context.Background(), "accidental log"); !errors.Is(err, ErrMalformedFrame) {
		t.Fatalf("WriteFrame(log) error = %v, want %v", err, ErrMalformedFrame)
	}
	if protocol.Len() != 0 {
		t.Fatalf("protocol output = %q, want empty", protocol.String())
	}
}

func TestFrameWriterPreservesAndClassifiesArbitraryOutputFailure(t *testing.T) {
	writeErr := errors.New("disk disappeared")
	writer := NewFrameWriter(nopWriteCloser{Writer: errorWriter{err: writeErr}})
	err := writer.WriteFrame(context.Background(), ProfileNegotiation{Version: HostProtocolVersionV1})
	if !errors.Is(err, writeErr) {
		t.Fatalf("WriteFrame() error = %v, want wrapped %v", err, writeErr)
	}
	if got := ClassifyExit(err); got != ExitOutputFailure {
		t.Fatalf("ClassifyExit() = %q, want %q", got, ExitOutputFailure)
	}
}

func TestServerNegotiatesThenDispatchesThroughCoordinator(t *testing.T) {
	control := &countingControl{}
	var input bytes.Buffer
	input.Write(append(mustJSON(t, ProfileNegotiation{Version: HostProtocolVersionV1}), '\n'))
	input.Write(append(mustJSON(t, validActionCommand()), '\n'))
	var protocol bytes.Buffer
	server, err := NewServer(host.New(nil, host.WithSourceControl(control)), ServerConfig{
		Input:          io.NopCloser(&input),
		ProtocolOutput: nopWriteCloser{Writer: &protocol},
		LogOutput:      io.Discard,
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := server.Serve(context.Background()); err != nil {
		t.Fatalf("Serve() error = %v", err)
	}
	if got := control.calls.Load(); got != 1 {
		t.Fatalf("source mutations = %d, want 1", got)
	}
	frames := decodeAllFrames(t, protocol.Bytes())
	if len(frames) != 2 {
		t.Fatalf("protocol frames = %d, want negotiation and command response; output=%q", len(frames), protocol.String())
	}
	if got := frames[0]["version"]; got != HostProtocolVersionV1 {
		t.Fatalf("negotiated version = %v", got)
	}
	if got := frames[1]["kind"]; got != string(types.HostEnvelopeKindCommandResponse) {
		t.Fatalf("response kind = %v", got)
	}
}

func TestServerClassifiesMalformedAndUnsupportedNegotiation(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  error
		class ExitClassification
	}{
		{name: "malformed", input: "not-json\n", want: ErrMalformedFrame, class: ExitMalformedInput},
		{name: "unsupported", input: `{"version":"future"}` + "\n", want: ErrUnsupportedVersion, class: ExitUnsupportedVersion},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var diagnostics bytes.Buffer
			server, err := NewServer(host.New(nil), ServerConfig{
				Input:          io.NopCloser(strings.NewReader(tt.input)),
				ProtocolOutput: nopWriteCloser{Writer: io.Discard},
				LogOutput:      &diagnostics,
			})
			if err != nil {
				t.Fatal(err)
			}
			err = server.Serve(context.Background())
			if !errors.Is(err, tt.want) {
				t.Fatalf("Serve() error = %v, want %v", err, tt.want)
			}
			if got := ClassifyExit(err); got != tt.class {
				t.Fatalf("ClassifyExit() = %q, want %q", got, tt.class)
			}
			if !strings.Contains(diagnostics.String(), string(tt.class)) {
				t.Fatalf("diagnostics = %q, want classification %q", diagnostics.String(), tt.class)
			}
		})
	}
}

func TestServerCleanEOFClosesTransportWithoutDiagnostic(t *testing.T) {
	input := &trackingReadCloser{Reader: strings.NewReader("")}
	output := &trackingWriteCloser{Writer: io.Discard}
	var diagnostics bytes.Buffer
	server, err := NewServer(host.New(nil), ServerConfig{Input: input, ProtocolOutput: output, LogOutput: &diagnostics})
	if err != nil {
		t.Fatal(err)
	}

	if err := server.Serve(context.Background()); err != nil {
		t.Fatalf("Serve() error = %v, want clean EOF", err)
	}
	if !input.Closed() || !output.Closed() {
		t.Fatalf("cleanup input=%v output=%v, want both closed", input.Closed(), output.Closed())
	}
	if diagnostics.Len() != 0 {
		t.Fatalf("diagnostics = %q, want empty", diagnostics.String())
	}
}

func TestServerEOFSettlesPendingAndLeavesNoConnectionState(t *testing.T) {
	server, err := NewServer(host.New(nil), ServerConfig{
		Input:          io.NopCloser(strings.NewReader("")),
		ProtocolOutput: nopWriteCloser{Writer: io.Discard},
		LogOutput:      io.Discard,
	})
	if err != nil {
		t.Fatal(err)
	}
	pending, err := server.Connection().RegisterPending("request-1")
	if err != nil {
		t.Fatal(err)
	}
	if err := server.Serve(context.Background()); err != nil {
		t.Fatalf("Serve() error = %v", err)
	}
	if got := server.Connection().PendingCount(); got != 0 {
		t.Fatalf("pending = %d, want 0", got)
	}
	if _, ok := <-pending; ok {
		t.Fatal("pending response channel remains open")
	}
}

func TestFrameWriterCancellationClosesBlockedOutput(t *testing.T) {
	output := newBlockingWriteCloser()
	writer := NewFrameWriter(output)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	err := writer.WriteFrame(ctx, ProfileNegotiation{Version: HostProtocolVersionV1})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("WriteFrame() error = %v, want deadline exceeded", err)
	}
	if !output.Closed() {
		t.Fatal("blocked output was not closed")
	}
}

func TestFrameWriterReturnsSuccessWhenFrameWrittenBeforeCancellation(t *testing.T) {
	output := newCompleteThenBlockWriteCloser()
	writer := NewFrameWriter(output)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	err := writer.WriteFrame(ctx, ProfileNegotiation{Version: HostProtocolVersionV1})
	if err != nil {
		t.Fatalf("WriteFrame() error = %v, want nil after complete frame write", err)
	}
	want := string(mustJSON(t, ProfileNegotiation{Version: HostProtocolVersionV1})) + "\n"
	if got := output.String(); got != want {
		t.Fatalf("protocol output = %q, want complete frame %q", got, want)
	}
}

func TestServerClassifiesBlockedNegotiationOutput(t *testing.T) {
	input := bytes.NewBuffer(append(mustJSON(t, ProfileNegotiation{Version: HostProtocolVersionV1}), '\n'))
	output := newBlockingWriteCloser()
	server, err := NewServer(host.New(nil), ServerConfig{
		Input:           io.NopCloser(input),
		ProtocolOutput:  output,
		LogOutput:       io.Discard,
		DeliveryTimeout: 20 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}

	err = server.Serve(context.Background())
	if !errors.Is(err, host.ErrDeliveryTimeout) {
		t.Fatalf("Serve() error = %v, want %v", err, host.ErrDeliveryTimeout)
	}
	if got := ClassifyExit(err); got != ExitOutputFailure {
		t.Fatalf("ClassifyExit() = %q, want %q", got, ExitOutputFailure)
	}
	if !output.Closed() {
		t.Fatal("blocked protocol output was not closed")
	}
}

func TestServerParentCancellationInterruptsInputAndClassifiesExit(t *testing.T) {
	input := newBlockingReadCloser()
	server, err := NewServer(host.New(nil), ServerConfig{
		Input:          input,
		ProtocolOutput: nopWriteCloser{Writer: io.Discard},
		LogOutput:      io.Discard,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- server.Serve(ctx) }()
	<-input.started
	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Serve() error = %v, want context canceled", err)
		}
		if got := ClassifyExit(err); got != ExitParentCanceled {
			t.Fatalf("ClassifyExit() = %q, want %q", got, ExitParentCanceled)
		}
	case <-time.After(time.Second):
		t.Fatal("Serve did not return after parent cancellation")
	}
	if !input.Closed() {
		t.Fatal("input was not closed on parent cancellation")
	}
}

func validActionCommand() types.HostCommandEnvelope {
	return types.HostCommandEnvelope{
		Version: types.HostProtocolVersionV1, MessageID: "message-1", Kind: types.HostCommandKindAction,
		Time: time.Unix(1, 0).UTC(), RequestID: "request-1", SessionID: "session-1", RunID: "run-1",
		Payload: map[string]any{"action": string(types.ProtocolActionCancel)},
	}
}

type countingControl struct{ calls atomic.Int32 }

func (c *countingControl) ExecuteAction(context.Context, types.HostCommandEnvelope, types.ProtocolAction) (types.HostAdmissionStatus, string, error) {
	c.calls.Add(1)
	return types.HostAdmissionStatusAccepted, "", nil
}

func (c *countingControl) IngestRealtime(context.Context, types.HostCommandEnvelope, types.RealtimeEventEnvelope) (types.HostAdmissionStatus, string, error) {
	c.calls.Add(1)
	return types.HostAdmissionStatusAccepted, "", nil
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func decodeAllFrames(t *testing.T, raw []byte) []map[string]any {
	t.Helper()
	decoder := NewDecoder(bytes.NewReader(raw))
	var frames []map[string]any
	for {
		frame, err := decoder.Next()
		if errors.Is(err, io.EOF) {
			return frames
		}
		if err != nil {
			t.Fatalf("decode protocol output: %v", err)
		}
		var decoded map[string]any
		if err := json.Unmarshal(frame, &decoded); err != nil {
			t.Fatal(err)
		}
		frames = append(frames, decoded)
	}
}

type nopWriteCloser struct{ io.Writer }

func (nopWriteCloser) Close() error { return nil }

type errorWriter struct{ err error }

func (w errorWriter) Write([]byte) (int, error) { return 0, w.err }

type trackingReadCloser struct {
	io.Reader
	closed atomic.Bool
}

func (r *trackingReadCloser) Close() error { r.closed.Store(true); return nil }
func (r *trackingReadCloser) Closed() bool { return r.closed.Load() }

type trackingWriteCloser struct {
	io.Writer
	closed atomic.Bool
}

func (w *trackingWriteCloser) Close() error { w.closed.Store(true); return nil }
func (w *trackingWriteCloser) Closed() bool { return w.closed.Load() }

type blockingWriteCloser struct {
	started chan struct{}
	closed  chan struct{}
	once    sync.Once
}

type completeThenBlockWriteCloser struct {
	started chan struct{}
	closed  chan struct{}
	once    sync.Once
	mu      sync.Mutex
	data    bytes.Buffer
}

func newCompleteThenBlockWriteCloser() *completeThenBlockWriteCloser {
	return &completeThenBlockWriteCloser{started: make(chan struct{}), closed: make(chan struct{})}
}

func (w *completeThenBlockWriteCloser) Write(p []byte) (int, error) {
	w.mu.Lock()
	_, _ = w.data.Write(p)
	w.mu.Unlock()
	w.once.Do(func() { close(w.started) })
	<-w.closed
	return len(p), nil
}

func (w *completeThenBlockWriteCloser) Close() error {
	select {
	case <-w.closed:
	default:
		close(w.closed)
	}
	return nil
}

func (w *completeThenBlockWriteCloser) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.data.String()
}

func newBlockingWriteCloser() *blockingWriteCloser {
	return &blockingWriteCloser{started: make(chan struct{}), closed: make(chan struct{})}
}

func (w *blockingWriteCloser) Write([]byte) (int, error) {
	w.once.Do(func() { close(w.started) })
	<-w.closed
	return 0, io.ErrClosedPipe
}

func (w *blockingWriteCloser) Close() error {
	select {
	case <-w.closed:
	default:
		close(w.closed)
	}
	return nil
}

func (w *blockingWriteCloser) Closed() bool {
	select {
	case <-w.closed:
		return true
	default:
		return false
	}
}

type blockingReadCloser struct {
	started chan struct{}
	closed  chan struct{}
	once    sync.Once
}

func newBlockingReadCloser() *blockingReadCloser {
	return &blockingReadCloser{started: make(chan struct{}), closed: make(chan struct{})}
}

func (r *blockingReadCloser) Read([]byte) (int, error) {
	r.once.Do(func() { close(r.started) })
	<-r.closed
	return 0, io.ErrClosedPipe
}

func (r *blockingReadCloser) Close() error {
	select {
	case <-r.closed:
	default:
		close(r.closed)
	}
	return nil
}

func (r *blockingReadCloser) Closed() bool {
	select {
	case <-r.closed:
		return true
	default:
		return false
	}
}
