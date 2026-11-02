package sourceadapter

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sync"
)

// MaxStateBytes bounds the JSON envelope, including worst-case escaping of
// cursor tokens and the 200-entry reference-only receipt window.
const MaxStateBytes = 8 << 20

// FileStore supports one live instance per file, on the same host, with
// sequential reopen. It does not provide cross-instance or distributed CAS,
// nor a power-loss transaction guarantee. Do not externally mutate its file.
type FileStore struct {
	mu    sync.Mutex
	path  string
	state State
}

func OpenFileStore(path string, m Metadata) (*FileStore, error) {
	s, err := InitialState(m)
	if err != nil {
		return nil, err
	}
	if path == "" {
		return nil, ErrInvalid
	}
	path, err = filepath.Abs(path)
	if err != nil {
		return nil, ErrInvalid
	}
	f := &FileStore{path: path, state: s}
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		if err := f.write(s); err != nil {
			return nil, err
		}
		return f, nil
	}
	if err != nil {
		return nil, ErrDependency
	}
	if !info.Mode().IsRegular() || info.Size() > MaxStateBytes {
		return nil, ErrInvalid
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, ErrDependency
	}
	b, readErr := io.ReadAll(io.LimitReader(file, MaxStateBytes+1))
	closeErr := file.Close()
	if readErr != nil || closeErr != nil {
		return nil, ErrDependency
	}
	if len(b) > MaxStateBytes {
		return nil, ErrInvalid
	}
	if err := uniqueJSON(b); err != nil {
		return nil, err
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	// Decode existing files into a zero value. Constructor defaults must never
	// manufacture source binding/schema for an incomplete or null envelope.
	s = State{}
	if err := d.Decode(&s); err != nil {
		return nil, ErrInvalid
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return nil, ErrInvalid
	}
	if err := s.Validate(m); err != nil {
		return nil, err
	}
	f.state = cloneState(s)
	return f, nil
}

// uniqueJSON rejects duplicate keys and excessive nesting before struct decode.
func uniqueJSON(b []byte) error {
	d := json.NewDecoder(bytes.NewReader(b))
	var value func(int) error
	value = func(depth int) error {
		if depth > 16 {
			return ErrInvalid
		}
		token, err := d.Token()
		if err != nil {
			return ErrInvalid
		}
		delim, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		switch delim {
		case '{':
			keys := make(map[string]bool)
			for d.More() {
				k, err := d.Token()
				if err != nil {
					return ErrInvalid
				}
				key, ok := k.(string)
				if !ok || keys[key] {
					return ErrInvalid
				}
				keys[key] = true
				if err := value(depth + 1); err != nil {
					return err
				}
			}
			end, err := d.Token()
			if err != nil || end != json.Delim('}') {
				return ErrInvalid
			}
		case '[':
			for d.More() {
				if err := value(depth + 1); err != nil {
					return err
				}
			}
			end, err := d.Token()
			if err != nil || end != json.Delim(']') {
				return ErrInvalid
			}
		default:
			return ErrInvalid
		}
		return nil
	}
	if err := value(0); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return ErrInvalid
	}
	return nil
}

func (s *FileStore) write(next State) error {
	b, err := json.Marshal(next)
	if err != nil || len(b) > MaxStateBytes {
		return ErrInvalid
	}
	f, err := os.CreateTemp(filepath.Dir(s.path), ".source-adapter-*.tmp")
	if err != nil {
		return ErrDependency
	}
	name := f.Name()
	defer func() { _ = os.Remove(name) }()
	if err := f.Chmod(0600); err != nil {
		_ = f.Close()
		return ErrDependency
	}
	if _, err := f.Write(b); err != nil {
		_ = f.Close()
		return ErrDependency
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return ErrDependency
	}
	if err := f.Close(); err != nil {
		return ErrDependency
	}
	if err := os.Rename(name, s.path); err != nil {
		return ErrDependency
	}
	return nil
}

func (s *FileStore) Load(ctx context.Context) (State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return State{}, err
	}
	return cloneState(s.state), nil
}

func (s *FileStore) CompareAndSwap(ctx context.Context, expected uint64, next State) (State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return State{}, err
	}
	if err := validateCAS(s.state, expected, next); err != nil {
		return State{}, err
	}
	// Cancellation is checked at admission. Once writing starts it can commit;
	// callers must Load rather than infer rollback from a racing cancellation.
	if err := s.write(next); err != nil {
		return State{}, err
	}
	s.state = cloneState(next)
	return cloneState(s.state), nil
}
