package assembler

import (
	"strings"
	"testing"
)

func TestRunThresholdTuningRejectsUnsupportedSchema(t *testing.T) {
	_, err := RunThresholdTuning(ThresholdTuningRequest{
		SchemaVersion: "v2",
		Samples:       []ThresholdTuningSample{{ID: "1", Score: 0.9, Label: true}},
	})
	if err == nil {
		t.Fatal("expected schema version error")
	}
}

func TestRunThresholdTuningProducesDeterministicRecommendation(t *testing.T) {
	req := ThresholdTuningRequest{
		SchemaVersion: ThresholdTuningSchemaVersion,
		Samples: []ThresholdTuningSample{
			{ID: "1", Provider: "openai", Model: "m", Score: 0.91, Label: true},
			{ID: "2", Provider: "openai", Model: "m", Score: 0.85, Label: true},
			{ID: "3", Provider: "openai", Model: "m", Score: 0.78, Label: true},
			{ID: "4", Provider: "openai", Model: "m", Score: 0.42, Label: false},
			{ID: "5", Provider: "openai", Model: "m", Score: 0.31, Label: false},
		},
		MinPrecision: 0.7,
		MinRecall:    0.7,
	}
	left, err := RunThresholdTuning(req)
	if err != nil {
		t.Fatalf("RunThresholdTuning failed: %v", err)
	}
	right, err := RunThresholdTuning(req)
	if err != nil {
		t.Fatalf("RunThresholdTuning failed: %v", err)
	}
	if left.Recommendation.Threshold != right.Recommendation.Threshold ||
		left.Recommendation.Accepting != right.Recommendation.Accepting {
		t.Fatalf("non-deterministic recommendation: left=%#v right=%#v", left.Recommendation, right.Recommendation)
	}
}

func TestRunThresholdTuningEmitsNonAcceptingReason(t *testing.T) {
	report, err := RunThresholdTuning(ThresholdTuningRequest{
		SchemaVersion: ThresholdTuningSchemaVersion,
		Samples: []ThresholdTuningSample{
			{ID: "1", Score: 0.52, Label: true},
			{ID: "2", Score: 0.51, Label: false},
			{ID: "3", Score: 0.50, Label: true},
			{ID: "4", Score: 0.49, Label: false},
		},
		MinPrecision: 0.95,
		MinRecall:    0.95,
	})
	if err != nil {
		t.Fatalf("RunThresholdTuning failed: %v", err)
	}
	if report.Recommendation.Accepting {
		t.Fatalf("recommendation should be non-accepting: %#v", report.Recommendation)
	}
	if report.Recommendation.ReasonCode == "" {
		t.Fatal("reason code should not be empty for non-accepting recommendation")
	}
}

func TestRenderThresholdTuningMarkdown(t *testing.T) {
	md := RenderThresholdTuningMarkdown(ThresholdTuningReport{
		SchemaVersion: ThresholdTuningSchemaVersion,
		Recommendation: ThresholdRecommendation{
			Threshold:  0.62,
			Precision:  0.8,
			Recall:     0.75,
			F1:         0.774,
			Accepting:  true,
			Confidence: "medium",
		},
		CorpusReadiness:    "limited",
		CorpusReadinessMsg: "guidance",
	})
	if !strings.Contains(md, "CA3 Threshold Tuning Report") {
		t.Fatalf("unexpected markdown output: %s", md)
	}
	if !strings.Contains(md, "Threshold: `0.620`") {
		t.Fatalf("threshold missing in markdown: %s", md)
	}
}
