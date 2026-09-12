package host

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
)

type pendingAwaitResult struct {
	response types.HostResponseEnvelope
	err      error
}

func correlatedHostResponse(requestID string) types.HostResponseEnvelope {
	return types.HostResponseEnvelope{
		Version:   types.HostProtocolVersionV1,
		MessageID: "response-" + requestID,
		Kind:      types.HostEnvelopeKindHostResponse,
		Time:      time.Now().UTC(),
		RequestID: requestID,
		SessionID: "session-1",
		RunID:     "run-1",
		Accepted:  true,
	}
}

func TestPendingEstablishedResponseWinsOverCancellationAndDisconnect(t *testing.T) {
	for iteration := 0; iteration < 100; iteration++ {
		requestID := fmt.Sprintf("established-%d", iteration)
		conn := New(nil).Connect(func(any) error { return nil })
		entry, err := conn.registerPending(requestID, "session-1", "run-1")
		if err != nil {
			t.Fatal(err)
		}
		response := correlatedHostResponse(requestID)
		if err := conn.Respond(response); err != nil {
			t.Fatal(err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		conn.Close()
		got, err := conn.awaitPending(ctx, time.Nanosecond, requestID, entry)
		if err != nil || got.MessageID != response.MessageID {
			t.Fatalf("iteration %d: response=%#v err=%v", iteration, got, err)
		}
		if conn.PendingCount() != 0 {
			t.Fatalf("iteration %d: pending=%d", iteration, conn.PendingCount())
		}
	}
}

func TestPendingConcurrentResponseAndCancellationHaveOneOutcome(t *testing.T) {
	for iteration := 0; iteration < 500; iteration++ {
		requestID := fmt.Sprintf("cancel-race-%d", iteration)
		conn := New(nil).Connect(func(any) error { return nil })
		entry, err := conn.registerPending(requestID, "session-1", "run-1")
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		response := correlatedHostResponse(requestID)
		start := make(chan struct{})
		awaited := make(chan pendingAwaitResult, 1)
		responded := make(chan error, 1)
		go func() {
			<-start
			got, err := conn.awaitPending(ctx, time.Second, requestID, entry)
			awaited <- pendingAwaitResult{response: got, err: err}
		}()
		go func() {
			<-start
			responded <- conn.Respond(response)
		}()
		close(start)
		cancel()

		respondErr := <-responded
		result := <-awaited
		switch {
		case respondErr == nil:
			if result.err != nil || result.response.MessageID != response.MessageID {
				t.Fatalf("iteration %d: response won but waiter got response=%#v err=%v", iteration, result.response, result.err)
			}
		case errors.Is(respondErr, ErrPendingNotFound):
			if !errors.Is(result.err, context.Canceled) {
				t.Fatalf("iteration %d: cancellation won but waiter err=%v", iteration, result.err)
			}
		default:
			t.Fatalf("iteration %d: unexpected response error=%v", iteration, respondErr)
		}
		if conn.PendingCount() != 0 {
			t.Fatalf("iteration %d: pending=%d", iteration, conn.PendingCount())
		}
		conn.Close()
	}
}

func TestPendingConcurrentResponseAndTimeoutHaveOneOutcome(t *testing.T) {
	for iteration := 0; iteration < 300; iteration++ {
		requestID := fmt.Sprintf("timeout-race-%d", iteration)
		conn := New(nil).Connect(func(any) error { return nil })
		entry, err := conn.registerPending(requestID, "session-1", "run-1")
		if err != nil {
			t.Fatal(err)
		}
		response := correlatedHostResponse(requestID)
		awaited := make(chan pendingAwaitResult, 1)
		go func() {
			got, err := conn.awaitPending(context.Background(), time.Nanosecond, requestID, entry)
			awaited <- pendingAwaitResult{response: got, err: err}
		}()
		respondErr := conn.Respond(response)
		result := <-awaited
		switch {
		case respondErr == nil:
			if result.err != nil || result.response.MessageID != response.MessageID {
				t.Fatalf("iteration %d: response won but waiter got response=%#v err=%v", iteration, result.response, result.err)
			}
		case errors.Is(respondErr, ErrPendingNotFound):
			if !errors.Is(result.err, context.DeadlineExceeded) {
				t.Fatalf("iteration %d: timeout won but waiter err=%v", iteration, result.err)
			}
		default:
			t.Fatalf("iteration %d: unexpected response error=%v", iteration, respondErr)
		}
		if conn.PendingCount() != 0 {
			t.Fatalf("iteration %d: pending=%d", iteration, conn.PendingCount())
		}
		conn.Close()
	}
}

func TestPendingConcurrentResponseAndDisconnectHaveOneOutcome(t *testing.T) {
	for iteration := 0; iteration < 500; iteration++ {
		requestID := fmt.Sprintf("disconnect-race-%d", iteration)
		conn := New(nil).Connect(func(any) error { return nil })
		entry, err := conn.registerPending(requestID, "session-1", "run-1")
		if err != nil {
			t.Fatal(err)
		}
		response := correlatedHostResponse(requestID)
		start := make(chan struct{})
		awaited := make(chan pendingAwaitResult, 1)
		responded := make(chan error, 1)
		go func() {
			<-start
			got, err := conn.awaitPending(context.Background(), time.Second, requestID, entry)
			awaited <- pendingAwaitResult{response: got, err: err}
		}()
		go func() {
			<-start
			responded <- conn.Respond(response)
		}()
		close(start)
		conn.Close()

		respondErr := <-responded
		result := <-awaited
		switch {
		case respondErr == nil:
			if result.err != nil || result.response.MessageID != response.MessageID {
				t.Fatalf("iteration %d: response won but waiter got response=%#v err=%v", iteration, result.response, result.err)
			}
		case errors.Is(respondErr, ErrPendingNotFound):
			if !errors.Is(result.err, ErrConnectionClosed) {
				t.Fatalf("iteration %d: disconnect won but waiter err=%v", iteration, result.err)
			}
		default:
			t.Fatalf("iteration %d: unexpected response error=%v", iteration, respondErr)
		}
		if conn.PendingCount() != 0 {
			t.Fatalf("iteration %d: pending=%d", iteration, conn.PendingCount())
		}
	}
}

func TestPendingConcurrentDuplicateResponsesAcceptExactlyOne(t *testing.T) {
	conn := New(nil).Connect(func(any) error { return nil })
	entry, err := conn.registerPending("duplicate-race", "session-1", "run-1")
	if err != nil {
		t.Fatal(err)
	}
	first := correlatedHostResponse("duplicate-race")
	first.MessageID = "response-first"
	second := correlatedHostResponse("duplicate-race")
	second.MessageID = "response-second"
	start := make(chan struct{})
	errorsByMessage := make(chan struct {
		messageID string
		err       error
	}, 2)
	for _, response := range []types.HostResponseEnvelope{first, second} {
		response := response
		go func() {
			<-start
			errorsByMessage <- struct {
				messageID string
				err       error
			}{messageID: response.MessageID, err: conn.Respond(response)}
		}()
	}
	close(start)
	results := []struct {
		messageID string
		err       error
	}{<-errorsByMessage, <-errorsByMessage}
	accepted := ""
	for _, result := range results {
		if result.err == nil {
			if accepted != "" {
				t.Fatalf("two responses accepted: %q and %q", accepted, result.messageID)
			}
			accepted = result.messageID
			continue
		}
		if !errors.Is(result.err, ErrPendingNotFound) {
			t.Fatalf("response %q error=%v", result.messageID, result.err)
		}
	}
	if accepted == "" {
		t.Fatal("no response accepted")
	}
	got, err := conn.awaitPending(context.Background(), time.Second, "duplicate-race", entry)
	if err != nil || got.MessageID != accepted {
		t.Fatalf("winner=%q waiter response=%#v err=%v", accepted, got, err)
	}
	if conn.PendingCount() != 0 {
		t.Fatalf("pending=%d", conn.PendingCount())
	}
	conn.Close()
}

func TestPendingConcurrentResponseAndOutputFailureHaveOneOutcome(t *testing.T) {
	for iteration := 0; iteration < 100; iteration++ {
		writeStarted := make(chan struct{})
		releaseWrite := make(chan struct{})
		writeErr := fmt.Errorf("output failed %d", iteration)
		conn := New(nil, WithDeliveryTimeout(time.Second)).Connect(func(any) error {
			close(writeStarted)
			<-releaseWrite
			return writeErr
		})
		requestID := fmt.Sprintf("output-race-%d", iteration)
		resolved := make(chan pendingAwaitResult, 1)
		go func() {
			got, err := conn.Resolve(context.Background(), types.ClarificationResolveRequest{
				RunID: "run-1", SessionID: "session-1", Timeout: time.Second,
				Request: types.ClarificationRequest{RequestID: requestID, Questions: []string{"continue?"}},
			})
			resolved <- pendingAwaitResult{
				response: types.HostResponseEnvelope{RequestID: got.RequestID},
				err:      err,
			}
		}()
		<-writeStarted
		response := correlatedHostResponse(requestID)
		response.Payload = map[string]any{"answers": []any{"yes"}}
		start := make(chan struct{})
		responded := make(chan error, 1)
		go func() {
			<-start
			responded <- conn.Respond(response)
		}()
		close(start)
		close(releaseWrite)

		respondErr := <-responded
		result := <-resolved
		switch {
		case respondErr == nil:
			if result.err != nil || result.response.RequestID != requestID {
				t.Fatalf("iteration %d: response won but resolver result=%#v err=%v", iteration, result.response, result.err)
			}
		case errors.Is(respondErr, ErrPendingNotFound):
			if !errors.Is(result.err, writeErr) {
				t.Fatalf("iteration %d: output failure won but resolver err=%v", iteration, result.err)
			}
		default:
			t.Fatalf("iteration %d: unexpected response error=%v", iteration, respondErr)
		}
		if conn.PendingCount() != 0 {
			t.Fatalf("iteration %d: pending=%d", iteration, conn.PendingCount())
		}
	}
}

func TestOutputFailurePreservesWinningCauseForPendingRequest(t *testing.T) {
	writeErr := errors.New("protocol output failed")
	for iteration := 0; iteration < 50; iteration++ {
		conn := New(nil, WithDeliveryTimeout(time.Second)).Connect(func(any) error { return writeErr })
		_, err := conn.Resolve(context.Background(), types.ClarificationResolveRequest{
			RunID:     "run-1",
			SessionID: "session-1",
			Timeout:   time.Second,
			Request: types.ClarificationRequest{
				RequestID: fmt.Sprintf("write-failure-%d", iteration),
				Questions: []string{"continue?"},
			},
		})
		if !errors.Is(err, writeErr) {
			t.Fatalf("iteration %d: resolver err=%v, want write cause", iteration, err)
		}
		if conn.PendingCount() != 0 {
			t.Fatalf("iteration %d: pending=%d", iteration, conn.PendingCount())
		}
	}
}

func TestConcurrentCloseSettlesAllPendingWithOneCause(t *testing.T) {
	conn := New(nil).Connect(func(any) error { return nil })
	const pendingCount = 32
	entries := make([]*pendingRequest, 0, pendingCount)
	for index := 0; index < pendingCount; index++ {
		entry, err := conn.registerPending(fmt.Sprintf("close-all-%d", index), "session-1", "run-1")
		if err != nil {
			t.Fatal(err)
		}
		entries = append(entries, entry)
	}
	var closers sync.WaitGroup
	for index := 0; index < 16; index++ {
		closers.Add(1)
		go func() {
			defer closers.Done()
			conn.Close()
		}()
	}
	closers.Wait()
	for index, entry := range entries {
		got, err := conn.awaitPending(context.Background(), time.Second, fmt.Sprintf("close-all-%d", index), entry)
		if !errors.Is(err, ErrConnectionClosed) || got.MessageID != "" {
			t.Fatalf("entry %d: response=%#v err=%v", index, got, err)
		}
	}
	if conn.PendingCount() != 0 {
		t.Fatalf("pending=%d", conn.PendingCount())
	}
}
