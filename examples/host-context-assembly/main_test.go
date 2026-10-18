package main

import (
	"reflect"
	"testing"
)

func TestOfflineExampleEmitsContractMarkers(t *testing.T) {
	got, err := execute()
	if err != nil {
		t.Fatalf("execute() error = %v", err)
	}
	want := []string{
		"host_context.projection_admitted",
		"host_context.run_stream_equivalent",
		"host_context.capability_expansion",
		"host_context.restore_revalidation_failed",
		"host_context.legacy_rollback",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("markers = %#v, want %#v", got, want)
	}
}
