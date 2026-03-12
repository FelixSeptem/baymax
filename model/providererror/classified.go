package providererror

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/FelixSeptem/baymax/core/types"
)

// TODO(r3-m2): split provider-specific model errors into finer ErrorClass dimensions.
type Classified struct {
	Class     types.ErrorClass
	Reason    string
	Retryable bool
	Cause     error
}

func (e *Classified) Error() string {
	if e == nil {
		return ""
	}
	msg := "model provider error"
	if e.Cause != nil {
		msg = e.Cause.Error()
	}
	return fmt.Sprintf("%s (%s)", msg, e.Reason)
}

func (e *Classified) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func (e *Classified) ClassifiedError() *types.ClassifiedError {
	if e == nil {
		return nil
	}
	msg := e.Error()
	if e.Cause != nil {
		msg = e.Cause.Error()
	}
	return &types.ClassifiedError{
		Class:     e.Class,
		Message:   msg,
		Retryable: e.Retryable,
		Details: map[string]any{
			"provider_reason": e.Reason,
		},
	}
}

func FromStatusCode(err error, status int) error {
	switch {
	case status == 401 || status == 403:
		return &Classified{Class: types.ErrModel, Reason: "auth", Retryable: false, Cause: err}
	case status == 429:
		return &Classified{Class: types.ErrModel, Reason: "rate_limit", Retryable: true, Cause: err}
	case status == 400 || status == 404 || status == 422:
		return &Classified{Class: types.ErrModel, Reason: "request", Retryable: false, Cause: err}
	case status == 408 || status == 504:
		return &Classified{Class: types.ErrPolicyTimeout, Reason: "timeout", Retryable: true, Cause: err}
	case status >= 500:
		return &Classified{Class: types.ErrModel, Reason: "server", Retryable: true, Cause: err}
	default:
		return FromError(err)
	}
}

func FromError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return &Classified{Class: types.ErrPolicyTimeout, Reason: "timeout", Retryable: true, Cause: err}
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return &Classified{Class: types.ErrPolicyTimeout, Reason: "timeout", Retryable: true, Cause: err}
	}
	msg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(msg, "deadline exceeded"), strings.Contains(msg, "timeout"), strings.Contains(msg, "timed out"):
		return &Classified{Class: types.ErrPolicyTimeout, Reason: "timeout", Retryable: true, Cause: err}
	case strings.Contains(msg, "500"), strings.Contains(msg, "502"), strings.Contains(msg, "503"), strings.Contains(msg, "504"), strings.Contains(msg, "internal server"), strings.Contains(msg, "service unavailable"):
		return &Classified{Class: types.ErrModel, Reason: "server", Retryable: true, Cause: err}
	case strings.Contains(msg, "401"), strings.Contains(msg, "403"), strings.Contains(msg, "unauthorized"), strings.Contains(msg, "forbidden"), strings.Contains(msg, "authentication"):
		return &Classified{Class: types.ErrModel, Reason: "auth", Retryable: false, Cause: err}
	case strings.Contains(msg, "429"), strings.Contains(msg, "rate limit"), strings.Contains(msg, "quota"):
		return &Classified{Class: types.ErrModel, Reason: "rate_limit", Retryable: true, Cause: err}
	case strings.Contains(msg, "400"), strings.Contains(msg, "invalid argument"), strings.Contains(msg, "bad request"), strings.Contains(msg, "unprocessable"):
		return &Classified{Class: types.ErrModel, Reason: "request", Retryable: false, Cause: err}
	default:
		return &Classified{Class: types.ErrModel, Reason: "unknown", Retryable: false, Cause: err}
	}
}
