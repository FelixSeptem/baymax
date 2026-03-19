package workflow

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

const maxSubgraphDepth = 3

var conditionTemplateVarPattern = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_.-]+)\s*\}\}`)

type graphCompileSummary struct {
	SubgraphExpansionTotal int
	ConditionTemplateTotal int
	GraphCompileFailed     bool
}

type compiledNode struct {
	rawID    string
	steps    []Step
	entryIDs []string
	exitIDs  []string
	rawDeps  []string
}

func (e *Engine) compileDefinition(def Definition) (Definition, graphCompileSummary, error) {
	summary := graphCompileSummary{}
	if !e.graphComposabilityEnabled {
		if hasGraphComposabilitySyntax(def) {
			summary.GraphCompileFailed = true
			return Definition{}, summary, ValidationErrors{
				{
					Code:    ErrCodeGraphComposabilityDisabled,
					Field:   "workflow.graph_composability.enabled",
					Message: "workflow graph composability is disabled",
				},
			}
		}
		return def, summary, nil
	}
	if !hasGraphComposabilitySyntax(def) {
		return def, summary, nil
	}

	expandedSteps, err := compileScope(def, def.Steps, "", 0, nil, &summary)
	if err != nil {
		summary.GraphCompileFailed = true
		return Definition{}, summary, err
	}
	if err := ensureExpandedStepIDUniqueness(expandedSteps); err != nil {
		summary.GraphCompileFailed = true
		return Definition{}, summary, err
	}

	out := def
	out.Steps = expandedSteps
	out.Subgraphs = nil
	out.ConditionTemplates = nil
	return out, summary, nil
}

func hasGraphComposabilitySyntax(def Definition) bool {
	if len(def.Subgraphs) > 0 || len(def.ConditionTemplates) > 0 {
		return true
	}
	for _, step := range def.Steps {
		if stepUsesGraphComposability(step) {
			return true
		}
	}
	return false
}

func stepUsesGraphComposability(step Step) bool {
	return strings.TrimSpace(step.UseSubgraph) != "" ||
		strings.TrimSpace(step.Alias) != "" ||
		strings.TrimSpace(step.ConditionTemplate) != "" ||
		len(step.TemplateVars) > 0 ||
		len(step.Overrides) > 0
}

func compileScope(
	def Definition,
	rawSteps []Step,
	prefix string,
	depth int,
	stack []string,
	summary *graphCompileSummary,
) ([]Step, error) {
	nodes := make([]compiledNode, 0, len(rawSteps))
	byRawID := make(map[string]*compiledNode, len(rawSteps))
	aliasSeen := map[string]struct{}{}
	rawIDSeen := map[string]struct{}{}

	for _, rawStep := range rawSteps {
		step := rawStep
		rawID := strings.TrimSpace(step.StepID)
		if rawID == "" {
			return nil, ValidationErrors{{
				Code:    ErrCodeStepIDRequired,
				Field:   "steps.step_id",
				Message: "step_id is required",
			}}
		}
		if _, ok := rawIDSeen[rawID]; ok {
			return nil, ValidationErrors{{
				Code:    ErrCodeDuplicateStepID,
				StepID:  rawID,
				Field:   "steps.step_id",
				Message: "duplicate step id",
			}}
		}
		rawIDSeen[rawID] = struct{}{}

		node := compiledNode{
			rawID:   rawID,
			rawDeps: append([]string(nil), step.DependsOn...),
		}

		if strings.TrimSpace(step.UseSubgraph) == "" {
			expandedStep, err := compileConcreteStep(def, step, prefix, summary)
			if err != nil {
				return nil, err
			}
			node.steps = []Step{expandedStep}
			node.entryIDs = []string{expandedStep.StepID}
			node.exitIDs = []string{expandedStep.StepID}
			nodes = append(nodes, node)
			byRawID[rawID] = &nodes[len(nodes)-1]
			continue
		}

		refName := strings.TrimSpace(step.UseSubgraph)
		subgraph, ok := def.Subgraphs[refName]
		if !ok {
			return nil, ValidationErrors{{
				Code:    ErrCodeSubgraphNotFound,
				StepID:  rawID,
				Field:   "steps.use_subgraph",
				Message: fmt.Sprintf("subgraph %q not found", refName),
			}}
		}
		if depth+1 > maxSubgraphDepth {
			return nil, ValidationErrors{{
				Code:    ErrCodeSubgraphDepthExceeded,
				StepID:  rawID,
				Field:   "steps.use_subgraph",
				Message: fmt.Sprintf("subgraph recursion depth exceeds %d", maxSubgraphDepth),
			}}
		}
		if containsString(stack, refName) {
			return nil, ValidationErrors{{
				Code:    ErrCodeSubgraphCycle,
				StepID:  rawID,
				Field:   "steps.use_subgraph",
				Message: fmt.Sprintf("subgraph cycle detected for %q", refName),
			}}
		}

		alias := resolveSubgraphAlias(step)
		if strings.Contains(alias, "/") {
			return nil, ValidationErrors{{
				Code:    ErrCodeSubgraphAliasCollision,
				StepID:  rawID,
				Field:   "steps.alias",
				Message: fmt.Sprintf("subgraph alias %q contains reserved character '/'", alias),
			}}
		}
		if _, exists := aliasSeen[alias]; exists {
			return nil, ValidationErrors{{
				Code:    ErrCodeSubgraphAliasCollision,
				StepID:  rawID,
				Field:   "steps.alias",
				Message: fmt.Sprintf("duplicate subgraph alias %q", alias),
			}}
		}
		aliasSeen[alias] = struct{}{}

		nextPrefix := alias
		if prefix != "" {
			nextPrefix = prefix + "/" + alias
		}

		subgraphSteps, err := applySubgraphOverrides(step, subgraph.Steps)
		if err != nil {
			return nil, err
		}
		childSteps, err := compileScope(def, subgraphSteps, nextPrefix, depth+1, append(stack, refName), summary)
		if err != nil {
			return nil, err
		}
		entryIDs, exitIDs := inferEntryAndExit(childSteps)
		node.steps = childSteps
		node.entryIDs = entryIDs
		node.exitIDs = exitIDs
		summary.SubgraphExpansionTotal += len(childSteps)
		nodes = append(nodes, node)
		byRawID[rawID] = &nodes[len(nodes)-1]
	}

	for i := range nodes {
		resolvedDeps := make([]string, 0)
		for _, dep := range nodes[i].rawDeps {
			depID := strings.TrimSpace(dep)
			if depID == "" {
				continue
			}
			target, ok := byRawID[depID]
			if !ok {
				return nil, ValidationErrors{{
					Code:    ErrCodeMissingDependency,
					StepID:  nodes[i].rawID,
					Field:   "steps.depends_on",
					Message: fmt.Sprintf("missing dependency %q", depID),
				}}
			}
			resolvedDeps = appendUniqueStrings(resolvedDeps, target.exitIDs...)
		}
		if len(nodes[i].steps) == 1 && nodes[i].entryIDs[0] == nodes[i].steps[0].StepID && nodes[i].exitIDs[0] == nodes[i].steps[0].StepID {
			nodes[i].steps[0].DependsOn = appendUniqueStrings(nodes[i].steps[0].DependsOn, resolvedDeps...)
			continue
		}
		entrySet := make(map[string]struct{}, len(nodes[i].entryIDs))
		for _, id := range nodes[i].entryIDs {
			entrySet[id] = struct{}{}
		}
		for sIdx := range nodes[i].steps {
			if _, ok := entrySet[nodes[i].steps[sIdx].StepID]; !ok {
				continue
			}
			nodes[i].steps[sIdx].DependsOn = appendUniqueStrings(nodes[i].steps[sIdx].DependsOn, resolvedDeps...)
		}
	}

	out := make([]Step, 0)
	seenExpanded := map[string]struct{}{}
	for i := range nodes {
		for _, step := range nodes[i].steps {
			if _, ok := seenExpanded[step.StepID]; ok {
				return nil, ValidationErrors{{
					Code:    ErrCodeExpandedStepIDCollision,
					StepID:  step.StepID,
					Field:   "steps.step_id",
					Message: fmt.Sprintf("expanded step id %q collides", step.StepID),
				}}
			}
			seenExpanded[step.StepID] = struct{}{}
			out = append(out, step)
		}
	}
	return out, nil
}

func compileConcreteStep(def Definition, step Step, prefix string, summary *graphCompileSummary) (Step, error) {
	rawID := strings.TrimSpace(step.StepID)
	fullID := rawID
	if prefix != "" {
		fullID = prefix + "/" + rawID
	}
	step.StepID = fullID
	step.TaskID = strings.TrimSpace(step.TaskID)
	if step.TaskID == "" || step.TaskID == rawID {
		step.TaskID = fullID
	}
	step.DependsOn = nil
	if err := applyConditionTemplate(def, &step, summary); err != nil {
		return Step{}, err
	}
	step.UseSubgraph = ""
	step.Alias = ""
	step.ConditionTemplate = ""
	step.TemplateVars = nil
	step.Overrides = nil
	return step, nil
}

func applySubgraphOverrides(instance Step, steps []Step) ([]Step, error) {
	if len(instance.Overrides) == 0 {
		return cloneStepSlice(steps), nil
	}
	out := cloneStepSlice(steps)
	index := map[string]int{}
	for i := range out {
		index[strings.TrimSpace(out[i].StepID)] = i
	}
	for target, override := range instance.Overrides {
		stepID := strings.TrimSpace(target)
		idx, ok := index[stepID]
		if !ok {
			return nil, ValidationErrors{{
				Code:    ErrCodeSubgraphOverrideStepMissing,
				StepID:  strings.TrimSpace(instance.StepID),
				Field:   "steps.overrides",
				Message: fmt.Sprintf("override step %q not found in subgraph", stepID),
			}}
		}
		if override.Kind != nil && strings.TrimSpace(string(*override.Kind)) != "" {
			return nil, ValidationErrors{{
				Code:    ErrCodeSubgraphOverrideForbidden,
				StepID:  strings.TrimSpace(instance.StepID),
				Field:   "steps.overrides.kind",
				Message: "subgraph override does not allow kind",
			}}
		}
		if override.Retry != nil {
			out[idx].Retry = *override.Retry
		}
		if override.Timeout != nil {
			out[idx].Timeout = *override.Timeout
		}
	}
	return out, nil
}

func applyConditionTemplate(def Definition, step *Step, summary *graphCompileSummary) error {
	if step == nil {
		return nil
	}
	if strings.TrimSpace(step.ConditionTemplate) == "" {
		if len(step.TemplateVars) > 0 {
			return ValidationErrors{{
				Code:    ErrCodeConditionTemplateScope,
				StepID:  strings.TrimSpace(step.StepID),
				Field:   "steps.template_vars",
				Message: "template_vars requires condition_template",
			}}
		}
		return nil
	}
	templateName := strings.TrimSpace(step.ConditionTemplate)
	templateValue, ok := def.ConditionTemplates[templateName]
	if !ok {
		return ValidationErrors{{
			Code:    ErrCodeConditionTemplateNotFound,
			StepID:  strings.TrimSpace(step.StepID),
			Field:   "steps.condition_template",
			Message: fmt.Sprintf("condition template %q not found", templateName),
		}}
	}
	resolved, err := resolveConditionTemplate(templateValue, step.TemplateVars)
	if err != nil {
		return ValidationErrors{{
			Code:    ErrCodeConditionTemplateVarMissing,
			StepID:  strings.TrimSpace(step.StepID),
			Field:   "steps.template_vars",
			Message: err.Error(),
		}}
	}
	condition := normalizeCondition(StepCondition(resolved))
	switch condition {
	case ConditionAlways, ConditionOnSuccess, ConditionOnFailure:
		step.Condition = condition
		summary.ConditionTemplateTotal++
		return nil
	default:
		return ValidationErrors{{
			Code:    ErrCodeConditionTemplateScope,
			StepID:  strings.TrimSpace(step.StepID),
			Field:   "steps.condition_template",
			Message: fmt.Sprintf("condition template %q must resolve to one of [always,on_success,on_failure]", templateName),
		}}
	}
}

func resolveConditionTemplate(template string, vars map[string]string) (string, error) {
	resolved := strings.TrimSpace(template)
	matches := conditionTemplateVarPattern.FindAllStringSubmatch(template, -1)
	if len(matches) == 0 {
		return resolved, nil
	}
	out := conditionTemplateVarPattern.ReplaceAllStringFunc(template, func(token string) string {
		sub := conditionTemplateVarPattern.FindStringSubmatch(token)
		if len(sub) != 2 {
			return token
		}
		key := strings.TrimSpace(sub[1])
		value, ok := vars[key]
		if !ok {
			return token
		}
		return strings.TrimSpace(value)
	})
	missing := make([]string, 0)
	for _, match := range matches {
		if len(match) != 2 {
			continue
		}
		key := strings.TrimSpace(match[1])
		if _, ok := vars[key]; !ok {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return "", fmt.Errorf("missing condition template variables: %s", strings.Join(missing, ","))
	}
	return strings.TrimSpace(out), nil
}

func inferEntryAndExit(steps []Step) ([]string, []string) {
	if len(steps) == 0 {
		return nil, nil
	}
	stepIDs := map[string]struct{}{}
	inDegree := map[string]int{}
	outDegree := map[string]int{}
	for _, step := range steps {
		id := strings.TrimSpace(step.StepID)
		if id == "" {
			continue
		}
		stepIDs[id] = struct{}{}
		inDegree[id] = 0
		outDegree[id] = 0
	}
	for _, step := range steps {
		id := strings.TrimSpace(step.StepID)
		if id == "" {
			continue
		}
		for _, dep := range step.DependsOn {
			depID := strings.TrimSpace(dep)
			if depID == "" {
				continue
			}
			if _, ok := stepIDs[depID]; !ok {
				continue
			}
			inDegree[id]++
			outDegree[depID]++
		}
	}
	entry := make([]string, 0)
	exit := make([]string, 0)
	for id := range stepIDs {
		if inDegree[id] == 0 {
			entry = append(entry, id)
		}
		if outDegree[id] == 0 {
			exit = append(exit, id)
		}
	}
	sort.Strings(entry)
	sort.Strings(exit)
	return entry, exit
}

func ensureExpandedStepIDUniqueness(steps []Step) error {
	seen := map[string]struct{}{}
	for _, step := range steps {
		id := strings.TrimSpace(step.StepID)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			return ValidationErrors{{
				Code:    ErrCodeExpandedStepIDCollision,
				StepID:  id,
				Field:   "steps.step_id",
				Message: fmt.Sprintf("expanded step id %q collides", id),
			}}
		}
		seen[id] = struct{}{}
	}
	return nil
}

func resolveSubgraphAlias(step Step) string {
	alias := strings.TrimSpace(step.Alias)
	if alias != "" {
		return alias
	}
	alias = strings.TrimSpace(step.StepID)
	if alias != "" {
		return alias
	}
	return strings.TrimSpace(step.UseSubgraph)
}

func containsString(items []string, key string) bool {
	for _, item := range items {
		if strings.TrimSpace(item) == strings.TrimSpace(key) {
			return true
		}
	}
	return false
}

func cloneStepSlice(in []Step) []Step {
	if len(in) == 0 {
		return nil
	}
	out := make([]Step, len(in))
	copy(out, in)
	return out
}

func appendUniqueStrings(base []string, extra ...string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(base)+len(extra))
	for _, item := range base {
		key := strings.TrimSpace(item)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	for _, item := range extra {
		key := strings.TrimSpace(item)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	return out
}
