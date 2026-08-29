package evalcontract

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const (
	CorpusVersionV1          = "evaluation_corpus.v1"
	ExperimentVersionV1      = "experiment_comparison.v1"
	FeedbackVersionV1        = "feedback_recommendation.v1"
	BadcaseStatusReplayable  = "reproducible"
	BadcaseStatusUnavailable = "unavailable"
	BadcaseStatusDrifted     = "drifted"
)

const (
	ReasonCorpusVersionDrift         = "corpus_version_drift"
	ReasonMalformedCorpus            = "malformed_corpus"
	ReasonCorpusReferenceUnavailable = "corpus_reference_unavailable"
	ReasonBadcaseDrift               = "badcase_replay_drift"
	ReasonMetricRubricDrift          = "metric_rubric_drift"
	ReasonExperimentConflict         = "experiment_aggregate_conflict"
	ReasonApprovalMissing            = "approval_missing"
)

type Reference struct {
	URI     string `json:"uri,omitempty"`
	Digest  string `json:"digest,omitempty"`
	Version string `json:"version,omitempty"`
}

type CorpusItem struct {
	ID              string    `json:"id"`
	Scenario        string    `json:"scenario"`
	Input           Reference `json:"input,omitempty"`
	Tool            Reference `json:"tool,omitempty"`
	Policy          Reference `json:"policy,omitempty"`
	RuntimeSnapshot Reference `json:"runtime_snapshot,omitempty"`
}

type Corpus struct {
	Version string       `json:"version"`
	ID      string       `json:"id"`
	Items   []CorpusItem `json:"items"`
}

type Rubric struct {
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Declaration map[string]string `json:"declaration,omitempty"`
}

type Correlation struct {
	RunID      string    `json:"run_id,omitempty"`
	StepID     string    `json:"step_id,omitempty"`
	TraceID    string    `json:"trace_id,omitempty"`
	Artifact   Reference `json:"artifact,omitempty"`
	Checkpoint Reference `json:"checkpoint,omitempty"`
}

type Badcase struct {
	ID             string      `json:"id"`
	Category       string      `json:"category"`
	Reproduction   Reference   `json:"reproduction"`
	Correlation    Correlation `json:"correlation,omitempty"`
	Status         string      `json:"status"`
	ExpectedDigest string      `json:"expected_digest,omitempty"`
	ObservedDigest string      `json:"observed_digest,omitempty"`
}

type ShardMetric struct {
	ShardID string `json:"shard_id"`
	ItemID  string `json:"item_id"`
	Digest  string `json:"digest"`
	Passed  int    `json:"passed"`
	Total   int    `json:"total"`
}

type Experiment struct {
	Version       string        `json:"version"`
	ID            string        `json:"id"`
	CorpusVersion string        `json:"corpus_version"`
	Rubric        Rubric        `json:"rubric"`
	RunBatch      string        `json:"run_batch"`
	ExecutionMode string        `json:"execution_mode"`
	Shards        []ShardMetric `json:"shards"`
}

type ComparisonResult struct {
	ExperimentID  string `json:"experiment_id"`
	CorpusVersion string `json:"corpus_version"`
	RubricDigest  string `json:"rubric_digest"`
	ExecutionMode string `json:"execution_mode"`
	Passed        int    `json:"passed"`
	Total         int    `json:"total"`
	Digest        string `json:"digest"`
}

type FeedbackRecommendation struct {
	Version         string `json:"version"`
	ID              string `json:"id"`
	ExperimentID    string `json:"experiment_id,omitempty"`
	BadcaseID       string `json:"badcase_id,omitempty"`
	ReviewerID      string `json:"reviewer_id"`
	DecisionContext string `json:"decision_context"`
	Status          string `json:"status"`
	Recommendation  string `json:"recommendation"`
}

// CorrelationPayload projects the shared evaluation metadata consumed by both
// Run and Stream diagnostic paths. It intentionally contains references only.
func CorrelationPayload(corpusVersion, itemID, badcaseID, experimentID, rubricVersion, comparisonStatus, feedbackStatus string) map[string]any {
	return map[string]any{
		"eval_corpus_version":    strings.TrimSpace(corpusVersion),
		"eval_corpus_item_id":    strings.TrimSpace(itemID),
		"eval_badcase_id":        strings.TrimSpace(badcaseID),
		"eval_experiment_id":     strings.TrimSpace(experimentID),
		"eval_rubric_version":    strings.TrimSpace(rubricVersion),
		"eval_comparison_status": strings.TrimSpace(comparisonStatus),
		"eval_feedback_status":   strings.TrimSpace(feedbackStatus),
	}
}

func NormalizeCorpus(in Corpus) (Corpus, string, error) {
	in.Version = strings.ToLower(strings.TrimSpace(in.Version))
	in.ID = strings.TrimSpace(in.ID)
	if in.Version != CorpusVersionV1 {
		return Corpus{}, "", fmt.Errorf("%s: %s", ReasonCorpusVersionDrift, in.Version)
	}
	if in.ID == "" || len(in.Items) == 0 {
		return Corpus{}, "", fmt.Errorf("%s", ReasonMalformedCorpus)
	}
	seen := map[string]bool{}
	for i := range in.Items {
		in.Items[i].ID = strings.TrimSpace(in.Items[i].ID)
		in.Items[i].Scenario = strings.TrimSpace(in.Items[i].Scenario)
		if in.Items[i].ID == "" || in.Items[i].Scenario == "" || seen[in.Items[i].ID] {
			return Corpus{}, "", fmt.Errorf("%s", ReasonMalformedCorpus)
		}
		seen[in.Items[i].ID] = true
		normalizeReference(&in.Items[i].Input)
		normalizeReference(&in.Items[i].Tool)
		normalizeReference(&in.Items[i].Policy)
		normalizeReference(&in.Items[i].RuntimeSnapshot)
	}
	sort.Slice(in.Items, func(i, j int) bool { return in.Items[i].ID < in.Items[j].ID })
	digest, err := digestValue(in)
	return in, digest, err
}

func NormalizeRubric(in Rubric) (Rubric, string, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Version = strings.TrimSpace(in.Version)
	if in.Name == "" || in.Version == "" {
		return Rubric{}, "", fmt.Errorf("%s", ReasonMalformedCorpus)
	}
	if in.Declaration == nil {
		in.Declaration = map[string]string{}
	}
	clean := make(map[string]string, len(in.Declaration))
	for k, v := range in.Declaration {
		k = strings.TrimSpace(k)
		if k != "" {
			clean[k] = strings.TrimSpace(v)
		}
	}
	in.Declaration = clean
	d, err := digestValue(in)
	return in, d, err
}

func ClassifyBadcase(in Badcase, observedDigest string, referenceAvailable bool) (Badcase, error) {
	in.ID = strings.TrimSpace(in.ID)
	in.Category = strings.TrimSpace(in.Category)
	if in.ID == "" || in.Category == "" {
		return Badcase{}, fmt.Errorf("%s", ReasonMalformedCorpus)
	}
	if !referenceAvailable {
		in.Status = BadcaseStatusUnavailable
		return in, fmt.Errorf("%s", ReasonCorpusReferenceUnavailable)
	}
	in.ExpectedDigest = strings.TrimSpace(in.ExpectedDigest)
	in.ObservedDigest = strings.TrimSpace(observedDigest)
	if in.ExpectedDigest != "" && in.ExpectedDigest != in.ObservedDigest {
		in.Status = BadcaseStatusDrifted
		return in, fmt.Errorf("%s", ReasonBadcaseDrift)
	}
	in.Status = BadcaseStatusReplayable
	return in, nil
}

func CompareExperiments(in Experiment) (ComparisonResult, error) {
	in.Version = strings.ToLower(strings.TrimSpace(in.Version))
	in.ID = strings.TrimSpace(in.ID)
	in.CorpusVersion = strings.TrimSpace(in.CorpusVersion)
	in.RunBatch = strings.TrimSpace(in.RunBatch)
	in.ExecutionMode = strings.ToLower(strings.TrimSpace(in.ExecutionMode))
	if in.Version == "" {
		in.Version = ExperimentVersionV1
	}
	if in.Version != ExperimentVersionV1 || in.ID == "" || in.CorpusVersion == "" || in.RunBatch == "" {
		return ComparisonResult{}, fmt.Errorf("%s", ReasonMalformedCorpus)
	}
	if in.ExecutionMode != "local" && in.ExecutionMode != "distributed" {
		return ComparisonResult{}, fmt.Errorf("%s", ReasonMalformedCorpus)
	}
	_, rubricDigest, err := NormalizeRubric(in.Rubric)
	if err != nil {
		return ComparisonResult{}, err
	}
	shards := append([]ShardMetric(nil), in.Shards...)
	sort.Slice(shards, func(i, j int) bool {
		if shards[i].ShardID == shards[j].ShardID {
			return shards[i].ItemID < shards[j].ItemID
		}
		return shards[i].ShardID < shards[j].ShardID
	})
	seen := map[string]string{}
	passed, total := 0, 0
	for _, s := range shards {
		key := strings.TrimSpace(s.ShardID) + "/" + strings.TrimSpace(s.ItemID)
		if key == "/" || s.Total < 0 || s.Passed < 0 || s.Passed > s.Total {
			return ComparisonResult{}, fmt.Errorf("%s", ReasonMalformedCorpus)
		}
		if prev, ok := seen[key]; ok {
			if prev != s.Digest {
				return ComparisonResult{}, fmt.Errorf("%s", ReasonExperimentConflict)
			}
			continue
		}
		seen[key] = s.Digest
		passed += s.Passed
		total += s.Total
	}
	result := ComparisonResult{ExperimentID: in.ID, CorpusVersion: in.CorpusVersion, RubricDigest: rubricDigest, ExecutionMode: in.ExecutionMode, Passed: passed, Total: total}
	result.Digest, err = digestValue(result)
	return result, err
}

func ValidateFeedback(in FeedbackRecommendation) error {
	in.Version = strings.ToLower(strings.TrimSpace(in.Version))
	in.Status = strings.ToLower(strings.TrimSpace(in.Status))
	if in.Version == "" {
		in.Version = FeedbackVersionV1
	}
	if in.Version != FeedbackVersionV1 || in.ID == "" || (in.ExperimentID == "" && in.BadcaseID == "") || strings.TrimSpace(in.ReviewerID) == "" || strings.TrimSpace(in.DecisionContext) == "" {
		return fmt.Errorf("%s", ReasonApprovalMissing)
	}
	switch in.Status {
	case "pending", "approved", "rejected":
		return nil
	default:
		return fmt.Errorf("%s", ReasonMalformedCorpus)
	}
}

func normalizeReference(r *Reference) {
	r.URI = strings.TrimSpace(r.URI)
	r.Digest = strings.TrimSpace(r.Digest)
	r.Version = strings.TrimSpace(r.Version)
}
func digestValue(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:]), nil
}
