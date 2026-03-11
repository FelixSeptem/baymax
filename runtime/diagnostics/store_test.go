package diagnostics

import (
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestStoreConcurrentAccess(t *testing.T) {
	d := NewStore(32, 16, 8, 20)
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				d.AddCall(CallRecord{
					Time:      time.Now(),
					Component: "mcp",
					Transport: "http",
					CallID:    strconv.Itoa(id*100 + j),
				})
				d.AddSkill(SkillRecord{
					Time:      time.Now(),
					SkillName: "skill-a",
					Action:    "compile",
					Status:    "success",
				})
				_ = d.RecentCalls(5)
				_ = d.RecentSkills(5)
			}
		}(i)
	}
	wg.Wait()
	if got := len(d.RecentCalls(100)); got > 32 {
		t.Fatalf("call records = %d, want <= 32", got)
	}
	if got := len(d.RecentSkills(100)); got > 20 {
		t.Fatalf("skill records = %d, want <= 20", got)
	}
}

func TestSanitizeMap(t *testing.T) {
	in := map[string]any{
		"api_key": "abc",
		"nested": map[string]any{
			"token": "x",
			"name":  "ok",
		},
	}
	out := SanitizeMap(in)
	if out["api_key"] != "***" {
		t.Fatalf("api_key should be masked")
	}
	nested, _ := out["nested"].(map[string]any)
	if nested["token"] != "***" {
		t.Fatalf("nested token should be masked")
	}
	if nested["name"] != "ok" {
		t.Fatalf("non-sensitive field should keep value")
	}
}
