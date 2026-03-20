package profile

import (
	"errors"
	"reflect"
	"testing"
)

func TestParseRecognizedProfile(t *testing.T) {
	v, err := Parse(" V1ALPHA1 ")
	if err != nil {
		t.Fatalf("parse profile: %v", err)
	}
	if v.String() != ProfileV1Alpha1 {
		t.Fatalf("unexpected profile: %s", v.String())
	}
}

func TestParseRejectsUnknownProfileDeterministically(t *testing.T) {
	_, err1 := Parse("v9alpha9")
	_, err2 := Parse("v9alpha9")
	pe1 := asProfileErr(t, err1)
	pe2 := asProfileErr(t, err2)
	if pe1.Code != CodeUnknownProfileVersion {
		t.Fatalf("unexpected code: %#v", pe1)
	}
	if !reflect.DeepEqual(pe1, pe2) {
		t.Fatalf("non-deterministic profile error: %#v vs %#v", pe1, pe2)
	}
}

func TestValidateCompatibilityDefaultWindow(t *testing.T) {
	window := DefaultWindow()
	if _, err := ValidateCompatibility(ProfileV1Alpha1, window); err != nil {
		t.Fatalf("current profile should be compatible: %v", err)
	}
	if _, err := ValidateCompatibility(ProfileV1Alpha0, window); err != nil {
		t.Fatalf("previous profile should be compatible: %v", err)
	}
}

func TestValidateCompatibilityOutOfWindow(t *testing.T) {
	window, err := NewWindow(ProfileV1Alpha0, true)
	if err != nil {
		t.Fatalf("new window: %v", err)
	}
	_, err = ValidateCompatibility(ProfileV1Alpha1, window)
	if err == nil {
		t.Fatal("expected out-of-window error")
	}
	pe := asProfileErr(t, err)
	if pe.Code != CodeProfileOutOfWindow {
		t.Fatalf("unexpected code: %#v", pe)
	}
}

func asProfileErr(t *testing.T, err error) *Error {
	t.Helper()
	if err == nil {
		t.Fatal("expected profile error")
	}
	pe := &Error{}
	if !errors.As(err, &pe) {
		t.Fatalf("expected profile error, got %T (%v)", err, err)
	}
	return pe
}
