package assembler

import (
	"context"
	"fmt"
	"strings"

	"github.com/FelixSeptem/baymax/core/types"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

type ca3Compactor interface {
	mode() string
	compact(ctx context.Context, req ca3CompactionRequest) (ca3CompactionResult, error)
}

type ca3CompactionRequest struct {
	AssembleReq types.ContextAssembleRequest
	ModelReq    types.ModelRequest
	Config      runtimeconfig.ContextAssemblerCA3Config
}

type ca3CompactionResult struct {
	Messages         []types.Message
	CompressionRatio float64
	Fallback         bool
}

type truncateCompactor struct{}

func (c *truncateCompactor) mode() string {
	return "truncate"
}

func (c *truncateCompactor) compact(_ context.Context, req ca3CompactionRequest) (ca3CompactionResult, error) {
	before := 0
	after := 0
	messages := make([]types.Message, 0, len(req.ModelReq.Messages))
	maxRunes := req.Config.Squash.MaxContentRunes
	if maxRunes <= 0 {
		maxRunes = 320
	}
	for _, msg := range req.ModelReq.Messages {
		before += len([]rune(msg.Content))
		if strings.EqualFold(strings.TrimSpace(msg.Role), "system") || isProtectedMessage(msg.Content, req.Config.Protection) {
			after += len([]rune(msg.Content))
			messages = append(messages, msg)
			continue
		}
		content := msg.Content
		if len([]rune(content)) > maxRunes {
			content = string([]rune(content)[:maxRunes]) + " ...[squashed]"
		}
		after += len([]rune(content))
		msg.Content = content
		messages = append(messages, msg)
	}
	compression := 0.0
	if before > 0 {
		compression = float64(before-after) / float64(before)
		if compression < 0 {
			compression = 0
		}
	}
	return ca3CompactionResult{
		Messages:         messages,
		CompressionRatio: compression,
	}, nil
}

type semanticCompactor struct {
	client types.ModelClient
}

func (c *semanticCompactor) mode() string {
	return "semantic"
}

func (c *semanticCompactor) compact(ctx context.Context, req ca3CompactionRequest) (ca3CompactionResult, error) {
	if c.client == nil {
		return ca3CompactionResult{}, fmt.Errorf("semantic compactor model client not available")
	}
	before := 0
	after := 0
	out := make([]types.Message, 0, len(req.ModelReq.Messages))
	maxRunes := req.Config.Squash.MaxContentRunes
	if maxRunes <= 0 {
		maxRunes = 320
	}
	for _, msg := range req.ModelReq.Messages {
		before += len([]rune(msg.Content))
		if strings.EqualFold(strings.TrimSpace(msg.Role), "system") || isProtectedMessage(msg.Content, req.Config.Protection) {
			after += len([]rune(msg.Content))
			out = append(out, msg)
			continue
		}
		if len([]rune(msg.Content)) <= maxRunes {
			after += len([]rune(msg.Content))
			out = append(out, msg)
			continue
		}
		prompt := buildSemanticCompactionPrompt(msg.Content, maxRunes)
		resp, err := c.client.Generate(ctx, types.ModelRequest{
			Model: req.ModelReq.Model,
			Input: prompt,
		})
		if err != nil {
			return ca3CompactionResult{}, fmt.Errorf("semantic compaction generate failed: %w", err)
		}
		content := strings.TrimSpace(resp.FinalAnswer)
		if content == "" {
			return ca3CompactionResult{}, fmt.Errorf("semantic compaction returned empty content")
		}
		if len([]rune(content)) > maxRunes {
			content = string([]rune(content)[:maxRunes]) + " ...[squashed]"
		}
		msg.Content = content
		after += len([]rune(content))
		out = append(out, msg)
	}
	compression := 0.0
	if before > 0 {
		compression = float64(before-after) / float64(before)
		if compression < 0 {
			compression = 0
		}
	}
	return ca3CompactionResult{
		Messages:         out,
		CompressionRatio: compression,
	}, nil
}

func buildSemanticCompactionPrompt(content string, maxRunes int) string {
	return strings.TrimSpace(fmt.Sprintf(
		"Compress the text for context-window efficiency while preserving intent, constraints, decisions, todo, and risk details. "+
			"Return plain text only in Chinese if source is Chinese, otherwise keep source language. "+
			"Keep output under %d characters.\n\nSource:\n%s",
		maxRunes,
		content,
	))
}
