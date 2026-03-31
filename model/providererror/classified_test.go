package providererror

import (
	"errors"
	"testing"

	"github.com/FelixSeptem/baymax/core/types"
)

func TestFromErrorCanonicalToolCallingTaxonomy(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "capability_unsupported",
			err:  errors.New("provider reports tool calling unsupported"),
			want: "capability_unsupported",
		},
		{
			name: "feedback_invalid",
			err:  errors.New("tool result feedback invalid payload"),
			want: "feedback_invalid",
		},
		{
			name: "request_invalid",
			err:  errors.New("invalid argument"),
			want: "request_invalid",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := FromError(tc.err)
			classified, ok := got.(*Classified)
			if !ok {
				t.Fatalf("expected classified error, got %T", got)
			}
			if classified.Reason != tc.want {
				t.Fatalf("reason=%q, want %q", classified.Reason, tc.want)
			}
			if classified.Class != types.ErrModel {
				t.Fatalf("class=%q, want %q", classified.Class, types.ErrModel)
			}
		})
	}
}

func TestFromStatusCodeRequestInvalid(t *testing.T) {
	got := FromStatusCode(errors.New("bad request"), 400)
	classified, ok := got.(*Classified)
	if !ok {
		t.Fatalf("expected classified error, got %T", got)
	}
	if classified.Reason != "request_invalid" {
		t.Fatalf("reason=%q, want request_invalid", classified.Reason)
	}
}
