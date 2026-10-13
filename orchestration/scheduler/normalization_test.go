package scheduler

import "testing"

func TestNormalizeTaskNativeParity(t *testing.T) {
	raw := Task{TaskID: " task ", Payload: map[string]any{"priority": "high"}}
	a, e := NormalizeTask(raw)
	b, n := normalizeTask(raw)
	if e != nil || n != nil || a.TaskID != b.TaskID || a.Priority != b.Priority || a.MaxAttempts != b.MaxAttempts {
		t.Fatalf("defaults drift: %+v %v", a, e)
	}
	_, e = NormalizeTask(Task{TaskID: "t", Priority: "invalid"})
	if e == nil {
		t.Fatal("invalid priority accepted")
	}
}
