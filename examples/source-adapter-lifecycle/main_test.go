package main

import (
	"reflect"
	"testing"
)

func TestSourceAdapterExample(t *testing.T) {
	got, err := run()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"sent_confirmed", "gone_confirmed", "retryable_preserved", "duplicate_no_effect", "restart_confirmed_cursor", "abort_preserved"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("markers=%v", got)
	}
}
