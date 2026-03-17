package assembler

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

const ThresholdTuningSchemaVersion = "v1"

type ThresholdTuningSample struct {
	ID       string  `json:"id"`
	Provider string  `json:"provider"`
	Model    string  `json:"model"`
	Score    float64 `json:"score"`
	Label    bool    `json:"label"`
}

type ThresholdTuningRequest struct {
	SchemaVersion string                  `json:"schema_version"`
	Samples       []ThresholdTuningSample `json:"samples"`
	MinPrecision  float64                 `json:"min_precision"`
	MinRecall     float64                 `json:"min_recall"`
}

type ThresholdRecommendation struct {
	Threshold  float64 `json:"threshold"`
	Precision  float64 `json:"precision"`
	Recall     float64 `json:"recall"`
	F1         float64 `json:"f1"`
	Accepting  bool    `json:"accepting"`
	ReasonCode string  `json:"reason_code,omitempty"`
	Confidence string  `json:"confidence"`
}

type ThresholdTuningReport struct {
	SchemaVersion      string                  `json:"schema_version"`
	Recommendation     ThresholdRecommendation `json:"recommendation"`
	CorpusReadiness    string                  `json:"corpus_readiness"`
	CorpusReadinessMsg string                  `json:"corpus_readiness_message"`
}

func RunThresholdTuning(req ThresholdTuningRequest) (ThresholdTuningReport, error) {
	if strings.TrimSpace(req.SchemaVersion) != ThresholdTuningSchemaVersion {
		return ThresholdTuningReport{}, fmt.Errorf("unsupported schema_version %q", req.SchemaVersion)
	}
	if len(req.Samples) == 0 {
		return ThresholdTuningReport{}, errors.New("samples must not be empty")
	}
	minPrecision := req.MinPrecision
	minRecall := req.MinRecall
	if minPrecision <= 0 {
		minPrecision = 0.7
	}
	if minRecall <= 0 {
		minRecall = 0.7
	}
	thresholds := collectThresholds(req.Samples)
	best := ThresholdRecommendation{ReasonCode: "no_candidate"}
	bestF1 := -1.0
	for _, threshold := range thresholds {
		p, r, f1 := calcMetrics(req.Samples, threshold)
		if f1 > bestF1 || (f1 == bestF1 && threshold < best.Threshold) {
			bestF1 = f1
			best = ThresholdRecommendation{
				Threshold: threshold,
				Precision: p,
				Recall:    r,
				F1:        f1,
			}
		}
	}
	best.Accepting = best.Precision >= minPrecision && best.Recall >= minRecall
	if !best.Accepting {
		best.ReasonCode = "quality_gate_not_met"
	}
	best.Confidence = confidenceBySampleCount(len(req.Samples))
	corpusReadiness, corpusReadinessMsg := corpusReadinessGuidance(len(req.Samples))
	return ThresholdTuningReport{
		SchemaVersion:      ThresholdTuningSchemaVersion,
		Recommendation:     best,
		CorpusReadiness:    corpusReadiness,
		CorpusReadinessMsg: corpusReadinessMsg,
	}, nil
}

func RenderThresholdTuningMarkdown(report ThresholdTuningReport) string {
	var b strings.Builder
	b.WriteString("# CA3 Threshold Tuning Report\n\n")
	_, _ = fmt.Fprintf(&b, "- Schema: `%s`\n", report.SchemaVersion)
	_, _ = fmt.Fprintf(&b, "- Corpus readiness: `%s`\n", report.CorpusReadiness)
	_, _ = fmt.Fprintf(&b, "- Readiness guidance: %s\n\n", report.CorpusReadinessMsg)
	b.WriteString("## Recommendation\n\n")
	_, _ = fmt.Fprintf(&b, "- Threshold: `%.3f`\n", report.Recommendation.Threshold)
	_, _ = fmt.Fprintf(&b, "- Precision: `%.3f`\n", report.Recommendation.Precision)
	_, _ = fmt.Fprintf(&b, "- Recall: `%.3f`\n", report.Recommendation.Recall)
	_, _ = fmt.Fprintf(&b, "- F1: `%.3f`\n", report.Recommendation.F1)
	_, _ = fmt.Fprintf(&b, "- Accepting: `%t`\n", report.Recommendation.Accepting)
	if strings.TrimSpace(report.Recommendation.ReasonCode) != "" {
		_, _ = fmt.Fprintf(&b, "- Reason: `%s`\n", report.Recommendation.ReasonCode)
	}
	_, _ = fmt.Fprintf(&b, "- Confidence: `%s`\n", report.Recommendation.Confidence)
	return b.String()
}

func collectThresholds(samples []ThresholdTuningSample) []float64 {
	seen := map[float64]struct{}{}
	out := make([]float64, 0, len(samples)+2)
	for _, sample := range samples {
		score := sample.Score
		if score < 0 {
			score = 0
		}
		if score > 1 {
			score = 1
		}
		if _, ok := seen[score]; ok {
			continue
		}
		seen[score] = struct{}{}
		out = append(out, score)
	}
	out = append(out, 0.0, 1.0)
	sort.Float64s(out)
	return out
}

func calcMetrics(samples []ThresholdTuningSample, threshold float64) (float64, float64, float64) {
	tp := 0.0
	fp := 0.0
	fn := 0.0
	for _, sample := range samples {
		predictedPositive := sample.Score >= threshold
		if predictedPositive && sample.Label {
			tp++
		}
		if predictedPositive && !sample.Label {
			fp++
		}
		if !predictedPositive && sample.Label {
			fn++
		}
	}
	precision := 0.0
	if tp+fp > 0 {
		precision = tp / (tp + fp)
	}
	recall := 0.0
	if tp+fn > 0 {
		recall = tp / (tp + fn)
	}
	f1 := 0.0
	if precision+recall > 0 {
		f1 = 2 * precision * recall / (precision + recall)
	}
	return precision, recall, f1
}

func confidenceBySampleCount(count int) string {
	switch {
	case count >= 1000:
		return "high"
	case count >= 300:
		return "medium"
	default:
		return "low"
	}
}

func corpusReadinessGuidance(count int) (string, string) {
	switch {
	case count >= 1000:
		return "sufficient", "Sample size is typically sufficient for stable threshold estimation."
	case count >= 300:
		return "limited", "Sample size is workable but may show drift; monitor after rollout."
	default:
		return "insufficient", "Sample size is small; treat recommendation as low-confidence guidance."
	}
}
