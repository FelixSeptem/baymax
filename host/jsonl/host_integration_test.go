package jsonl_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/host"
	"github.com/FelixSeptem/baymax/host/jsonl"
)

type failingFlushTransport struct {
	bytes.Buffer
	err error
}

func (w *failingFlushTransport) Flush() error { return w.err }

func TestJSONLWriteFailureSettlesHostPendingExactlyOnce(t *testing.T) {
	writeErr := errors.New("jsonl flush failed")
	protocolWriter := jsonl.NewWriter(&failingFlushTransport{err: writeErr})
	connection := host.New(nil, host.WithDeliveryTimeout(time.Second)).Connect(func(frame any) error {
		encoded, err := json.Marshal(frame)
		if err != nil {
			return err
		}
		return protocolWriter.WriteFrame(encoded)
	})

	_, err := connection.Resolve(context.Background(), types.ClarificationResolveRequest{
		RunID: "run-1", SessionID: "session-1", Timeout: time.Second,
		Request: types.ClarificationRequest{RequestID: "clarification-1", Questions: []string{"continue?"}},
	})
	if !errors.Is(err, writeErr) {
		t.Fatalf("resolver err=%v want JSONL write failure", err)
	}
	if got := connection.PendingCount(); got != 0 {
		t.Fatalf("pending=%d want 0", got)
	}
	late := types.HostResponseEnvelope{
		Version: types.HostProtocolVersionV1, MessageID: "late-response", Kind: types.HostEnvelopeKindHostResponse,
		Time: time.Now().UTC(), RequestID: "clarification-1", SessionID: "session-1", RunID: "run-1",
	}
	if err := connection.Respond(late); !errors.Is(err, host.ErrPendingNotFound) {
		t.Fatalf("late response err=%v want %v", err, host.ErrPendingNotFound)
	}
}
