package scheduler

import "errors"

// ErrDispatchBindingMismatch rejects reuse of a native task with another binding.
var ErrDispatchBindingMismatch = errors.New("scheduler dispatch binding mismatch")
