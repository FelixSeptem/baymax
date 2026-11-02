// Package graph defines bounded, deterministic execution topology facts.
// It does not execute nodes or own execution state.
package graph

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"
)

const (
	MaxGraphIDBytes   = 128
	MaxVersionBytes   = 32
	MaxNodeIDBytes    = 128
	MaxNodes          = 1024
	MaxEdges          = 8192
	MaxCanonicalBytes = 1 << 20
	TopologySchemaV1  = "execution_graph.v1"
)

type Definition struct {
	GraphID string `json:"graph_id"`
	Version string `json:"version"`
	Nodes   []Node `json:"nodes"`
}

type Node struct {
	ID        string   `json:"id"`
	DependsOn []string `json:"depends_on,omitempty"`
}

type Identity struct {
	GraphID        string `json:"graph_id"`
	Version        string `json:"version"`
	TopologyDigest string `json:"topology_digest"`
}

type Compiled struct {
	Definition Definition `json:"definition"`
	Identity   Identity   `json:"identity"`
	Order      []string   `json:"order"`
}

type ErrorCode string

const (
	CodeGraphIDRequired    ErrorCode = "graph_id_required"
	CodeGraphIDTooLong     ErrorCode = "graph_id_too_long"
	CodeVersionRequired    ErrorCode = "version_required"
	CodeVersionTooLong     ErrorCode = "version_too_long"
	CodeNoNodes            ErrorCode = "no_nodes"
	CodeInvalidUTF8        ErrorCode = "invalid_utf8"
	CodeNodeLimitExceeded  ErrorCode = "node_limit_exceeded"
	CodeEdgeLimitExceeded  ErrorCode = "edge_limit_exceeded"
	CodeCanonicalTooLarge  ErrorCode = "canonical_graph_too_large"
	CodeNodeIDRequired     ErrorCode = "node_id_required"
	CodeNodeIDTooLong      ErrorCode = "node_id_too_long"
	CodeDuplicateNodeID    ErrorCode = "duplicate_node_id"
	CodeDependencyRequired ErrorCode = "dependency_id_required"
	CodeDependencyNotFound ErrorCode = "dependency_not_found"
	CodeDependencyCycle    ErrorCode = "dependency_cycle"
)

type ValidationError struct {
	Code   ErrorCode `json:"code"`
	Field  string    `json:"field,omitempty"`
	NodeID string    `json:"node_id,omitempty"`
	Value  string    `json:"value,omitempty"`
}

func (e *ValidationError) Error() string {
	if e.NodeID != "" {
		return fmt.Sprintf("execution graph %s at node %q", e.Code, e.NodeID)
	}
	return fmt.Sprintf("execution graph %s at %s", e.Code, e.Field)
}

// Compile validates and canonicalizes a topology without invoking an owner.
// On error it returns the zero Compiled value rather than partial graph facts.
func Compile(input Definition) (Compiled, error) {
	definition := Definition{
		GraphID: strings.TrimSpace(input.GraphID),
		Version: strings.TrimSpace(input.Version),
	}
	if definition.GraphID == "" {
		return Compiled{}, &ValidationError{Code: CodeGraphIDRequired, Field: "graph_id"}
	}
	if !utf8.ValidString(definition.GraphID) {
		return Compiled{}, &ValidationError{Code: CodeInvalidUTF8, Field: "graph_id"}
	}
	if len([]byte(definition.GraphID)) > MaxGraphIDBytes {
		return Compiled{}, &ValidationError{Code: CodeGraphIDTooLong, Field: "graph_id"}
	}
	if definition.Version == "" {
		return Compiled{}, &ValidationError{Code: CodeVersionRequired, Field: "version"}
	}
	if !utf8.ValidString(definition.Version) {
		return Compiled{}, &ValidationError{Code: CodeInvalidUTF8, Field: "version"}
	}
	if len([]byte(definition.Version)) > MaxVersionBytes {
		return Compiled{}, &ValidationError{Code: CodeVersionTooLong, Field: "version"}
	}
	if len(input.Nodes) == 0 {
		return Compiled{}, &ValidationError{Code: CodeNoNodes, Field: "nodes"}
	}
	if len(input.Nodes) > MaxNodes {
		return Compiled{}, &ValidationError{Code: CodeNodeLimitExceeded, Field: "nodes"}
	}

	nodesByID := make(map[string]Node, len(input.Nodes))
	edgeCount := 0
	rawDependencyCount := 0
	for _, inputNode := range input.Nodes {
		node := Node{ID: strings.TrimSpace(inputNode.ID)}
		if node.ID == "" {
			return Compiled{}, &ValidationError{Code: CodeNodeIDRequired, Field: "nodes.id"}
		}
		if !utf8.ValidString(node.ID) {
			return Compiled{}, &ValidationError{Code: CodeInvalidUTF8, Field: "nodes.id", NodeID: node.ID}
		}
		if len([]byte(node.ID)) > MaxNodeIDBytes {
			return Compiled{}, &ValidationError{Code: CodeNodeIDTooLong, Field: "nodes.id", NodeID: node.ID}
		}
		if _, exists := nodesByID[node.ID]; exists {
			return Compiled{}, &ValidationError{Code: CodeDuplicateNodeID, Field: "nodes.id", NodeID: node.ID}
		}
		if len(inputNode.DependsOn) > MaxEdges-rawDependencyCount {
			return Compiled{}, &ValidationError{Code: CodeEdgeLimitExceeded, Field: "nodes.depends_on"}
		}
		rawDependencyCount += len(inputNode.DependsOn)
		seenDependencies := make(map[string]struct{}, len(inputNode.DependsOn))
		for _, rawDependency := range inputNode.DependsOn {
			dependency := strings.TrimSpace(rawDependency)
			if dependency == "" {
				return Compiled{}, &ValidationError{Code: CodeDependencyRequired, Field: "nodes.depends_on", NodeID: node.ID}
			}
			if !utf8.ValidString(dependency) {
				return Compiled{}, &ValidationError{Code: CodeInvalidUTF8, Field: "nodes.depends_on", NodeID: node.ID}
			}
			if len([]byte(dependency)) > MaxNodeIDBytes {
				return Compiled{}, &ValidationError{Code: CodeNodeIDTooLong, Field: "nodes.depends_on", NodeID: node.ID, Value: dependency}
			}
			if _, exists := seenDependencies[dependency]; exists {
				continue
			}
			seenDependencies[dependency] = struct{}{}
			node.DependsOn = append(node.DependsOn, dependency)
			edgeCount++
			if edgeCount > MaxEdges {
				return Compiled{}, &ValidationError{Code: CodeEdgeLimitExceeded, Field: "nodes.depends_on"}
			}
		}
		sort.Strings(node.DependsOn)
		nodesByID[node.ID] = node
	}

	definition.Nodes = make([]Node, 0, len(nodesByID))
	for _, node := range nodesByID {
		definition.Nodes = append(definition.Nodes, node)
	}
	sort.Slice(definition.Nodes, func(i, j int) bool { return definition.Nodes[i].ID < definition.Nodes[j].ID })
	for _, node := range definition.Nodes {
		for _, dependency := range node.DependsOn {
			if dependency == node.ID {
				return Compiled{}, &ValidationError{Code: CodeDependencyCycle, Field: "nodes.depends_on", NodeID: node.ID}
			}
			if _, exists := nodesByID[dependency]; !exists {
				return Compiled{}, &ValidationError{Code: CodeDependencyNotFound, Field: "nodes.depends_on", NodeID: node.ID, Value: dependency}
			}
		}
	}
	order, err := stableTopologicalOrder(definition.Nodes)
	if err != nil {
		return Compiled{}, err
	}

	canonical, err := json.Marshal(struct {
		Schema  string `json:"schema"`
		GraphID string `json:"graph_id"`
		Version string `json:"version"`
		Nodes   []Node `json:"nodes"`
	}{Schema: TopologySchemaV1, GraphID: definition.GraphID, Version: definition.Version, Nodes: definition.Nodes})
	if err != nil {
		return Compiled{}, err
	}
	if len(canonical) > MaxCanonicalBytes {
		return Compiled{}, &ValidationError{Code: CodeCanonicalTooLarge, Field: "definition"}
	}
	sum := sha256.Sum256(canonical)
	return Compiled{
		Definition: definition,
		Identity:   Identity{GraphID: definition.GraphID, Version: definition.Version, TopologyDigest: "sha256:" + hex.EncodeToString(sum[:])},
		Order:      order,
	}, nil
}

func stableTopologicalOrder(nodes []Node) ([]string, error) {
	indegree := make(map[string]int, len(nodes))
	dependents := make(map[string][]string, len(nodes))
	for _, node := range nodes {
		indegree[node.ID] = len(node.DependsOn)
		for _, dependency := range node.DependsOn {
			dependents[dependency] = append(dependents[dependency], node.ID)
		}
	}
	ready := make([]string, 0, len(nodes))
	for id, degree := range indegree {
		if degree == 0 {
			ready = append(ready, id)
		}
	}
	sort.Strings(ready)
	order := make([]string, 0, len(nodes))
	for len(ready) > 0 {
		id := ready[0]
		ready = ready[1:]
		order = append(order, id)
		for _, dependent := range dependents[id] {
			indegree[dependent]--
			if indegree[dependent] == 0 {
				index := sort.SearchStrings(ready, dependent)
				ready = append(ready, "")
				copy(ready[index+1:], ready[index:])
				ready[index] = dependent
			}
		}
	}
	if len(order) != len(nodes) {
		return nil, &ValidationError{Code: CodeDependencyCycle, Field: "nodes.depends_on"}
	}
	return order, nil
}
