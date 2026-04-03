package runner

import (
	"fmt"
	"strings"
)

const (
	reactPlanActionCreate   = "create"
	reactPlanActionRevise   = "revise"
	reactPlanActionComplete = "complete"
	reactPlanActionRecover  = "recover"
)

const (
	reactPlanStatusPending   = "pending"
	reactPlanStatusActive    = "active"
	reactPlanStatusCompleted = "completed"
)

type reactPlanHistoryEntry struct {
	PlanID  string `json:"plan_id"`
	Version int    `json:"version"`
	Status  string `json:"status"`
	Action  string `json:"action"`
	Reason  string `json:"reason,omitempty"`
}

type reactPlanNotebook struct {
	PlanID         string                  `json:"plan_id"`
	Version        int                     `json:"version"`
	Status         string                  `json:"status"`
	History        []reactPlanHistoryEntry `json:"history,omitempty"`
	ChangeTotal    int                     `json:"change_total"`
	RecoverCount   int                     `json:"recover_count"`
	LastAction     string                  `json:"last_action,omitempty"`
	LastReason     string                  `json:"last_reason,omitempty"`
	maxHistory     int
	idempotencySet map[string]struct{}
}

type reactPlanDiagnosticsSnapshot struct {
	PlanID       string
	Version      int
	ChangeTotal  int
	LastAction   string
	LastReason   string
	RecoverCount int
	HookStatus   string
}

func newReactPlanNotebook(planID string, maxHistory int) *reactPlanNotebook {
	id := strings.TrimSpace(planID)
	if id == "" {
		id = "react-plan"
	}
	if maxHistory <= 0 {
		maxHistory = 1
	}
	return &reactPlanNotebook{
		PlanID:         id,
		Status:         reactPlanStatusPending,
		maxHistory:     maxHistory,
		idempotencySet: map[string]struct{}{},
	}
}

func snapshotReactPlanDiagnostics(n *reactPlanNotebook, hookStatus string) reactPlanDiagnosticsSnapshot {
	status := strings.ToLower(strings.TrimSpace(hookStatus))
	if status == "" {
		status = "disabled"
	}
	if n == nil {
		return reactPlanDiagnosticsSnapshot{HookStatus: status}
	}
	return reactPlanDiagnosticsSnapshot{
		PlanID:       strings.TrimSpace(n.PlanID),
		Version:      n.Version,
		ChangeTotal:  n.ChangeTotal,
		LastAction:   strings.ToLower(strings.TrimSpace(n.LastAction)),
		LastReason:   strings.TrimSpace(n.LastReason),
		RecoverCount: n.RecoverCount,
		HookStatus:   status,
	}
}

func (n *reactPlanNotebook) apply(action, reason, idempotencyKey string) error {
	if n == nil {
		return fmt.Errorf("react plan notebook is nil")
	}
	act := strings.ToLower(strings.TrimSpace(action))
	switch act {
	case reactPlanActionCreate, reactPlanActionRevise, reactPlanActionComplete, reactPlanActionRecover:
	default:
		return fmt.Errorf("react plan action must be one of [%s,%s,%s,%s], got %q",
			reactPlanActionCreate,
			reactPlanActionRevise,
			reactPlanActionComplete,
			reactPlanActionRecover,
			action,
		)
	}
	key := strings.TrimSpace(idempotencyKey)
	if key != "" {
		if _, exists := n.idempotencySet[key]; exists {
			return nil
		}
	}

	if n.Status == reactPlanStatusCompleted {
		return fmt.Errorf("react plan notebook is frozen after %q", reactPlanStatusCompleted)
	}
	if act == reactPlanActionCreate && n.Version > 0 {
		return fmt.Errorf("react plan notebook create is only allowed at initial version")
	}
	if (act == reactPlanActionRevise || act == reactPlanActionComplete) && n.Version <= 0 {
		return fmt.Errorf("react plan notebook action %q requires created plan", act)
	}

	n.Version++
	switch act {
	case reactPlanActionCreate:
		n.Status = reactPlanStatusActive
	case reactPlanActionRevise:
		n.Status = reactPlanStatusActive
	case reactPlanActionRecover:
		n.Status = reactPlanStatusActive
		n.RecoverCount++
	case reactPlanActionComplete:
		n.Status = reactPlanStatusCompleted
	}
	n.ChangeTotal++
	n.LastAction = act
	n.LastReason = strings.TrimSpace(reason)
	n.History = append(n.History, reactPlanHistoryEntry{
		PlanID:  n.PlanID,
		Version: n.Version,
		Status:  n.Status,
		Action:  act,
		Reason:  n.LastReason,
	})
	if len(n.History) > n.maxHistory {
		n.History = append([]reactPlanHistoryEntry(nil), n.History[len(n.History)-n.maxHistory:]...)
	}
	if key != "" {
		n.idempotencySet[key] = struct{}{}
	}
	return nil
}
