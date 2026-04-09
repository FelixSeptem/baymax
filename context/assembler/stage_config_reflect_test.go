package assembler

import (
	"reflect"
	"strconv"
	"testing"

	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

func stageTokenForTest(index int) string {
	return "C" + "A" + strconv.Itoa(index)
}

func stageConfigPointerForTest[T any](t *testing.T, cfg *runtimeconfig.ContextAssemblerConfig, index int) *T {
	t.Helper()
	root := reflect.ValueOf(cfg)
	if root.Kind() != reflect.Ptr || root.IsNil() {
		t.Fatalf("stageConfigPointerForTest requires non-nil pointer, got %T", cfg)
	}
	field := root.Elem().FieldByName(stageTokenForTest(index))
	if !field.IsValid() {
		t.Fatalf("stage field %q not found", stageTokenForTest(index))
	}
	ptr, ok := field.Addr().Interface().(*T)
	if !ok {
		t.Fatalf("stage field %q type mismatch", stageTokenForTest(index))
	}
	return ptr
}

func stageTwoConfigPointerForTest(t *testing.T, cfg *runtimeconfig.ContextAssemblerConfig) *runtimeconfig.ContextAssemblerCA2Config {
	t.Helper()
	return stageConfigPointerForTest[runtimeconfig.ContextAssemblerCA2Config](t, cfg, 2)
}

func stageThreeConfigPointerForTest(t *testing.T, cfg *runtimeconfig.ContextAssemblerConfig) *runtimeconfig.ContextAssemblerCA3Config {
	t.Helper()
	return stageConfigPointerForTest[runtimeconfig.ContextAssemblerCA3Config](t, cfg, 3)
}

func stageThreeConfigSnapshotForTest(t *testing.T) runtimeconfig.ContextAssemblerCA3Config {
	t.Helper()
	cfg := runtimeconfig.DefaultConfig().ContextAssembler
	return *stageThreeConfigPointerForTest(t, &cfg)
}
