package memory

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

const (
	ContractVersionMemoryV1 = "memory.v1"
)

const (
	ModeExternalSPI       = "external_spi"
	ModeBuiltinFilesystem = "builtin_filesystem"
)

const (
	FallbackPolicyFailFast             = "fail_fast"
	FallbackPolicyDegradeToBuiltin     = "degrade_to_builtin"
	FallbackPolicyDegradeWithoutMemory = "degrade_without_memory"
)

const (
	ScopeSession = "session"
	ScopeProject = "project"
	ScopeGlobal  = "global"
)

const (
	WriteModeAutomatic = "automatic"
	WriteModeAgentic   = "agentic"
)

const (
	TruncatePolicyScoreThenRecency = "score_then_recency"
	TruncatePolicyRecencyThenID    = "recency_then_id"
)

const (
	IndexUpdatePolicyIncremental               = "incremental"
	IndexUpdatePolicyFullRebuildOnProfileDrift = "full_rebuild_on_profile_drift"
)

const (
	DriftRecoveryPolicyIncrementalThenFull = "incremental_then_full"
	DriftRecoveryPolicyFullRebuild         = "full_rebuild"
)

const (
	LifecycleActionNone                     = ""
	LifecycleActionRetention                = "retention_applied"
	LifecycleActionTTL                      = "ttl_expired"
	LifecycleActionForget                   = "forget_applied"
	LifecycleActionRecoveryConsistencyDrift = "recovery_consistency_drift"
)

type ExternalConfig struct {
	Provider        string `json:"provider"`
	Profile         string `json:"profile"`
	ContractVersion string `json:"contract_version"`
}

type BuiltinConfig struct {
	RootDir    string                     `json:"root_dir"`
	Compaction FilesystemCompactionConfig `json:"compaction"`
}

type FallbackConfig struct {
	Policy string `json:"policy"`
}

type Config struct {
	Mode            string                `json:"mode"`
	External        ExternalConfig        `json:"external"`
	Builtin         BuiltinConfig         `json:"builtin"`
	Fallback        FallbackConfig        `json:"fallback"`
	Scope           ScopeConfig           `json:"scope"`
	WriteMode       WriteModeConfig       `json:"write_mode"`
	InjectionBudget InjectionBudgetConfig `json:"injection_budget"`
	Lifecycle       LifecycleConfig       `json:"lifecycle"`
	Search          SearchConfig          `json:"search"`
}

type ScopeConfig struct {
	Default         string   `json:"default"`
	Allowed         []string `json:"allowed"`
	AllowOverride   bool     `json:"allow_override"`
	GlobalNamespace string   `json:"global_namespace"`
}

type WriteModeConfig struct {
	Mode              string        `json:"mode"`
	AutomaticWindow   time.Duration `json:"automatic_window"`
	AgenticWindow     time.Duration `json:"agentic_window"`
	IdempotencyWindow time.Duration `json:"idempotency_window"`
}

type InjectionBudgetConfig struct {
	MaxRecords     int    `json:"max_records"`
	MaxBytes       int    `json:"max_bytes"`
	TruncatePolicy string `json:"truncate_policy"`
}

type LifecycleConfig struct {
	RetentionDays    int           `json:"retention_days"`
	TTLEnabled       bool          `json:"ttl_enabled"`
	TTL              time.Duration `json:"ttl"`
	ForgetScopeAllow []string      `json:"forget_scope_allow"`
}

type SearchConfig struct {
	Hybrid              SearchHybridConfig        `json:"hybrid"`
	Rerank              SearchRerankConfig        `json:"rerank"`
	TemporalDecay       SearchTemporalDecayConfig `json:"temporal_decay"`
	IndexUpdatePolicy   string                    `json:"index_update_policy"`
	DriftRecoveryPolicy string                    `json:"drift_recovery_policy"`
}

type SearchHybridConfig struct {
	Enabled       bool    `json:"enabled"`
	KeywordWeight float64 `json:"keyword_weight"`
	VectorWeight  float64 `json:"vector_weight"`
}

type SearchRerankConfig struct {
	Enabled       bool `json:"enabled"`
	MaxCandidates int  `json:"max_candidates"`
}

type SearchTemporalDecayConfig struct {
	Enabled      bool          `json:"enabled"`
	HalfLife     time.Duration `json:"half_life"`
	MaxBoostRate float64       `json:"max_boost_rate"`
}

type ExternalEngineFactory func(cfg ExternalConfig) (Engine, error)

type Facade struct {
	mode            string
	provider        string
	profile         string
	contractVersion string
	fallbackPolicy  string
	scopeCfg        ScopeConfig
	writeModeCfg    WriteModeConfig
	budgetCfg       InjectionBudgetConfig
	lifecycleCfg    LifecycleConfig
	searchCfg       SearchConfig

	active  Engine
	builtin Engine

	mu          sync.Mutex
	agenticSeen map[string]time.Time
}

func NewFacade(cfg Config, externalFactory ExternalEngineFactory) (*Facade, error) {
	normalized := normalizeConfig(cfg)
	if err := validateConfig(normalized); err != nil {
		return nil, err
	}
	out := &Facade{
		mode:            normalized.Mode,
		profile:         normalized.External.Profile,
		contractVersion: normalized.External.ContractVersion,
		fallbackPolicy:  normalized.Fallback.Policy,
		scopeCfg:        normalized.Scope,
		writeModeCfg:    normalized.WriteMode,
		budgetCfg:       normalized.InjectionBudget,
		lifecycleCfg:    normalized.Lifecycle,
		searchCfg:       normalized.Search,
		agenticSeen:     map[string]time.Time{},
	}
	switch normalized.Mode {
	case ModeBuiltinFilesystem:
		builtin, err := newBuiltinEngine(normalized)
		if err != nil {
			return nil, err
		}
		out.provider = ModeBuiltinFilesystem
		out.active = builtin
		out.builtin = builtin
	case ModeExternalSPI:
		out.provider = normalized.External.Provider
		if _, err := ResolveProfile(normalized.External.Profile); err != nil {
			return nil, err
		}
		if externalFactory == nil {
			return nil, &Error{
				Operation: OperationQuery,
				Code:      ReasonCodeProviderUnavailable,
				Layer:     LayerRuntime,
				Message:   "external memory engine factory is required in external_spi mode",
			}
		}
		external, err := externalFactory(normalized.External)
		if err != nil {
			return nil, normalizeError(OperationQuery, err)
		}
		out.active = external
		if normalized.Fallback.Policy == FallbackPolicyDegradeToBuiltin {
			builtin, err := newBuiltinEngine(normalized)
			if err != nil {
				return nil, &Error{
					Operation: OperationQuery,
					Code:      ReasonCodeFallbackTargetMissing,
					Layer:     LayerRuntime,
					Message:   "fallback target builtin filesystem is unavailable",
					Cause:     err,
				}
			}
			out.builtin = builtin
		}
	default:
		return nil, &Error{
			Operation: OperationQuery,
			Code:      ReasonCodeInvalidRequest,
			Layer:     LayerRuntime,
			Message:   fmt.Sprintf("unsupported memory mode %q", normalized.Mode),
		}
	}
	return out, nil
}

func (f *Facade) Query(req QueryRequest) (QueryResponse, error) {
	if f == nil || f.active == nil {
		return QueryResponse{}, &Error{
			Operation: OperationQuery,
			Code:      ReasonCodeProviderUnavailable,
			Layer:     LayerRuntime,
			Message:   "memory facade is not initialized",
		}
	}
	return f.queryWithGovernance(req)
}

func (f *Facade) Upsert(req UpsertRequest) (UpsertResponse, error) {
	if f == nil || f.active == nil {
		return UpsertResponse{}, &Error{
			Operation: OperationUpsert,
			Code:      ReasonCodeProviderUnavailable,
			Layer:     LayerRuntime,
			Message:   "memory facade is not initialized",
		}
	}
	if f.writeModeCfg.Mode == WriteModeAgentic && f.shouldSkipAgenticUpsert(req) {
		resp := UpsertResponse{
			OperationID: req.OperationID,
			Namespace:   strings.TrimSpace(req.Namespace),
			Upserted:    0,
			ReasonCode:  ReasonCodeOK,
		}
		f.decorateUpsertResponse(&resp, false, "", f.mode, f.provider)
		return resp, nil
	}
	resp, err := f.active.Upsert(req)
	if err == nil {
		f.decorateUpsertResponse(&resp, false, "", f.mode, f.provider)
		if resp.MemoryLifecycleAction == "" {
			resp.MemoryLifecycleAction = LifecycleActionNone
		}
		f.recordAgenticUpsert(req)
		return resp, nil
	}
	fallbackResp, fallbackErr := f.upsertWithFallback(req, err)
	if fallbackErr == nil {
		if fallbackResp.MemoryLifecycleAction == "" {
			fallbackResp.MemoryLifecycleAction = LifecycleActionNone
		}
		f.recordAgenticUpsert(req)
	}
	return fallbackResp, fallbackErr
}

func (f *Facade) Delete(req DeleteRequest) (DeleteResponse, error) {
	if f == nil || f.active == nil {
		return DeleteResponse{}, &Error{
			Operation: OperationDelete,
			Code:      ReasonCodeProviderUnavailable,
			Layer:     LayerRuntime,
			Message:   "memory facade is not initialized",
		}
	}
	forgetScope := strings.ToLower(strings.TrimSpace(req.Scope))
	if forgetScope != "" && !slices.Contains(f.lifecycleCfg.ForgetScopeAllow, forgetScope) {
		return DeleteResponse{}, &Error{
			Operation: OperationDelete,
			Code:      ReasonCodeInvalidRequest,
			Layer:     LayerRuntime,
			Message:   fmt.Sprintf("forget scope %q is not allowed by lifecycle policy", forgetScope),
		}
	}
	resp, err := f.active.Delete(req)
	if err == nil {
		f.decorateDeleteResponse(&resp, false, "", f.mode, f.provider)
		if forgetScope != "" && resp.MemoryLifecycleAction == "" {
			resp.MemoryLifecycleAction = LifecycleActionForget
		}
		return resp, nil
	}
	fallbackResp, fallbackErr := f.deleteWithFallback(req, err)
	if fallbackErr == nil && forgetScope != "" && fallbackResp.MemoryLifecycleAction == "" {
		fallbackResp.MemoryLifecycleAction = LifecycleActionForget
	}
	return fallbackResp, fallbackErr
}

func (f *Facade) Close() error {
	if f == nil {
		return nil
	}
	var closeErr error
	if err := closeEngine(f.active); err != nil {
		closeErr = err
	}
	if f.builtin != nil && f.builtin != f.active {
		if err := closeEngine(f.builtin); err != nil && closeErr == nil {
			closeErr = err
		}
	}
	return closeErr
}

func (f *Facade) queryWithGovernance(req QueryRequest) (QueryResponse, error) {
	scopes, err := f.orderedScopesForQuery(req)
	if err != nil {
		return QueryResponse{}, err
	}
	if len(scopes) == 0 {
		return QueryResponse{}, &Error{
			Operation: OperationQuery,
			Code:      ReasonCodeInvalidRequest,
			Layer:     LayerRuntime,
			Message:   "memory scope resolution produced no candidates",
		}
	}

	selectedScope := scopes[0]
	var selected QueryResponse
	found := false
	for _, scope := range scopes {
		if scope == ScopeSession && strings.TrimSpace(req.SessionID) == "" {
			continue
		}
		scopeReq := f.queryRequestForScope(req, scope)
		resp, runErr := f.executeQuery(scopeReq)
		if runErr != nil {
			return QueryResponse{}, runErr
		}
		selected = resp
		selectedScope = scope
		if resp.Total > 0 {
			found = true
			break
		}
	}
	if !found && strings.TrimSpace(selected.OperationID) == "" {
		resp, runErr := f.executeQuery(f.queryRequestForScope(req, selectedScope))
		if runErr != nil {
			return QueryResponse{}, runErr
		}
		selected = resp
	}

	records, rerankStats := f.applySearchGovernance(selected.Records, strings.TrimSpace(req.Query))
	hits := len(records)
	records, budgetUsed := f.applyInjectionBudget(records, req.MaxItems)
	selected.Records = records
	selected.Total = len(records)
	selected.MemoryScopeSelected = selectedScope
	selected.MemoryBudgetUsed = budgetUsed
	selected.MemoryHits = hits
	selected.MemoryRerankStats = rerankStats
	if selected.MemoryLifecycleAction == "" {
		selected.MemoryLifecycleAction = LifecycleActionNone
	}
	if selected.Metadata == nil {
		selected.Metadata = map[string]any{}
	}
	selected.Metadata["memory_scope_selected"] = selected.MemoryScopeSelected
	selected.Metadata["memory_budget_used"] = selected.MemoryBudgetUsed
	selected.Metadata["memory_hits"] = selected.MemoryHits
	if len(selected.MemoryRerankStats) > 0 {
		stats := make(map[string]any, len(selected.MemoryRerankStats))
		for key, value := range selected.MemoryRerankStats {
			stats[key] = value
		}
		selected.Metadata["memory_rerank_stats"] = stats
	}
	if selected.MemoryLifecycleAction != "" {
		selected.Metadata["memory_lifecycle_action"] = selected.MemoryLifecycleAction
	}
	return selected, nil
}

func (f *Facade) executeQuery(req QueryRequest) (QueryResponse, error) {
	resp, err := f.active.Query(req)
	if err == nil {
		f.decorateQueryResponse(&resp, false, "", f.mode, f.provider)
		return resp, nil
	}
	return f.queryWithFallback(req, err)
}

func (f *Facade) orderedScopesForQuery(req QueryRequest) ([]string, error) {
	allowed := normalizeScopeList(f.scopeCfg.Allowed)
	if len(allowed) == 0 {
		return nil, &Error{
			Operation: OperationQuery,
			Code:      ReasonCodeInvalidRequest,
			Layer:     LayerRuntime,
			Message:   "runtime.memory.scope.allowed must not be empty",
		}
	}
	override := strings.ToLower(strings.TrimSpace(req.Scope))
	if override != "" {
		if !f.scopeCfg.AllowOverride {
			return nil, &Error{
				Operation: OperationQuery,
				Code:      ReasonCodeInvalidRequest,
				Layer:     LayerRuntime,
				Message:   "memory scope override is disabled by policy",
			}
		}
		if !isScope(override) {
			return nil, &Error{
				Operation: OperationQuery,
				Code:      ReasonCodeInvalidRequest,
				Layer:     LayerRuntime,
				Message:   fmt.Sprintf("memory scope override %q is invalid", override),
			}
		}
		if !slices.Contains(allowed, override) {
			return nil, &Error{
				Operation: OperationQuery,
				Code:      ReasonCodeInvalidRequest,
				Layer:     LayerRuntime,
				Message:   fmt.Sprintf("memory scope override %q is not allowed", override),
			}
		}
		return []string{override}, nil
	}
	out := make([]string, 0, 3)
	for _, scope := range []string{ScopeSession, ScopeProject, ScopeGlobal} {
		if slices.Contains(allowed, scope) {
			out = append(out, scope)
		}
	}
	return out, nil
}

func (f *Facade) queryRequestForScope(req QueryRequest, scope string) QueryRequest {
	out := req
	switch scope {
	case ScopeSession:
		// Keep request as-is: session scope requires matching SessionID.
	case ScopeProject:
		out.SessionID = ""
	case ScopeGlobal:
		out.Namespace = strings.TrimSpace(f.scopeCfg.GlobalNamespace)
		out.SessionID = ""
	}
	out.Scope = scope
	return out
}

type scoredRecord struct {
	Record
	score        float64
	keywordScore float64
	vectorScore  float64
}

func (f *Facade) applySearchGovernance(in []Record, query string) ([]Record, map[string]int) {
	if len(in) == 0 {
		return nil, map[string]int{
			"input_total":    0,
			"reranked_total": 0,
			"output_total":   0,
		}
	}
	scored := make([]scoredRecord, 0, len(in))
	for _, record := range in {
		keyword := keywordScore(record.Content, query)
		vector := vectorScore(record)
		score := keyword
		if f.searchCfg.Hybrid.Enabled {
			score = f.searchCfg.Hybrid.KeywordWeight*keyword + f.searchCfg.Hybrid.VectorWeight*vector
		}
		scored = append(scored, scoredRecord{
			Record:       cloneRecord(record),
			score:        score,
			keywordScore: keyword,
			vectorScore:  vector,
		})
	}

	if f.searchCfg.TemporalDecay.Enabled {
		now := time.Now().UTC()
		halfLife := f.searchCfg.TemporalDecay.HalfLife
		boost := f.searchCfg.TemporalDecay.MaxBoostRate
		for i := range scored {
			if scored[i].UpdatedAt.IsZero() {
				continue
			}
			age := now.Sub(scored[i].UpdatedAt)
			if age <= 0 {
				scored[i].score *= 1 + boost
				continue
			}
			decay := math.Exp(-float64(age) / float64(halfLife))
			scored[i].score *= 1 + boost*decay
		}
	}

	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		if !scored[i].UpdatedAt.Equal(scored[j].UpdatedAt) {
			return scored[i].UpdatedAt.After(scored[j].UpdatedAt)
		}
		return scored[i].ID < scored[j].ID
	})

	rerankedTotal := len(scored)
	if f.searchCfg.Rerank.Enabled && f.searchCfg.Rerank.MaxCandidates > 0 && len(scored) > f.searchCfg.Rerank.MaxCandidates {
		rerankedTotal = f.searchCfg.Rerank.MaxCandidates
		scored = append([]scoredRecord(nil), scored[:f.searchCfg.Rerank.MaxCandidates]...)
	}
	out := make([]Record, 0, len(scored))
	for i := range scored {
		out = append(out, scored[i].Record)
	}
	return out, map[string]int{
		"input_total":    len(in),
		"reranked_total": rerankedTotal,
		"output_total":   len(out),
	}
}

func keywordScore(content, query string) float64 {
	query = strings.ToLower(strings.TrimSpace(query))
	content = strings.ToLower(strings.TrimSpace(content))
	if query == "" || content == "" {
		return 1.0
	}
	tokens := strings.Fields(query)
	if len(tokens) == 0 {
		return 1.0
	}
	hits := 0.0
	for _, token := range tokens {
		if token == "" {
			continue
		}
		if strings.Contains(content, token) {
			hits++
		}
	}
	return hits / float64(len(tokens))
}

func vectorScore(record Record) float64 {
	if record.Metadata == nil {
		return 1.0
	}
	if raw, ok := record.Metadata["vector_score"]; ok {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			return 1.0
		}
		var parsed float64
		if _, err := fmt.Sscanf(raw, "%f", &parsed); err == nil {
			if parsed < 0 {
				return 0
			}
			return parsed
		}
	}
	return 1.0
}

func (f *Facade) applyInjectionBudget(in []Record, requestMaxItems int) ([]Record, int) {
	if len(in) == 0 {
		return nil, 0
	}
	records := append([]Record(nil), in...)
	if f.budgetCfg.TruncatePolicy == TruncatePolicyRecencyThenID {
		sort.SliceStable(records, func(i, j int) bool {
			if !records[i].UpdatedAt.Equal(records[j].UpdatedAt) {
				return records[i].UpdatedAt.After(records[j].UpdatedAt)
			}
			return records[i].ID < records[j].ID
		})
	}
	maxRecords := f.budgetCfg.MaxRecords
	if requestMaxItems > 0 && requestMaxItems < maxRecords {
		maxRecords = requestMaxItems
	}
	maxBytes := f.budgetCfg.MaxBytes
	out := make([]Record, 0, len(records))
	usedBytes := 0
	for _, record := range records {
		if len(out) >= maxRecords {
			break
		}
		recordBytes := estimateRecordBytes(record)
		if usedBytes+recordBytes > maxBytes {
			break
		}
		usedBytes += recordBytes
		out = append(out, record)
	}
	return out, len(out)
}

func estimateRecordBytes(record Record) int {
	total := len(record.ID) + len(record.Namespace) + len(record.SessionID) + len(record.RunID)
	total += len(record.Content)
	for key, value := range record.Metadata {
		total += len(key) + len(value)
	}
	if total <= 0 {
		return utf8.RuneCountInString(record.ID)
	}
	return total
}

func (f *Facade) shouldSkipAgenticUpsert(req UpsertRequest) bool {
	key := upsertIdempotencyKey(req)
	if key == "" {
		return false
	}
	now := time.Now().UTC()
	f.mu.Lock()
	defer f.mu.Unlock()
	f.pruneAgenticSeen(now)
	last, ok := f.agenticSeen[key]
	if !ok {
		return false
	}
	return now.Sub(last) <= f.writeModeCfg.IdempotencyWindow
}

func (f *Facade) recordAgenticUpsert(req UpsertRequest) {
	if f.writeModeCfg.Mode != WriteModeAgentic {
		return
	}
	key := upsertIdempotencyKey(req)
	if key == "" {
		return
	}
	now := time.Now().UTC()
	f.mu.Lock()
	defer f.mu.Unlock()
	f.pruneAgenticSeen(now)
	f.agenticSeen[key] = now
}

func (f *Facade) pruneAgenticSeen(now time.Time) {
	if len(f.agenticSeen) == 0 {
		return
	}
	window := f.writeModeCfg.IdempotencyWindow
	for key, ts := range f.agenticSeen {
		if now.Sub(ts) > window {
			delete(f.agenticSeen, key)
		}
	}
}

func upsertIdempotencyKey(req UpsertRequest) string {
	if op := strings.TrimSpace(req.OperationID); op != "" {
		return op
	}
	ids := make([]string, 0, len(req.Records))
	for _, record := range req.Records {
		id := strings.TrimSpace(record.ID)
		if id == "" {
			continue
		}
		ids = append(ids, id)
	}
	sort.Strings(ids)
	if len(ids) == 0 {
		return strings.TrimSpace(req.Namespace)
	}
	return strings.TrimSpace(req.Namespace) + "|" + strings.Join(ids, ",")
}

func normalizeConfig(cfg Config) Config {
	out := cfg
	out.Mode = strings.ToLower(strings.TrimSpace(out.Mode))
	if out.Mode == "" {
		out.Mode = ModeBuiltinFilesystem
	}
	out.External.Provider = strings.ToLower(strings.TrimSpace(out.External.Provider))
	out.External.Profile = strings.ToLower(strings.TrimSpace(out.External.Profile))
	if out.External.Profile == "" {
		out.External.Profile = ProfileGeneric
	}
	out.External.ContractVersion = strings.ToLower(strings.TrimSpace(out.External.ContractVersion))
	if out.External.ContractVersion == "" {
		out.External.ContractVersion = ContractVersionMemoryV1
	}
	out.Builtin.RootDir = strings.TrimSpace(out.Builtin.RootDir)
	out.Fallback.Policy = strings.ToLower(strings.TrimSpace(out.Fallback.Policy))
	if out.Fallback.Policy == "" {
		out.Fallback.Policy = FallbackPolicyFailFast
	}
	out.Scope.Default = strings.ToLower(strings.TrimSpace(out.Scope.Default))
	if out.Scope.Default == "" {
		out.Scope.Default = ScopeSession
	}
	out.Scope.Allowed = normalizeScopeList(out.Scope.Allowed)
	if len(out.Scope.Allowed) == 0 {
		out.Scope.Allowed = []string{ScopeSession, ScopeProject, ScopeGlobal}
	}
	out.Scope.GlobalNamespace = strings.TrimSpace(out.Scope.GlobalNamespace)
	if out.Scope.GlobalNamespace == "" {
		out.Scope.GlobalNamespace = "global"
	}
	out.WriteMode.Mode = strings.ToLower(strings.TrimSpace(out.WriteMode.Mode))
	if out.WriteMode.Mode == "" {
		out.WriteMode.Mode = WriteModeAutomatic
	}
	if out.WriteMode.AutomaticWindow <= 0 {
		out.WriteMode.AutomaticWindow = 30 * time.Minute
	}
	if out.WriteMode.AgenticWindow <= 0 {
		out.WriteMode.AgenticWindow = 2 * time.Hour
	}
	if out.WriteMode.IdempotencyWindow <= 0 {
		out.WriteMode.IdempotencyWindow = 24 * time.Hour
	}
	if out.InjectionBudget.MaxRecords <= 0 {
		out.InjectionBudget.MaxRecords = 8
	}
	if out.InjectionBudget.MaxBytes <= 0 {
		out.InjectionBudget.MaxBytes = 16 * 1024
	}
	out.InjectionBudget.TruncatePolicy = strings.ToLower(strings.TrimSpace(out.InjectionBudget.TruncatePolicy))
	if out.InjectionBudget.TruncatePolicy == "" {
		out.InjectionBudget.TruncatePolicy = TruncatePolicyScoreThenRecency
	}
	if out.Lifecycle.RetentionDays <= 0 {
		out.Lifecycle.RetentionDays = 30
	}
	if out.Lifecycle.TTL <= 0 {
		out.Lifecycle.TTL = 7 * 24 * time.Hour
	}
	out.Lifecycle.ForgetScopeAllow = normalizeScopeList(out.Lifecycle.ForgetScopeAllow)
	if len(out.Lifecycle.ForgetScopeAllow) == 0 {
		out.Lifecycle.ForgetScopeAllow = []string{ScopeSession, ScopeProject, ScopeGlobal}
	}
	if out.Search.Hybrid.KeywordWeight == 0 && out.Search.Hybrid.VectorWeight == 0 {
		out.Search.Hybrid.KeywordWeight = 0.6
		out.Search.Hybrid.VectorWeight = 0.4
	}
	if out.Search.Rerank.MaxCandidates <= 0 {
		out.Search.Rerank.MaxCandidates = 32
	}
	if out.Search.TemporalDecay.HalfLife <= 0 {
		out.Search.TemporalDecay.HalfLife = 7 * 24 * time.Hour
	}
	if out.Search.TemporalDecay.MaxBoostRate == 0 {
		out.Search.TemporalDecay.MaxBoostRate = 0.2
	}
	out.Search.IndexUpdatePolicy = strings.ToLower(strings.TrimSpace(out.Search.IndexUpdatePolicy))
	if out.Search.IndexUpdatePolicy == "" {
		out.Search.IndexUpdatePolicy = IndexUpdatePolicyIncremental
	}
	out.Search.DriftRecoveryPolicy = strings.ToLower(strings.TrimSpace(out.Search.DriftRecoveryPolicy))
	if out.Search.DriftRecoveryPolicy == "" {
		out.Search.DriftRecoveryPolicy = DriftRecoveryPolicyIncrementalThenFull
	}
	return out
}

func validateConfig(cfg Config) error {
	switch cfg.Mode {
	case ModeBuiltinFilesystem, ModeExternalSPI:
	default:
		return &Error{
			Operation: OperationQuery,
			Code:      ReasonCodeInvalidRequest,
			Layer:     LayerRuntime,
			Message:   fmt.Sprintf("runtime.memory.mode must be one of [%s,%s], got %q", ModeExternalSPI, ModeBuiltinFilesystem, cfg.Mode),
		}
	}
	switch cfg.Fallback.Policy {
	case FallbackPolicyFailFast, FallbackPolicyDegradeToBuiltin, FallbackPolicyDegradeWithoutMemory:
	default:
		return &Error{
			Operation: OperationQuery,
			Code:      ReasonCodeFallbackPolicyConflict,
			Layer:     LayerRuntime,
			Message:   fmt.Sprintf("runtime.memory.fallback.policy is unsupported: %q", cfg.Fallback.Policy),
		}
	}
	if cfg.External.ContractVersion != ContractVersionMemoryV1 {
		return &Error{
			Operation: OperationQuery,
			Code:      ReasonCodeContractVersionMismatch,
			Layer:     LayerSemantic,
			Message:   fmt.Sprintf("memory contract version %q is unsupported", cfg.External.ContractVersion),
		}
	}
	if cfg.Mode == ModeExternalSPI {
		if strings.TrimSpace(cfg.External.Provider) == "" {
			return &Error{
				Operation: OperationQuery,
				Code:      ReasonCodeProviderUnavailable,
				Layer:     LayerRuntime,
				Message:   "runtime.memory.external.provider is required when mode=external_spi",
			}
		}
		if _, err := ResolveProfile(cfg.External.Profile); err != nil {
			return err
		}
	}
	if cfg.Mode == ModeBuiltinFilesystem || cfg.Fallback.Policy == FallbackPolicyDegradeToBuiltin {
		if strings.TrimSpace(cfg.Builtin.RootDir) == "" {
			return &Error{
				Operation: OperationQuery,
				Code:      ReasonCodeInvalidRequest,
				Layer:     LayerRuntime,
				Message:   "runtime.memory.builtin.root_dir is required",
			}
		}
	}
	if err := validateScopeConfig(cfg.Scope); err != nil {
		return err
	}
	if err := validateWriteModeConfig(cfg.WriteMode); err != nil {
		return err
	}
	if err := validateInjectionBudgetConfig(cfg.InjectionBudget); err != nil {
		return err
	}
	if err := validateLifecycleConfig(cfg.Lifecycle); err != nil {
		return err
	}
	return validateSearchConfig(cfg.Search)
}

func normalizeScopeList(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	seen := map[string]struct{}{}
	for _, raw := range in {
		scope := strings.ToLower(strings.TrimSpace(raw))
		if scope == "" {
			continue
		}
		if _, ok := seen[scope]; ok {
			continue
		}
		seen[scope] = struct{}{}
		out = append(out, scope)
	}
	return out
}

func isScope(scope string) bool {
	switch strings.ToLower(strings.TrimSpace(scope)) {
	case ScopeSession, ScopeProject, ScopeGlobal:
		return true
	default:
		return false
	}
}

func validateScopeConfig(cfg ScopeConfig) error {
	if !isScope(cfg.Default) {
		return &Error{
			Operation: OperationQuery,
			Code:      ReasonCodeInvalidRequest,
			Layer:     LayerRuntime,
			Message:   fmt.Sprintf("runtime.memory.scope.default must be one of [%s,%s,%s], got %q", ScopeSession, ScopeProject, ScopeGlobal, cfg.Default),
		}
	}
	if len(cfg.Allowed) == 0 {
		return &Error{
			Operation: OperationQuery,
			Code:      ReasonCodeInvalidRequest,
			Layer:     LayerRuntime,
			Message:   "runtime.memory.scope.allowed must not be empty",
		}
	}
	hasDefault := false
	for i := range cfg.Allowed {
		if !isScope(cfg.Allowed[i]) {
			return &Error{
				Operation: OperationQuery,
				Code:      ReasonCodeInvalidRequest,
				Layer:     LayerRuntime,
				Message:   fmt.Sprintf("runtime.memory.scope.allowed[%d] must be one of [%s,%s,%s], got %q", i, ScopeSession, ScopeProject, ScopeGlobal, cfg.Allowed[i]),
			}
		}
		if cfg.Allowed[i] == cfg.Default {
			hasDefault = true
		}
	}
	if !hasDefault {
		return &Error{
			Operation: OperationQuery,
			Code:      ReasonCodeInvalidRequest,
			Layer:     LayerRuntime,
			Message:   "runtime.memory.scope.default must be included in runtime.memory.scope.allowed",
		}
	}
	if strings.TrimSpace(cfg.GlobalNamespace) == "" {
		return &Error{
			Operation: OperationQuery,
			Code:      ReasonCodeInvalidRequest,
			Layer:     LayerRuntime,
			Message:   "runtime.memory.scope.global_namespace is required",
		}
	}
	return nil
}

func validateWriteModeConfig(cfg WriteModeConfig) error {
	switch cfg.Mode {
	case WriteModeAutomatic, WriteModeAgentic:
	default:
		return &Error{
			Operation: OperationUpsert,
			Code:      ReasonCodeInvalidRequest,
			Layer:     LayerRuntime,
			Message:   fmt.Sprintf("runtime.memory.write_mode must be one of [%s,%s], got %q", WriteModeAutomatic, WriteModeAgentic, cfg.Mode),
		}
	}
	if cfg.AutomaticWindow <= 0 {
		return &Error{
			Operation: OperationUpsert,
			Code:      ReasonCodeInvalidRequest,
			Layer:     LayerRuntime,
			Message:   "runtime.memory.write_mode.automatic_window must be > 0",
		}
	}
	if cfg.AgenticWindow <= 0 {
		return &Error{
			Operation: OperationUpsert,
			Code:      ReasonCodeInvalidRequest,
			Layer:     LayerRuntime,
			Message:   "runtime.memory.write_mode.agentic_window must be > 0",
		}
	}
	if cfg.IdempotencyWindow <= 0 {
		return &Error{
			Operation: OperationUpsert,
			Code:      ReasonCodeInvalidRequest,
			Layer:     LayerRuntime,
			Message:   "runtime.memory.write_mode.idempotency_window must be > 0",
		}
	}
	return nil
}

func validateInjectionBudgetConfig(cfg InjectionBudgetConfig) error {
	if cfg.MaxRecords <= 0 {
		return &Error{
			Operation: OperationQuery,
			Code:      ReasonCodeInvalidRequest,
			Layer:     LayerRuntime,
			Message:   "runtime.memory.injection_budget.max_records must be > 0",
		}
	}
	if cfg.MaxBytes <= 0 {
		return &Error{
			Operation: OperationQuery,
			Code:      ReasonCodeInvalidRequest,
			Layer:     LayerRuntime,
			Message:   "runtime.memory.injection_budget.max_bytes must be > 0",
		}
	}
	switch cfg.TruncatePolicy {
	case TruncatePolicyScoreThenRecency, TruncatePolicyRecencyThenID:
	default:
		return &Error{
			Operation: OperationQuery,
			Code:      ReasonCodeInvalidRequest,
			Layer:     LayerRuntime,
			Message:   fmt.Sprintf("runtime.memory.injection_budget.truncate_policy must be one of [%s,%s], got %q", TruncatePolicyScoreThenRecency, TruncatePolicyRecencyThenID, cfg.TruncatePolicy),
		}
	}
	return nil
}

func validateLifecycleConfig(cfg LifecycleConfig) error {
	if cfg.RetentionDays <= 0 {
		return &Error{
			Operation: OperationQuery,
			Code:      ReasonCodeInvalidRequest,
			Layer:     LayerRuntime,
			Message:   "runtime.memory.lifecycle.retention_days must be > 0",
		}
	}
	if cfg.TTLEnabled && cfg.TTL <= 0 {
		return &Error{
			Operation: OperationQuery,
			Code:      ReasonCodeInvalidRequest,
			Layer:     LayerRuntime,
			Message:   "runtime.memory.lifecycle.ttl must be > 0 when ttl_enabled=true",
		}
	}
	if len(cfg.ForgetScopeAllow) == 0 {
		return &Error{
			Operation: OperationDelete,
			Code:      ReasonCodeInvalidRequest,
			Layer:     LayerRuntime,
			Message:   "runtime.memory.lifecycle.forget_scope_allow must not be empty",
		}
	}
	for i := range cfg.ForgetScopeAllow {
		if !isScope(cfg.ForgetScopeAllow[i]) {
			return &Error{
				Operation: OperationDelete,
				Code:      ReasonCodeInvalidRequest,
				Layer:     LayerRuntime,
				Message:   fmt.Sprintf("runtime.memory.lifecycle.forget_scope_allow[%d] must be one of [%s,%s,%s], got %q", i, ScopeSession, ScopeProject, ScopeGlobal, cfg.ForgetScopeAllow[i]),
			}
		}
	}
	return nil
}

func validateSearchConfig(cfg SearchConfig) error {
	if cfg.Hybrid.KeywordWeight < 0 || cfg.Hybrid.VectorWeight < 0 {
		return &Error{
			Operation: OperationQuery,
			Code:      ReasonCodeInvalidRequest,
			Layer:     LayerRuntime,
			Message:   "runtime.memory.search.hybrid keyword/vector weight must be >= 0",
		}
	}
	if cfg.Hybrid.Enabled && cfg.Hybrid.KeywordWeight+cfg.Hybrid.VectorWeight <= 0 {
		return &Error{
			Operation: OperationQuery,
			Code:      ReasonCodeInvalidRequest,
			Layer:     LayerRuntime,
			Message:   "runtime.memory.search.hybrid keyword/vector weight sum must be > 0 when hybrid.enabled=true",
		}
	}
	if cfg.Rerank.Enabled && cfg.Rerank.MaxCandidates <= 0 {
		return &Error{
			Operation: OperationQuery,
			Code:      ReasonCodeInvalidRequest,
			Layer:     LayerRuntime,
			Message:   "runtime.memory.search.rerank.max_candidates must be > 0 when rerank.enabled=true",
		}
	}
	if cfg.TemporalDecay.Enabled {
		if cfg.TemporalDecay.HalfLife <= 0 {
			return &Error{
				Operation: OperationQuery,
				Code:      ReasonCodeInvalidRequest,
				Layer:     LayerRuntime,
				Message:   "runtime.memory.search.temporal_decay.half_life must be > 0 when temporal_decay.enabled=true",
			}
		}
		if cfg.TemporalDecay.MaxBoostRate < 0 {
			return &Error{
				Operation: OperationQuery,
				Code:      ReasonCodeInvalidRequest,
				Layer:     LayerRuntime,
				Message:   "runtime.memory.search.temporal_decay.max_boost_rate must be >= 0",
			}
		}
	}
	switch cfg.IndexUpdatePolicy {
	case IndexUpdatePolicyIncremental, IndexUpdatePolicyFullRebuildOnProfileDrift:
	default:
		return &Error{
			Operation: OperationQuery,
			Code:      ReasonCodeInvalidRequest,
			Layer:     LayerRuntime,
			Message:   fmt.Sprintf("runtime.memory.search.index_update_policy must be one of [%s,%s], got %q", IndexUpdatePolicyIncremental, IndexUpdatePolicyFullRebuildOnProfileDrift, cfg.IndexUpdatePolicy),
		}
	}
	switch cfg.DriftRecoveryPolicy {
	case DriftRecoveryPolicyIncrementalThenFull, DriftRecoveryPolicyFullRebuild:
	default:
		return &Error{
			Operation: OperationQuery,
			Code:      ReasonCodeInvalidRequest,
			Layer:     LayerRuntime,
			Message:   fmt.Sprintf("runtime.memory.search.drift_recovery_policy must be one of [%s,%s], got %q", DriftRecoveryPolicyIncrementalThenFull, DriftRecoveryPolicyFullRebuild, cfg.DriftRecoveryPolicy),
		}
	}
	return nil
}

func newBuiltinEngine(cfg Config) (Engine, error) {
	engine, err := NewFilesystemEngine(FilesystemEngineConfig{
		RootDir: cfg.Builtin.RootDir,
		Compaction: FilesystemCompactionConfig{
			Enabled:        cfg.Builtin.Compaction.Enabled,
			MinOps:         cfg.Builtin.Compaction.MinOps,
			MaxWALBytes:    cfg.Builtin.Compaction.MaxWALBytes,
			FsyncBatchSize: cfg.Builtin.Compaction.FsyncBatchSize,
		},
		Lifecycle:          cfg.Lifecycle,
		Search:             cfg.Search,
		Profile:            cfg.External.Profile,
		Model:              cfg.External.Provider,
		IndexSchemaVersion: indexSchemaVersionV2,
	})
	if err != nil {
		return nil, err
	}
	return engine, nil
}

func normalizeError(operation string, err error) error {
	if err == nil {
		return nil
	}
	var memErr *Error
	if errors.As(err, &memErr) {
		if strings.TrimSpace(memErr.Operation) == "" {
			memErr.Operation = operation
		}
		return memErr
	}
	return &Error{
		Operation: operation,
		Code:      ReasonCodeStorageUnavailable,
		Layer:     LayerRuntime,
		Message:   "memory operation failed",
		Cause:     err,
	}
}

type engineCloser interface {
	Close() error
}

func closeEngine(engine Engine) error {
	if engine == nil {
		return nil
	}
	closer, ok := engine.(engineCloser)
	if !ok {
		return nil
	}
	return closer.Close()
}
