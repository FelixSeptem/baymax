package composer

import (
	"testing"

	"github.com/FelixSeptem/baymax/core/types"
)

func TestNewNativeEventHistoryRequiresChildSource(t *testing.T) {
	if _, err := NewNativeEventHistory(types.SourceRef{Kind: types.HistorySourceTeam, Namespace: "team", ID: "wrong"}); err == nil {
		t.Fatal("accepted non-child source")
	}
	if _, err := NewNativeEventHistory(types.SourceRef{Kind: types.HistorySourceChild, Namespace: "child.history.v1", ID: "child"}); err != nil {
		t.Fatal(err)
	}
}
