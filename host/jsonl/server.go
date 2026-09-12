package jsonl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/host"
)

var (
	ErrNegotiationRequired = errors.New("jsonl: protocol negotiation required")
	ErrOutputFailure       = errors.New("jsonl: protocol output failure")
)

const DefaultDeliveryTimeout = 5 * time.Second

// ExitClassification is the stable process-level result of serving a JSONL
// connection. An executable can map these values to exit codes without
// parsing implementation-specific error strings.
type ExitClassification string

const (
	ExitSuccess            ExitClassification = "success"
	ExitMalformedInput     ExitClassification = "malformed_input"
	ExitFrameTooLarge      ExitClassification = "frame_too_large"
	ExitUnsupportedVersion ExitClassification = "unsupported_version"
	ExitNegotiation        ExitClassification = "negotiation_required"
	ExitOutputFailure      ExitClassification = "output_failure"
	ExitParentCanceled     ExitClassification = "parent_canceled"
	ExitProtocol           ExitClassification = "protocol_failure"
)

// ClassifyExit gives callers a deterministic classification while preserving
// the original error for errors.Is and diagnostics.
func ClassifyExit(err error) ExitClassification {
	switch {
	case err == nil:
		return ExitSuccess
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		return ExitParentCanceled
	case errors.Is(err, ErrFrameTooLarge):
		return ExitFrameTooLarge
	case errors.Is(err, ErrEmptyFrame), errors.Is(err, ErrMalformedFrame), errors.Is(err, ErrPartialFrame):
		return ExitMalformedInput
	case errors.Is(err, ErrUnsupportedVersion):
		return ExitUnsupportedVersion
	case errors.Is(err, ErrNegotiationRequired):
		return ExitNegotiation
	case errors.Is(err, ErrOutputFailure), errors.Is(err, io.ErrClosedPipe), errors.Is(err, io.ErrShortWrite), errors.Is(err, host.ErrDeliveryTimeout):
		return ExitOutputFailure
	default:
		return ExitProtocol
	}
}

// ServerConfig separates protocol output from diagnostics. Defaults reserve
// stdout exclusively for protocol frames and route logs to stderr.
type ServerConfig struct {
	Input           io.ReadCloser
	ProtocolOutput  io.WriteCloser
	LogOutput       io.Writer
	MaxFrameBytes   int
	DeliveryTimeout time.Duration
}

func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		Input: os.Stdin, ProtocolOutput: os.Stdout, LogOutput: os.Stderr,
		MaxFrameBytes: DefaultMaxFrameBytes, DeliveryTimeout: DefaultDeliveryTimeout,
	}
}

type Server struct {
	input           io.ReadCloser
	logOutput       io.Writer
	deliveryTimeout time.Duration
	decoder         *Decoder
	writer          *FrameWriter
	connection      *host.Connection
	closeOnce       sync.Once
}

// NewServer creates one connection and one serialized protocol writer. The
// coordinator remains the only command-response and pending-correlation owner.
func NewServer(coordinator *host.Coordinator, config ServerConfig) (*Server, error) {
	defaults := DefaultServerConfig()
	if config.Input == nil {
		config.Input = defaults.Input
	}
	if config.ProtocolOutput == nil {
		config.ProtocolOutput = defaults.ProtocolOutput
	}
	if config.LogOutput == nil {
		config.LogOutput = defaults.LogOutput
	}
	if config.MaxFrameBytes == 0 {
		config.MaxFrameBytes = defaults.MaxFrameBytes
	}
	if config.DeliveryTimeout == 0 {
		config.DeliveryTimeout = defaults.DeliveryTimeout
	}
	if config.DeliveryTimeout < 0 {
		return nil, errors.New("jsonl: delivery timeout must be positive")
	}
	if coordinator == nil {
		return nil, errors.New("jsonl: host coordinator required")
	}
	decoder, err := NewDecoderWithError(config.Input, WithMaxFrameBytes(config.MaxFrameBytes))
	if err != nil {
		return nil, err
	}
	writer := NewFrameWriter(config.ProtocolOutput)
	server := &Server{input: config.Input, logOutput: config.LogOutput, deliveryTimeout: config.DeliveryTimeout, decoder: decoder, writer: writer}
	server.connection = coordinator.ConnectFrameWriter(writer)
	return server, nil
}

// Connection exposes the transport-neutral connection for installing the
// existing HITL resolvers. The Server still owns its lifetime.
func (s *Server) Connection() *host.Connection {
	if s == nil {
		return nil
	}
	return s.connection
}

// Serve reads a negotiated connection until clean EOF or the first failure.
// It never writes diagnostics to the protocol output.
func (s *Server) Serve(ctx context.Context) (resultErr error) {
	if s == nil || s.decoder == nil || s.connection == nil {
		return errors.New("jsonl: server is not initialized")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	watchDone := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			s.Close()
		case <-watchDone:
		}
	}()
	defer close(watchDone)
	defer func() {
		s.Close()
		if resultErr != nil && s.logOutput != nil {
			_, _ = fmt.Fprintf(s.logOutput, "host/jsonl: %s: %v\n", ClassifyExit(resultErr), resultErr)
		}
	}()

	frame, err := s.decoder.Next()
	if err != nil {
		return s.readResult(ctx, err)
	}
	var negotiationFields map[string]json.RawMessage
	if err := json.Unmarshal(frame, &negotiationFields); err != nil {
		return fmt.Errorf("%w: negotiation: %v", ErrMalformedFrame, err)
	}
	if len(negotiationFields) != 1 || negotiationFields["version"] == nil {
		return ErrNegotiationRequired
	}
	var negotiation ProfileNegotiation
	if err := json.Unmarshal(frame, &negotiation); err != nil {
		return fmt.Errorf("%w: negotiation: %v", ErrMalformedFrame, err)
	}
	if negotiation.Version == "" {
		return ErrNegotiationRequired
	}
	if err := NegotiateProfile(negotiation.Version); err != nil {
		return err
	}
	writeCtx, cancelWrite := s.deliveryContext(ctx)
	if err := s.writer.WriteFrame(writeCtx, negotiation); err != nil {
		cancelWrite()
		if errors.Is(err, context.DeadlineExceeded) && ctx.Err() == nil {
			return host.ErrDeliveryTimeout
		}
		return err
	}
	cancelWrite()

	for {
		frame, err = s.decoder.Next()
		if err != nil {
			return s.readResult(ctx, err)
		}
		var command types.HostCommandEnvelope
		if err := json.Unmarshal(frame, &command); err != nil {
			return fmt.Errorf("%w: command: %v", ErrMalformedFrame, err)
		}
		if _, err := s.connection.HandleCommand(ctx, command); err != nil {
			return err
		}
	}
}

func (s *Server) deliveryContext(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, s.deliveryTimeout)
}

func (s *Server) readResult(ctx context.Context, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	if errors.Is(err, io.EOF) {
		return nil
	}
	return err
}

func (s *Server) Close() {
	if s == nil {
		return
	}
	s.closeOnce.Do(func() {
		if s.input != nil {
			_ = s.input.Close()
		}
		if s.connection != nil {
			s.connection.Close()
		} else if s.writer != nil {
			_ = s.writer.Close()
		}
	})
}

// FrameWriter marshals host envelopes and delegates framing, serialization,
// partial-write handling, and flushing to Writer. Closing an underlying output
// is the cancellation mechanism for transports without write deadlines.
type FrameWriter struct {
	writer *Writer
	output io.WriteCloser
	closed chan struct{}
	once   sync.Once
}

func NewFrameWriter(output io.WriteCloser) *FrameWriter {
	// The closeable adapter owns cancellation. Hide optional deadline methods
	// here because ordinary stdout files expose SetWriteDeadline but reject it.
	return &FrameWriter{writer: NewWriter(writeOnly{Writer: output}), output: output, closed: make(chan struct{})}
}

type writeOnly struct{ io.Writer }

func (w *FrameWriter) WriteFrame(ctx context.Context, frame any) error {
	if w == nil || w.writer == nil || w.output == nil {
		return io.ErrClosedPipe
	}
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-w.closed:
		return io.ErrClosedPipe
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	raw, err := json.Marshal(frame)
	if err != nil {
		return fmt.Errorf("jsonl: marshal frame: %w", err)
	}
	stopClose := context.AfterFunc(ctx, func() { _ = w.Close() })
	err = w.writer.WriteFrameContext(ctx, raw)
	stopClose()
	if err == nil {
		return nil
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	return fmt.Errorf("%w: %w", ErrOutputFailure, err)
}

func (w *FrameWriter) Close() error {
	if w == nil {
		return nil
	}
	var closeErr error
	w.once.Do(func() {
		close(w.closed)
		if w.output != nil {
			closeErr = w.output.Close()
		}
	})
	return closeErr
}
