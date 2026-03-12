package provider

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

var ErrProviderNotReady = errors.New("context stage2 provider not ready")

type Request struct {
	RunID     string
	SessionID string
	Input     string
	MaxItems  int
}

type Response struct {
	Chunks []string
	Meta   map[string]any
}

type Provider interface {
	Name() string
	Fetch(ctx context.Context, req Request) (Response, error)
}

func New(name, filePath string) (Provider, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "file":
		return &fileProvider{path: strings.TrimSpace(filePath)}, nil
	case "rag", "db":
		return nil, fmt.Errorf("%w: provider=%s", ErrProviderNotReady, strings.ToLower(strings.TrimSpace(name)))
	default:
		return nil, fmt.Errorf("unsupported context stage2 provider %q", name)
	}
}

type fileProvider struct {
	path string
}

func (f *fileProvider) Name() string {
	return "file"
}

func (f *fileProvider) Fetch(ctx context.Context, req Request) (Response, error) {
	if strings.TrimSpace(f.path) == "" {
		return Response{}, errors.New("context stage2 file path is required")
	}
	file, err := os.Open(f.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Response{Chunks: nil, Meta: map[string]any{"source": "file", "matched": 0}}, nil
		}
		return Response{}, fmt.Errorf("open context stage2 file: %w", err)
	}
	defer func() { _ = file.Close() }()

	type row struct {
		RunID     string `json:"run_id"`
		SessionID string `json:"session_id"`
		Content   string `json:"content"`
	}
	items := make([]string, 0, req.MaxItems)
	sc := bufio.NewScanner(file)
	for sc.Scan() {
		if err := ctx.Err(); err != nil {
			return Response{}, err
		}
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var entry row
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		if entry.Content == "" {
			continue
		}
		if req.SessionID != "" && entry.SessionID != req.SessionID {
			continue
		}
		if req.RunID != "" && entry.RunID != "" && entry.RunID != req.RunID && req.SessionID == "" {
			continue
		}
		items = append(items, entry.Content)
	}
	if err := sc.Err(); err != nil {
		return Response{}, fmt.Errorf("scan context stage2 file: %w", err)
	}
	if req.MaxItems > 0 && len(items) > req.MaxItems {
		items = items[len(items)-req.MaxItems:]
	}
	return Response{
		Chunks: items,
		Meta: map[string]any{
			"source":  "file",
			"matched": len(items),
		},
	}, nil
}
