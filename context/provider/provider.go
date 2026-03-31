package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	memoryspi "github.com/FelixSeptem/baymax/memory"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

var ErrProviderNotReady = errors.New("context stage2 provider not ready")

type ErrorLayer string

const (
	ErrorLayerTransport ErrorLayer = "transport"
	ErrorLayerProtocol  ErrorLayer = "protocol"
	ErrorLayerSemantic  ErrorLayer = "semantic"
)

type FetchError struct {
	Layer   ErrorLayer
	Code    string
	Message string
	Cause   error
}

func (e *FetchError) Error() string {
	if e == nil {
		return ""
	}
	msg := strings.TrimSpace(e.Message)
	if msg == "" && e.Cause != nil {
		msg = e.Cause.Error()
	}
	if msg == "" {
		msg = "context stage2 fetch failed"
	}
	return msg
}

func (e *FetchError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

type Request struct {
	RunID     string
	SessionID string
	Input     string
	MaxItems  int
	Hints     CapabilityHints
}

type CapabilityHints struct {
	Capabilities []string
}

type Response struct {
	Chunks []string
	Meta   map[string]any
}

type Provider interface {
	Name() string
	Fetch(ctx context.Context, req Request) (Response, error)
}

type Config struct {
	Name     string
	FilePath string
	External runtimeconfig.ContextAssemblerCA2ExternalConfig
	Memory   runtimeconfig.RuntimeMemoryConfig
}

func New(name, filePath string) (Provider, error) {
	return NewWithConfig(Config{Name: name, FilePath: filePath})
}

func NewWithConfig(cfg Config) (Provider, error) {
	providerName := strings.ToLower(strings.TrimSpace(cfg.Name))
	switch providerName {
	case "", runtimeconfig.ContextStage2ProviderFile:
		return &fileProvider{path: strings.TrimSpace(cfg.FilePath)}, nil
	case runtimeconfig.ContextStage2ProviderHTTP, runtimeconfig.ContextStage2ProviderRAG, runtimeconfig.ContextStage2ProviderDB, runtimeconfig.ContextStage2ProviderElasticsearch:
		return &httpProvider{name: providerName, cfg: cfg.External, client: &http.Client{}}, nil
	case runtimeconfig.ContextStage2ProviderMemory:
		return newMemoryProvider(cfg)
	default:
		return nil, fmt.Errorf("unsupported context stage2 provider %q", cfg.Name)
	}
}

type httpMemoryEngine struct {
	p *httpProvider
}

func (h *httpMemoryEngine) Query(req memoryspi.QueryRequest) (memoryspi.QueryResponse, error) {
	if h == nil || h.p == nil {
		return memoryspi.QueryResponse{}, &memoryspi.Error{
			Operation: memoryspi.OperationQuery,
			Code:      memoryspi.ReasonCodeProviderUnavailable,
			Layer:     memoryspi.LayerRuntime,
			Message:   "external memory engine is not initialized",
		}
	}
	resp, err := h.p.Fetch(context.Background(), Request{
		RunID:     req.RunID,
		SessionID: req.SessionID,
		Input:     req.Query,
		MaxItems:  req.MaxItems,
	})
	if err != nil {
		var fetchErr *FetchError
		if errors.As(err, &fetchErr) {
			return memoryspi.QueryResponse{}, &memoryspi.Error{
				Operation: memoryspi.OperationQuery,
				Code:      memoryspi.ReasonCodeProviderUnavailable,
				Layer:     memoryspi.LayerTransport,
				Message:   strings.TrimSpace(fetchErr.Message),
				Cause:     err,
			}
		}
		return memoryspi.QueryResponse{}, &memoryspi.Error{
			Operation: memoryspi.OperationQuery,
			Code:      memoryspi.ReasonCodeProviderUnavailable,
			Layer:     memoryspi.LayerRuntime,
			Message:   "external memory query failed",
			Cause:     err,
		}
	}
	records := make([]memoryspi.Record, 0, len(resp.Chunks))
	for i, chunk := range resp.Chunks {
		records = append(records, memoryspi.Record{
			ID:        fmt.Sprintf("chunk-%d", i),
			Namespace: strings.TrimSpace(req.Namespace),
			SessionID: strings.TrimSpace(req.SessionID),
			RunID:     strings.TrimSpace(req.RunID),
			Content:   strings.TrimSpace(chunk),
		})
	}
	return memoryspi.QueryResponse{
		OperationID: req.OperationID,
		Namespace:   strings.TrimSpace(req.Namespace),
		Records:     records,
		Total:       len(records),
		ReasonCode:  memoryspi.ReasonCodeOK,
		Metadata:    cloneAnyMap(resp.Meta),
	}, nil
}

func (h *httpMemoryEngine) Upsert(req memoryspi.UpsertRequest) (memoryspi.UpsertResponse, error) {
	return memoryspi.UpsertResponse{}, &memoryspi.Error{
		Operation: memoryspi.OperationUpsert,
		Code:      memoryspi.ReasonCodeUnsupportedOperation,
		Layer:     memoryspi.LayerSemantic,
		Message:   "external stage2 adapter does not support upsert operation",
	}
}

func (h *httpMemoryEngine) Delete(req memoryspi.DeleteRequest) (memoryspi.DeleteResponse, error) {
	return memoryspi.DeleteResponse{}, &memoryspi.Error{
		Operation: memoryspi.OperationDelete,
		Code:      memoryspi.ReasonCodeUnsupportedOperation,
		Layer:     memoryspi.LayerSemantic,
		Message:   "external stage2 adapter does not support delete operation",
	}
}

type fileProvider struct {
	path string
}

func (f *fileProvider) Name() string {
	return runtimeconfig.ContextStage2ProviderFile
}

func (f *fileProvider) Fetch(ctx context.Context, req Request) (Response, error) {
	if strings.TrimSpace(f.path) == "" {
		return Response{}, errors.New("context stage2 file path is required")
	}
	file, err := os.Open(f.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Response{Chunks: nil, Meta: map[string]any{"source": "file", "matched": 0, "reason": "not_found"}}, nil
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
			"source":                       "file",
			"matched":                      len(items),
			"reason":                       "ok",
			"reason_code":                  "ok",
			"error_layer":                  "",
			"profile":                      "file",
			"template_profile":             "file",
			"template_resolution_source":   runtimeconfig.Stage2TemplateResolutionExplicitOnly,
			"hint_applied":                 false,
			"hint_mismatch_reason":         "",
			"capability_hints_forwarded":   []string{},
			"capability_hints_unsupported": []string{},
		},
	}, nil
}

type httpProvider struct {
	name   string
	cfg    runtimeconfig.ContextAssemblerCA2ExternalConfig
	client *http.Client
}

func (p *httpProvider) Name() string {
	return p.name
}

func (p *httpProvider) Fetch(ctx context.Context, req Request) (Response, error) {
	if strings.TrimSpace(p.cfg.Endpoint) == "" {
		return Response{}, &FetchError{
			Layer:   ErrorLayerProtocol,
			Code:    "missing_endpoint",
			Message: "context stage2 external endpoint is required",
		}
	}
	payload := p.buildRequestPayload(req)
	raw, err := json.Marshal(payload)
	if err != nil {
		return Response{}, &FetchError{
			Layer:   ErrorLayerProtocol,
			Code:    "request_encode_failed",
			Message: "marshal context stage2 external request failed",
			Cause:   err,
		}
	}

	httpReq, err := http.NewRequestWithContext(ctx, httpMethodOrDefault(p.cfg.Method), p.cfg.Endpoint, bytes.NewReader(raw))
	if err != nil {
		return Response{}, &FetchError{
			Layer:   ErrorLayerProtocol,
			Code:    "request_build_failed",
			Message: "build context stage2 external request failed",
			Cause:   err,
		}
	}
	httpReq.Header.Set("Content-Type", "application/json")
	for k, v := range p.cfg.Headers {
		if strings.TrimSpace(k) == "" {
			continue
		}
		httpReq.Header.Set(k, v)
	}
	if strings.TrimSpace(p.cfg.Auth.BearerToken) != "" {
		headerName := strings.TrimSpace(p.cfg.Auth.HeaderName)
		if headerName == "" {
			headerName = "Authorization"
		}
		httpReq.Header.Set(headerName, "Bearer "+strings.TrimSpace(p.cfg.Auth.BearerToken))
	}

	httpResp, err := p.client.Do(httpReq)
	if err != nil {
		return Response{}, classifyTransportError(err)
	}
	defer func() { _ = httpResp.Body.Close() }()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return Response{}, &FetchError{
			Layer:   ErrorLayerProtocol,
			Code:    "response_read_failed",
			Message: "read context stage2 external response failed",
			Cause:   err,
		}
	}
	var decoded map[string]any
	if err := json.Unmarshal(body, &decoded); err != nil {
		return Response{}, &FetchError{
			Layer:   ErrorLayerProtocol,
			Code:    "response_decode_failed",
			Message: "decode context stage2 external response failed",
			Cause:   err,
		}
	}
	if httpResp.StatusCode >= 400 {
		msg := asString(getByPath(decoded, p.cfg.Mapping.Response.ErrorMessageField))
		if msg == "" {
			msg = httpResp.Status
		}
		return Response{}, &FetchError{
			Layer:   ErrorLayerProtocol,
			Code:    "http_status",
			Message: fmt.Sprintf("context stage2 external http status=%d: %s", httpResp.StatusCode, msg),
		}
	}
	if errNode := getByPath(decoded, p.cfg.Mapping.Response.ErrorField); errNode != nil {
		msg := asString(getByPath(decoded, p.cfg.Mapping.Response.ErrorMessageField))
		if msg == "" {
			msg = asString(errNode)
		}
		if msg == "" {
			msg = "unknown external error"
		}
		return Response{}, &FetchError{
			Layer:   ErrorLayerSemantic,
			Code:    "upstream_error",
			Message: msg,
		}
	}
	chunks := asStringSlice(getByPath(decoded, p.cfg.Mapping.Response.ChunksField))
	source := asString(getByPath(decoded, p.cfg.Mapping.Response.SourceField))
	if source == "" {
		source = p.name
	}
	reason := asString(getByPath(decoded, p.cfg.Mapping.Response.ReasonField))
	if reason == "" {
		reason = "ok"
	}
	profile := strings.TrimSpace(p.cfg.Profile)
	if profile == "" {
		profile = runtimeconfig.ContextStage2ExternalProfileHTTPGeneric
	}
	templateResolutionSource := strings.TrimSpace(p.cfg.TemplateResolutionSource)
	if templateResolutionSource == "" {
		templateResolutionSource = runtimeconfig.Stage2TemplateResolutionProfileDefaultsOnly
	}
	hintApplied, hintMismatchReason, unsupportedHints := evaluateHintOutcome(p.name, req.Hints.Capabilities)
	return Response{Chunks: chunks, Meta: map[string]any{
		"source":                       source,
		"matched":                      len(chunks),
		"reason":                       reason,
		"reason_code":                  "ok",
		"error_layer":                  "",
		"profile":                      profile,
		"template_profile":             profile,
		"template_resolution_source":   templateResolutionSource,
		"hint_applied":                 hintApplied,
		"hint_mismatch_reason":         hintMismatchReason,
		"capability_hints_forwarded":   normalizeHintList(req.Hints.Capabilities),
		"capability_hints_unsupported": unsupportedHints,
	}}, nil
}

func (p *httpProvider) buildRequestPayload(req Request) map[string]any {
	mapping := p.cfg.Mapping.Request
	hints := normalizeHintList(req.Hints.Capabilities)
	if strings.EqualFold(strings.TrimSpace(mapping.Mode), "jsonrpc2") {
		params := map[string]any{}
		setByPath(params, nonEmpty(mapping.QueryField, "query"), req.Input)
		setByPath(params, nonEmpty(mapping.SessionIDField, "session_id"), req.SessionID)
		setByPath(params, nonEmpty(mapping.RunIDField, "run_id"), req.RunID)
		setByPath(params, nonEmpty(mapping.MaxItemsField, "max_items"), req.MaxItems)
		if len(hints) > 0 {
			params["capability_hints"] = hints
		}
		return map[string]any{
			"jsonrpc": nonEmpty(mapping.JSONRPCVersion, "2.0"),
			"id":      nonEmpty(req.RunID, strconv.FormatInt(time.Now().UnixNano(), 10)),
			"method":  mapping.MethodName,
			"params":  params,
		}
	}
	payload := map[string]any{}
	setByPath(payload, nonEmpty(mapping.QueryField, "query"), req.Input)
	setByPath(payload, nonEmpty(mapping.SessionIDField, "session_id"), req.SessionID)
	setByPath(payload, nonEmpty(mapping.RunIDField, "run_id"), req.RunID)
	setByPath(payload, nonEmpty(mapping.MaxItemsField, "max_items"), req.MaxItems)
	if len(hints) > 0 {
		payload["capability_hints"] = hints
	}
	return payload
}

func httpMethodOrDefault(v string) string {
	m := strings.ToUpper(strings.TrimSpace(v))
	if m == "" {
		return "POST"
	}
	return m
}

func getByPath(payload map[string]any, path string) any {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	parts := strings.Split(path, ".")
	var cur any = payload
	for _, part := range parts {
		m, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur, ok = m[part]
		if !ok {
			return nil
		}
	}
	return cur
}

func setByPath(payload map[string]any, path string, value any) {
	path = strings.TrimSpace(path)
	if path == "" {
		return
	}
	parts := strings.Split(path, ".")
	cur := payload
	for i := 0; i < len(parts)-1; i++ {
		part := parts[i]
		next, ok := cur[part].(map[string]any)
		if !ok {
			next = map[string]any{}
			cur[part] = next
		}
		cur = next
	}
	cur[parts[len(parts)-1]] = value
}

func asString(v any) string {
	s, _ := v.(string)
	return strings.TrimSpace(s)
}

func asStringSlice(v any) []string {
	if v == nil {
		return nil
	}
	list, ok := v.([]any)
	if !ok {
		if str := asString(v); str != "" {
			return []string{str}
		}
		return nil
	}
	out := make([]string, 0, len(list))
	for _, item := range list {
		str := asString(item)
		if str == "" {
			continue
		}
		out = append(out, str)
	}
	return out
}

func nonEmpty(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return strings.TrimSpace(v)
}

func classifyTransportError(err error) error {
	code := "request_failed"
	msg := "call context stage2 external provider failed"
	if errors.Is(err, context.DeadlineExceeded) {
		code = "timeout"
		msg = "context stage2 external provider timeout"
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		code = "timeout"
		msg = "context stage2 external provider timeout"
	}
	return &FetchError{
		Layer:   ErrorLayerTransport,
		Code:    code,
		Message: msg,
		Cause:   err,
	}
}

func normalizeHintList(in []string) []string {
	if len(in) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(in))
	seen := map[string]struct{}{}
	for _, item := range in {
		value := strings.ToLower(strings.TrimSpace(item))
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func evaluateHintOutcome(providerName string, requested []string) (bool, string, []string) {
	normalized := normalizeHintList(requested)
	if len(normalized) == 0 {
		return false, "", []string{}
	}
	supported := supportedHintCapabilities(providerName)
	unsupported := make([]string, 0, len(normalized))
	for _, capability := range normalized {
		if _, ok := supported[capability]; ok {
			continue
		}
		unsupported = append(unsupported, capability)
	}
	if len(unsupported) > 0 {
		return false, "hint.unsupported", unsupported
	}
	return true, "", []string{}
}

func supportedHintCapabilities(providerName string) map[string]struct{} {
	switch strings.ToLower(strings.TrimSpace(providerName)) {
	case runtimeconfig.ContextStage2ProviderElasticsearch:
		return map[string]struct{}{
			"dsl_query":        {},
			"metadata_filter":  {},
			"vector_filter":    {},
			"rerank_metadata":  {},
			"hybrid_candidate": {},
		}
	case runtimeconfig.ContextStage2ProviderRAG, runtimeconfig.ContextStage2ProviderDB:
		return map[string]struct{}{
			"metadata_filter": {},
			"rerank_metadata": {},
		}
	default:
		return map[string]struct{}{
			"metadata_filter": {},
		}
	}
}

func defaultHTTPClient() *http.Client {
	return &http.Client{}
}
