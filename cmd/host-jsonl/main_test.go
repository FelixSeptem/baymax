package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/host/jsonl"
)

var testBinary string
var conformanceBinary string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "host-jsonl-command-")
	if err != nil {
		panic(err)
	}
	testBinary = filepath.Join(dir, "host-jsonl")
	if runtime.GOOS == "windows" {
		testBinary += ".exe"
	}
	conformanceBinary = filepath.Join(dir, "host-jsonl-conformance-helper")
	if runtime.GOOS == "windows" {
		conformanceBinary += ".exe"
	}
	build := exec.Command("go", "build", "-o", testBinary, ".")
	build.Env = append(os.Environ(), "GOCACHE="+filepath.Join(dir, "go-cache"))
	if output, buildErr := build.CombinedOutput(); buildErr != nil {
		_, _ = os.Stderr.Write(output)
		_ = os.RemoveAll(dir)
		os.Exit(1)
	}
	helperBuild := exec.Command("go", "build", "-o", conformanceBinary, "./conformance-helper")
	helperBuild.Dir = "."
	helperBuild.Env = append(os.Environ(), "GOCACHE="+filepath.Join(dir, "go-cache-helper"))
	if output, buildErr := helperBuild.CombinedOutput(); buildErr != nil {
		_, _ = os.Stderr.Write(output)
		_ = os.RemoveAll(dir)
		os.Exit(1)
	}
	code := m.Run()
	if cleanupErr := os.RemoveAll(dir); cleanupErr != nil && code == 0 {
		_, _ = os.Stderr.WriteString("host-jsonl test cleanup: " + cleanupErr.Error() + "\n")
		code = 1
	}
	os.Exit(code)
}

func TestConformanceHelperPreservesUnicodeAndSerializesFrames(t *testing.T) {
	command := validRunStartCommand("run-unicode")
	stdout, stderr, exitCode := runInteractive(t, "default", []types.HostCommandEnvelope{command}, true)
	if exitCode != 0 {
		t.Fatalf("exit code=%d stderr=%q stdout=%q", exitCode, stderr, stdout)
	}
	frames := decodeProtocolFrames(t, stdout)
	if len(frames) < 3 {
		t.Fatalf("frames=%d want negotiation, response, event; stderr=%q", len(frames), stderr)
	}
	if frames[0]["version"] != types.HostProtocolVersionV1 || frames[1]["kind"] != string(types.HostEnvelopeKindCommandResponse) {
		t.Fatalf("initial frames=%v", frames[:2])
	}
	if frames[2]["kind"] != string(types.HostEnvelopeKindRuntimeEvent) {
		t.Fatalf("event frame=%v", frames[2])
	}
	event, ok := frames[2]["event"].(map[string]any)
	if !ok || event["payload"].(map[string]any)["unicode"] != "a\u2028b\u2029c" {
		t.Fatalf("unicode event payload=%v", frames[2]["event"])
	}
}

func TestConformanceHelperSerializesConcurrentEventsWithoutInterleaving(t *testing.T) {
	command := validRunStartCommand("run-serialized")
	stdout, stderr, exitCode := runInteractive(t, "serialized", []types.HostCommandEnvelope{command}, true)
	if exitCode != 0 {
		t.Fatalf("exit code=%d stderr=%q stdout=%q", exitCode, stderr, stdout)
	}
	frames := decodeProtocolFrames(t, stdout)
	if len(frames) != 34 {
		t.Fatalf("frames=%d want negotiation + response + 32 events", len(frames))
	}
	for i, frame := range frames {
		if i == 0 {
			continue
		}
		if i == 1 {
			if frame["kind"] != string(types.HostEnvelopeKindCommandResponse) {
				t.Fatalf("response frame=%v", frame)
			}
			continue
		}
		if frame["kind"] != string(types.HostEnvelopeKindRuntimeEvent) {
			t.Fatalf("event frame %d=%v", i, frame)
		}
	}
}

func TestConformanceHelperEOFSettlesPendingRequest(t *testing.T) {
	command := validRunStartCommand("run-pending")
	stdout, stderr, exitCode := runInteractive(t, "pending", []types.HostCommandEnvelope{command}, true)
	if exitCode != 0 {
		t.Fatalf("exit code=%d stderr=%q stdout=%q", exitCode, stderr, stdout)
	}
	if !strings.Contains(stderr, "pending_result=host: connection closed") {
		t.Fatalf("stderr=%q, want disconnected pending result", stderr)
	}
	frames := decodeProtocolFrames(t, stdout)
	if len(frames) < 3 {
		t.Fatalf("frames=%d want negotiation, response, host request; stderr=%q", len(frames), stderr)
	}
	if frames[1]["kind"] != string(types.HostEnvelopeKindCommandResponse) || frames[2]["kind"] != string(types.HostEnvelopeKindHostRequest) {
		t.Fatalf("frames=%v", frames)
	}
}

func TestConformanceHelperBackpressureExitsDeterministically(t *testing.T) {
	command := validRunStartCommand("run-burst")
	stdout, stderr, exitCode := runInteractive(t, "burst", []types.HostCommandEnvelope{command}, false)
	if exitCode != 0 {
		t.Fatalf("exit code=%d, want deterministic completion; stderr=%q", exitCode, stderr)
	}
	if !strings.Contains(stderr, "burst_result=returned") {
		t.Fatalf("stderr=%q, want bounded callback completion marker", stderr)
	}
	if len(stdout) > 1<<20 {
		t.Fatalf("stdout=%d bytes, want bounded output under backpressure", len(stdout))
	}
}

func validRunStartCommand(runID string) types.HostCommandEnvelope {
	return types.HostCommandEnvelope{Version: types.HostProtocolVersionV1, MessageID: "message-" + runID, Kind: types.HostCommandKindRunStart, Time: time.Now().UTC(), RequestID: "request-" + runID, SessionID: "session-" + runID, RunID: runID, Payload: map[string]any{"input": "hello"}}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func runInteractive(t *testing.T, mode string, commands []types.HostCommandEnvelope, drain bool) (string, string, int) {
	t.Helper()
	command := exec.Command(conformanceBinary, "-mode", mode)
	stdin, err := command.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdoutPipe, err := command.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	command.Stderr = &stderr
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	negotiation := append(mustJSON(t, map[string]string{"version": types.HostProtocolVersionV1}), '\n')
	if _, err := stdin.Write(negotiation); err != nil {
		t.Fatal(err)
	}
	readDone := make(chan struct{})
	if drain {
		go func() {
			_, _ = io.Copy(&stdout, stdoutPipe)
			close(readDone)
		}()
	} else {
		// Keep the read end open but unread so the child exercises bounded
		// protocol backpressure and its delivery deadline.
		readDone = nil
	}
	for _, cmd := range commands {
		if _, err := stdin.Write(append(mustJSON(t, cmd), '\n')); err != nil {
			t.Fatal(err)
		}
	}
	_ = drain
	time.Sleep(100 * time.Millisecond)
	if err := stdin.Close(); err != nil {
		t.Fatal(err)
	}
	err = command.Wait()
	if readDone != nil {
		<-readDone
	}
	if err == nil {
		return stdout.String(), stderr.String(), 0
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatal(err)
	}
	return stdout.String(), stderr.String(), exitErr.ExitCode()
}

func decodeProtocolFrames(t *testing.T, raw string) []map[string]any {
	t.Helper()
	decoder := jsonl.NewDecoder(strings.NewReader(raw))
	var frames []map[string]any
	for {
		frame, err := decoder.Next()
		if errors.Is(err, io.EOF) {
			return frames
		}
		if err != nil {
			t.Fatalf("decode frame: %v; output=%q", err, raw)
		}
		var decoded map[string]any
		if err := json.Unmarshal(frame, &decoded); err != nil {
			t.Fatal(err)
		}
		frames = append(frames, decoded)
	}
}

func TestExecutableNegotiationAndCleanEOFUseProtocolStdoutOnly(t *testing.T) {
	stdout, stderr, exitCode := runCommand(t, `{"version":"embedded_host_protocol.v1"}`+"\n")
	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0; stderr=%q", exitCode, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if !strings.HasSuffix(stdout, "\n") || strings.Contains(stdout, "\r\n") {
		t.Fatalf("stdout = %q, want one raw-LF frame", stdout)
	}
	lines := strings.Split(strings.TrimSuffix(stdout, "\n"), "\n")
	if len(lines) != 1 {
		t.Fatalf("stdout frames = %d, want 1; output=%q", len(lines), stdout)
	}
	var negotiation map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &negotiation); err != nil {
		t.Fatalf("stdout contains non-protocol data: %v; output=%q", err, stdout)
	}
	if negotiation["version"] != types.HostProtocolVersionV1 {
		t.Fatalf("negotiated version = %v", negotiation["version"])
	}
}

func TestExecutableMalformedInputUsesOnlyStderr(t *testing.T) {
	stdout, stderr, exitCode := runCommand(t, "accidental stdout log\n")
	if exitCode != 2 {
		t.Fatalf("exit code = %d, want 2; stderr=%q", exitCode, stderr)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want no invalid protocol frame", stdout)
	}
	if !strings.Contains(stderr, "malformed_input") {
		t.Fatalf("stderr = %q, want malformed_input classification", stderr)
	}
}

func TestExecutableUnsupportedVersionUsesOnlyStderr(t *testing.T) {
	stdout, stderr, exitCode := runCommand(t, `{"version":"embedded_host_protocol.v2"}`+"\n")
	if exitCode != 3 {
		t.Fatalf("exit code = %d, want 3; stderr=%q", exitCode, stderr)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want no invalid protocol frame", stdout)
	}
	if !strings.Contains(stderr, "unsupported_version") {
		t.Fatalf("stderr = %q, want unsupported_version classification", stderr)
	}
}

func TestExecutableOversizedFrameIsRejectedBeforeMutation(t *testing.T) {
	oversized := `{"payload":"` + strings.Repeat("x", jsonl.DefaultMaxFrameBytes) + `"}` + "\n"
	stdout, stderr, exitCode := runCommand(t, oversized)
	if exitCode != 2 {
		t.Fatalf("exit code = %d, want 2; stderr=%q", exitCode, stderr)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty on oversized frame", stdout)
	}
	if !strings.Contains(stderr, string(jsonl.ExitFrameTooLarge)) {
		t.Fatalf("stderr = %q, want frame-too-large classification", stderr)
	}
}

func runCommand(t *testing.T, stdin string) (string, string, int) {
	t.Helper()
	command := exec.Command(testBinary)
	command.Stdin = strings.NewReader(stdin)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	if err == nil {
		return stdout.String(), stderr.String(), 0
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("run command: %v", err)
	}
	return stdout.String(), stderr.String(), exitErr.ExitCode()
}
