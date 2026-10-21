package teams

import (
	"context"
	"testing"

	"github.com/FelixSeptem/baymax/core/types"
)

func TestNewNativeEventHistoryRequiresTeamSource(t *testing.T) {
	if _, err := NewNativeEventHistory(types.SourceRef{Kind: types.HistorySourceRunner, Namespace: "runner", ID: "wrong"}); err == nil {
		t.Fatal("accepted non-team source")
	}
	if _, err := NewNativeEventHistory(types.SourceRef{Kind: types.HistorySourceTeam, Namespace: "team.history.v1", ID: "team"}); err != nil {
		t.Fatal(err)
	}
}

func TestTeamTimelineCanAppendExplicitSourceFacts(t *testing.T) {
	source := types.SourceRef{Kind: types.HistorySourceTeam, Namespace: "team.history.v1", ID: "team-history"}
	journal, err := NewNativeEventHistory(source)
	if err != nil {
		t.Fatal(err)
	}
	e := New(WithNativeEventHistory(journal))
	seq := int64(0)
	e.emitTimeline(context.Background(), teamTimelineContext{runID: "run-1", teamID: source.ID}, &seq, TaskStatusSucceeded, "team.resolve", nil)
	page, err := journal.ReadAfter(context.Background(), source, nil, 10)
	if err != nil || len(page.Events) != 1 || !page.Events[0].IsTerminal() {
		t.Fatalf("team source fact = %+v err=%v", page, err)
	}
}
