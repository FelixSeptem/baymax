package planbridge

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"reflect"
	"strings"

	"github.com/FelixSeptem/baymax/core/types"
	"github.com/FelixSeptem/baymax/orchestration/composer"
	"github.com/FelixSeptem/baymax/orchestration/scheduler"
	"github.com/FelixSeptem/baymax/orchestration/teams"
	"github.com/FelixSeptem/baymax/orchestration/workflow"
	runtimeconfig "github.com/FelixSeptem/baymax/runtime/config"
)

// Describe populates normalized source facts. Hosts may persist this descriptor
// independently and later Compile it against another explicitly bound revision.
func Describe(d Descriptor, b Binding, l Limits) (Descriptor, error) {
	limits, err := resolveLimits(l)
	if err != nil {
		return Descriptor{}, err
	}
	_, facts, _, err := prepare(d.Shape, len(d.RequiredCapabilities), b, limits)
	if err != nil {
		return Descriptor{}, err
	}
	d.SourceFacts = facts
	if _, err = Compile(d, b, limits); err != nil {
		return Descriptor{}, err
	}
	return d, nil
}
func Compile(d Descriptor, b Binding, l Limits) (*Compiled, error) {
	limits, err := resolveLimits(l)
	if err != nil {
		return nil, err
	}
	if d.SchemaVersion != SchemaVersion {
		return nil, fail(UnsupportedVersion, "schema_version", nil)
	}
	if d.PlanVersion <= 0 {
		return nil, fail(InvalidPlan, "plan_version", nil)
	}
	refs := []struct{ name, value string }{{"plan_id", d.PlanID}, {"authority_digest", d.AuthorityDigest}, {"tool_capability_digest", d.ToolCapabilityDigest}, {"input_ref", d.InputRef}, {"input_digest", d.InputDigest}, {"handler_ref", d.HandlerRef}, {"handler_digest", d.HandlerDigest}}
	for _, r := range refs {
		if strings.TrimSpace(r.value) == "" {
			return nil, fail(InvalidPlan, r.name, nil)
		}
		if len(r.value) > limits.ReferenceBytes {
			return nil, fail(LimitExceeded, r.name, nil)
		}
	}
	if len(d.CheckpointRef) > limits.ReferenceBytes {
		return nil, fail(LimitExceeded, "checkpoint_ref", nil)
	}
	if d.Delivery != DeliveryInline {
		return nil, fail(UnsupportedCapability, "delivery", nil)
	}
	if d.Trigger != TriggerManual && d.Trigger != TriggerScheduled && d.Trigger != TriggerCallback {
		return nil, fail(UnsupportedCapability, "trigger", nil)
	}
	if len(d.RequiredCapabilities) > limits.StructuralItems {
		return nil, fail(LimitExceeded, "required_capabilities", nil)
	}
	for _, cap := range d.RequiredCapabilities {
		if cap != "source_timeline" && (cap != "token_streaming" || d.Shape != ShapeDirect) && (cap != "dynamic_pause" || d.Shape != ShapeDirect) {
			return nil, fail(UnsupportedCapability, "required_capabilities", nil)
		}
	}
	ds, err := snapshot(d, limits.DescriptorBytes, limits.NestingDepth)
	if err != nil {
		return nil, err
	}
	if err = checkReferences(reflect.ValueOf(ds.SourceFacts), limits.ReferenceBytes); err != nil {
		return nil, err
	}
	normalized, facts, cfgDigest, err := prepare(d.Shape, len(d.RequiredCapabilities), b, limits)
	if err != nil {
		return nil, err
	}
	actual, _ := encoded(facts)
	declared, _ := encoded(ds.SourceFacts)
	if !bytes.Equal(actual, declared) {
		return nil, fail(BindingMismatch, "source_facts", nil)
	}
	ds.SourceFacts = facts
	canonical, err := encoded(ds)
	if err != nil {
		return nil, err
	}
	if len(canonical) > limits.DescriptorBytes {
		return nil, fail(LimitExceeded, "descriptor.bytes", nil)
	}
	return &Compiled{identity: Identity{d.PlanID, d.PlanVersion, digest(canonical)}, descriptor: ds, binding: normalized, canonical: canonical, limits: limits, childConfigDigest: cfgDigest}, nil
}
func digest(raw []byte) string { return fmt.Sprintf("sha256:%x", sha256.Sum256(raw)) }
func checkReferences(v reflect.Value, max int) error {
	if !v.IsValid() {
		return nil
	}
	if v.Kind() == reflect.Interface {
		if v.IsNil() {
			return nil
		}
		return checkReferences(v.Elem(), max)
	}
	switch v.Kind() {
	case reflect.String:
		if len(v.String()) > max {
			return fail(LimitExceeded, "source_facts.reference", nil)
		}
	case reflect.Map:
		it := v.MapRange()
		for it.Next() {
			if e := checkReferences(it.Key(), max); e != nil {
				return e
			}
			if e := checkReferences(it.Value(), max); e != nil {
				return e
			}
		}
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			if e := checkReferences(v.Index(i), max); e != nil {
				return e
			}
		}
	}
	return nil
}

func prepare(shape Shape, requiredCapabilities int, b Binding, l Limits) (Binding, map[string]any, string, error) {
	count := 0
	for _, present := range []bool{b.Direct != nil, b.Team != nil, b.Workflow != nil, b.Child != nil} {
		if present {
			count++
		}
	}
	if count != 1 || b.Admit == nil || strings.TrimSpace(b.SemanticDigest) == "" {
		return Binding{}, nil, "", fail(InvalidPlan, "binding", nil)
	}
	if len(b.SemanticDigest) > l.ReferenceBytes {
		return Binding{}, nil, "", fail(LimitExceeded, "binding.semantic_digest", nil)
	}
	var facts map[string]any
	cfgDigest := ""
	switch shape {
	case ShapeDirect:
		if b.Direct == nil || b.Direct.Runner == nil {
			return Binding{}, nil, "", fail(BindingMismatch, "binding.direct", nil)
		}
		if b.Direct.Request.Realtime != nil {
			return Binding{}, nil, "", fail(UnsupportedCapability, "binding.direct.realtime", nil)
		}
		if requiredCapabilities+len(b.Direct.Request.Messages) > l.StructuralItems {
			return Binding{}, nil, "", fail(LimitExceeded, "binding.direct.messages", nil)
		}
		req, e := snapshot(b.Direct.Request, l.BindingBytes, l.NestingDepth)
		if e != nil {
			return Binding{}, nil, "", e
		}
		if strings.TrimSpace(req.RunID) == "" {
			return Binding{}, nil, "", fail(InvalidPlan, "binding.direct.run_id", nil)
		}
		b.Direct = &DirectBinding{Runner: b.Direct.Runner, Request: req}
		facts = asFacts(struct {
			RunID        string                       `json:"run_id"`
			SessionID    string                       `json:"session_id,omitempty"`
			Provider     string                       `json:"provider,omitempty"`
			Model        string                       `json:"model,omitempty"`
			Policy       *types.LoopPolicy            `json:"policy,omitempty"`
			Capabilities types.CapabilityRequirements `json:"capabilities"`
			Arbitration  string                       `json:"arbitration_rule_version,omitempty"`
			MessageCount int                          `json:"message_count"`
		}{req.RunID, req.SessionID, req.Provider, req.Model, req.Policy, req.Capabilities, req.ArbitrationRuleVersion, len(req.Messages)})
	case ShapeTeam:
		if b.Team == nil {
			return Binding{}, nil, "", fail(BindingMismatch, "binding.team", nil)
		}
		if !b.Team.TerminalOnly {
			return Binding{}, nil, "", fail(UnsupportedCapability, "binding.team.continuation", nil)
		}
		if requiredCapabilities+len(b.Team.Plan.Tasks) > l.StructuralItems {
			return Binding{}, nil, "", fail(LimitExceeded, "binding.team.tasks", nil)
		}
		plan, e := snapshot(b.Team.Plan, l.BindingBytes, l.NestingDepth)
		if e != nil {
			return Binding{}, nil, "", e
		}
		plan, e = teams.NormalizePlan(plan)
		if e != nil {
			return Binding{}, nil, "", fail(InvalidPlan, "binding.team.plan", e)
		}
		if plan.Strategy != teams.StrategyParallel && plan.Strategy != teams.StrategyVote {
			return Binding{}, nil, "", fail(UnsupportedCapability, "binding.team.strategy", nil)
		}
		b.Team = &TeamBinding{Plan: plan, TerminalOnly: true}
		projection := plan
		projection.Tasks = append([]teams.Task(nil), plan.Tasks...)
		for i := range projection.Tasks {
			projection.Tasks[i].Remote.Payload = nil
		}
		facts = asFacts(projection)
	case ShapeWorkflow:
		if b.Workflow == nil {
			return Binding{}, nil, "", fail(BindingMismatch, "binding.workflow", nil)
		}
		w := *b.Workflow
		if !w.TerminalOnly || w.Request.Resume {
			return Binding{}, nil, "", fail(UnsupportedCapability, "binding.workflow.continuation", nil)
		}
		def := w.Request.DSL
		if requiredCapabilities+len(def.Steps) > l.StructuralItems {
			return Binding{}, nil, "", fail(LimitExceeded, "binding.workflow.steps", nil)
		}
		if len(def.Subgraphs) > 0 || len(def.ConditionTemplates) > 0 {
			return Binding{}, nil, "", fail(UnsupportedCapability, "binding.workflow.subgraphs", nil)
		}
		deps := 0
		for _, s := range def.Steps {
			if len(s.DependsOn) > l.Dependencies-deps {
				return Binding{}, nil, "", fail(LimitExceeded, "binding.workflow.dependencies", nil)
			}
			deps += len(s.DependsOn)
			if s.UseSubgraph != "" || s.ConditionTemplate != "" || len(s.Overrides) > 0 || len(s.TemplateVars) > 0 {
				return Binding{}, nil, "", fail(UnsupportedCapability, "binding.workflow.subgraphs", nil)
			}
		}
		req, e := snapshot(w.Request, l.BindingBytes, l.NestingDepth)
		if e != nil {
			return Binding{}, nil, "", e
		}
		opts := []workflow.Option{}
		if w.DefaultStepTimeout != 0 {
			opts = append(opts, workflow.WithDefaultStepTimeout(w.DefaultStepTimeout))
		}
		req.DSL, e = workflow.New(opts...).PreflightDefinition(req.DSL)
		if e != nil {
			return Binding{}, nil, "", fail(InvalidPlan, "binding.workflow.definition", e)
		}
		for _, s := range req.DSL.Steps {
			var present bool
			switch s.Kind {
			case workflow.StepKindRunner:
				present = w.Adapter.Runner != nil
			case workflow.StepKindTool:
				present = w.Adapter.Tool != nil
			case workflow.StepKindMCP:
				present = w.Adapter.MCP != nil
			case workflow.StepKindSkill:
				present = w.Adapter.Skill != nil
			case workflow.StepKindA2A:
				present = w.Adapter.A2A != nil
			}
			if !present {
				return Binding{}, nil, "", fail(InvalidPlan, "binding.workflow.adapter", nil)
			}
		}
		w.Request = req
		b.Workflow = &w
		projection := req
		projection.DSL.Steps = append([]workflow.Step(nil), req.DSL.Steps...)
		for i := range projection.DSL.Steps {
			projection.DSL.Steps[i].Payload = nil
		}
		facts = asFacts(projection)
	case ShapeSubagent:
		if b.Child == nil {
			return Binding{}, nil, "", fail(BindingMismatch, "binding.child", nil)
		}
		child := *b.Child
		if requiredCapabilities+1 > l.StructuralItems {
			return Binding{}, nil, "", fail(LimitExceeded, "binding.child.task", nil)
		}
		req := child.Request
		if !child.TerminalOnly || req.Async || req.Target != composer.ChildTargetLocal || req.LocalRunner == nil {
			return Binding{}, nil, "", fail(UnsupportedCapability, "binding.child", nil)
		}
		if !req.Task.NotBefore.IsZero() {
			return Binding{}, nil, "", fail(UnsupportedCapability, "binding.child.delayed", nil)
		}
		task, e := snapshot(req.Task, l.BindingBytes, l.NestingDepth)
		if e != nil {
			return Binding{}, nil, "", e
		}
		cfg := childConfig(child)
		if cfg.Mailbox.Enabled || cfg.Recovery.Enabled {
			return Binding{}, nil, "", fail(UnsupportedCapability, "binding.child.background", nil)
		}
		if task.MaxAttempts <= 0 {
			task.MaxAttempts = cfg.Scheduler.RetryMaxAttempts
		}
		task, e = scheduler.NormalizeTask(task)
		if e != nil {
			return Binding{}, nil, "", fail(InvalidPlan, "binding.child.task", e)
		}
		if task.ParentRunID == "" || task.RunID == "" || req.ParentDepth < 0 || req.ParentActiveChildren < 0 || req.ParentRemainingBudget <= 0 {
			return Binding{}, nil, "", fail(InvalidPlan, "binding.child.parent", nil)
		}
		req.Task = task
		resolved, e := childTimeout(cfg, req)
		if e != nil {
			return Binding{}, nil, "", fail(InvalidPlan, "binding.child.timeout", e)
		}
		if e = guardrails(cfg).ValidateSpawn(scheduler.SpawnRequest{ParentDepth: req.ParentDepth, ParentActiveChildren: req.ParentActiveChildren}); e != nil {
			return Binding{}, nil, "", fail(InvalidPlan, "binding.child.guardrails", e)
		}
		child.Request = req
		b.Child = &child
		task.Payload = nil
		facts = asFacts(struct {
			Task             scheduler.Task       `json:"task"`
			ParentDepth      int                  `json:"parent_depth"`
			Active           int                  `json:"parent_active_children"`
			Remaining        int64                `json:"parent_remaining_budget"`
			RequestTimeout   int64                `json:"request_timeout"`
			ChildTimeout     int64                `json:"child_timeout"`
			Profile          string               `json:"operation_profile"`
			EffectiveTimeout int64                `json:"resolved_timeout"`
			Guardrails       scheduler.Guardrails `json:"guardrails"`
			ConfigDigest     string               `json:"config_digest"`
		}{task, req.ParentDepth, req.ParentActiveChildren, int64(req.ParentRemainingBudget), int64(req.RequestTimeout), int64(req.ChildTimeout), resolved.EffectiveProfile, int64(resolved.EffectiveTimeout), guardrails(cfg), childConfigDigest(cfg)})
		cfgDigest = childConfigDigest(cfg)
	default:
		return Binding{}, nil, "", fail(InvalidPlan, "shape", nil)
	}
	facts["semantic_digest"] = b.SemanticDigest
	if e := checkReferences(reflect.ValueOf(facts), l.ReferenceBytes); e != nil {
		return Binding{}, nil, "", e
	}
	if _, e := snapshot(facts, l.DescriptorBytes, l.NestingDepth); e != nil {
		return Binding{}, nil, "", e
	}
	return b, facts, cfgDigest, nil
}
func childConfig(b ChildBinding) runtimeconfig.Config {
	if b.Runtime != nil {
		return b.Runtime.EffectiveConfig()
	}
	return runtimeconfig.DefaultConfig()
}
func childConfigDigest(c runtimeconfig.Config) string {
	raw, _ := encoded(struct {
		Subagent  runtimeconfig.SubagentConfig
		Scheduler runtimeconfig.SchedulerConfig
		Profiles  runtimeconfig.RuntimeOperationProfilesConfig
		Mailbox   bool
		Recovery  bool
	}{c.Subagent, c.Scheduler, c.Runtime.OperationProfiles, c.Mailbox.Enabled, c.Recovery.Enabled})
	return digest(raw)
}
func guardrails(c runtimeconfig.Config) scheduler.Guardrails {
	return scheduler.Guardrails{MaxDepth: c.Subagent.MaxDepth, MaxActiveChildren: c.Subagent.MaxActiveChildren, ChildTimeoutBudget: c.Subagent.ChildTimeoutBudget}
}
func childTimeout(c runtimeconfig.Config, r composer.ChildDispatchRequest) (runtimeconfig.TimeoutResolutionResult, error) {
	timeout := r.RequestTimeout
	if timeout <= 0 {
		timeout = r.ChildTimeout
	}
	return runtimeconfig.ResolveOperationTimeout(c, runtimeconfig.TimeoutResolutionInput{RequestedProfile: r.OperationProfile, DomainTimeout: c.Subagent.ChildTimeoutBudget, RequestTimeout: timeout})
}
