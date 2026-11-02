package graph

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func TestCompileCanonicalizesEquivalentDefinitions(t *testing.T) {
	first, err := Compile(Definition{
		GraphID: "release", Version: "v1",
		Nodes: []Node{
			{ID: "deploy", DependsOn: []string{"build", "build"}},
			{ID: "test"},
			{ID: "build"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := Compile(Definition{
		GraphID: " release ", Version: " v1 ",
		Nodes: []Node{
			{ID: "build"},
			{ID: "deploy", DependsOn: []string{"build"}},
			{ID: "test"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first.Definition, second.Definition) {
		t.Fatalf("normalized definitions differ:\nfirst: %#v\nsecond: %#v", first.Definition, second.Definition)
	}
	if !reflect.DeepEqual(first.Order, []string{"build", "deploy", "test"}) {
		t.Fatalf("stable topological order = %v", first.Order)
	}
	if first.Identity != second.Identity {
		t.Fatalf("equivalent graph identities differ: %#v != %#v", first.Identity, second.Identity)
	}
	if first.Identity.TopologyDigest == "" {
		t.Fatal("topology digest is empty")
	}
}

func TestCompileRejectsInvalidGraphsWithoutPartialProjection(t *testing.T) {
	tests := []struct {
		name string
		def  Definition
		code ErrorCode
	}{
		{name: "missing_graph_id", def: Definition{Version: "v1", Nodes: []Node{{ID: "a"}}}, code: CodeGraphIDRequired},
		{name: "missing_version", def: Definition{GraphID: "g", Nodes: []Node{{ID: "a"}}}, code: CodeVersionRequired},
		{name: "empty_nodes", def: Definition{GraphID: "g", Version: "v1"}, code: CodeNoNodes},
		{name: "missing_node_id", def: Definition{GraphID: "g", Version: "v1", Nodes: []Node{{}}}, code: CodeNodeIDRequired},
		{name: "duplicate_node", def: Definition{GraphID: "g", Version: "v1", Nodes: []Node{{ID: "a"}, {ID: "a"}}}, code: CodeDuplicateNodeID},
		{name: "missing_dependency", def: Definition{GraphID: "g", Version: "v1", Nodes: []Node{{ID: "a", DependsOn: []string{"missing"}}}}, code: CodeDependencyNotFound},
		{name: "self_dependency", def: Definition{GraphID: "g", Version: "v1", Nodes: []Node{{ID: "a", DependsOn: []string{"a"}}}}, code: CodeDependencyCycle},
		{name: "dependency_cycle", def: Definition{GraphID: "g", Version: "v1", Nodes: []Node{{ID: "a", DependsOn: []string{"b"}}, {ID: "b", DependsOn: []string{"a"}}}}, code: CodeDependencyCycle},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compiled, err := Compile(tt.def)
			if err == nil {
				t.Fatalf("Compile() = %#v, want error", compiled)
			}
			var validationErr *ValidationError
			if !errors.As(err, &validationErr) {
				t.Fatalf("error type = %T, want *ValidationError: %v", err, err)
			}
			if validationErr.Code != tt.code {
				t.Fatalf("error code = %q, want %q", validationErr.Code, tt.code)
			}
			if !reflect.DeepEqual(compiled, Compiled{}) {
				t.Fatalf("failed compilation returned partial projection: %#v", compiled)
			}
		})
	}
}

func TestCompileRejectsInvalidUTF8AndIdentifierBounds(t *testing.T) {
	tests := []struct {
		name string
		def  Definition
		code ErrorCode
	}{
		{name: "graph_id_too_long", def: Definition{GraphID: strings.Repeat("g", MaxGraphIDBytes+1), Version: "v1", Nodes: []Node{{ID: "a"}}}, code: CodeGraphIDTooLong},
		{name: "version_too_long", def: Definition{GraphID: "g", Version: strings.Repeat("v", MaxVersionBytes+1), Nodes: []Node{{ID: "a"}}}, code: CodeVersionTooLong},
		{name: "node_id_too_long", def: Definition{GraphID: "g", Version: "v1", Nodes: []Node{{ID: strings.Repeat("n", MaxNodeIDBytes+1)}}}, code: CodeNodeIDTooLong},
		{name: "dependency_id_too_long", def: Definition{GraphID: "g", Version: "v1", Nodes: []Node{{ID: "a", DependsOn: []string{strings.Repeat("n", MaxNodeIDBytes+1)}}}}, code: CodeNodeIDTooLong},
		{name: "invalid_utf8_graph_id", def: Definition{GraphID: string([]byte{'g', 0xff}), Version: "v1", Nodes: []Node{{ID: "a"}}}, code: CodeInvalidUTF8},
		{name: "invalid_utf8_node_id", def: Definition{GraphID: "g", Version: "v1", Nodes: []Node{{ID: string([]byte{'a', 0xff})}}}, code: CodeInvalidUTF8},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := Compile(tt.def); !hasCode(err, tt.code) {
				t.Fatalf("Compile() error = %v, want %q", err, tt.code)
			}
		})
	}
}

func TestCompileRejectsCanonicalByteOverflow(t *testing.T) {
	nodes := make([]Node, MaxNodes)
	for i := range nodes {
		nodes[i].ID = strings.Repeat("n", 116) + "-" + strconv.Itoa(i)
	}
	for i := range nodes {
		for offset := 1; offset <= 8 && i+offset < len(nodes); offset++ {
			nodes[i].DependsOn = append(nodes[i].DependsOn, nodes[i+offset].ID)
		}
	}
	if _, err := Compile(Definition{GraphID: "g", Version: "v1", Nodes: nodes}); !hasCode(err, CodeCanonicalTooLarge) {
		t.Fatalf("canonical size error = %v", err)
	}
}

func TestCompileReportsBoundsAndDigestDrift(t *testing.T) {
	tooManyNodes := make([]Node, MaxNodes+1)
	for i := range tooManyNodes {
		tooManyNodes[i].ID = string(rune('a' + i))
	}
	if _, err := Compile(Definition{GraphID: "g", Version: "v1", Nodes: tooManyNodes}); !hasCode(err, CodeNodeLimitExceeded) {
		t.Fatalf("node bound error = %v", err)
	}

	base, err := Compile(Definition{GraphID: "g", Version: "v1", Nodes: []Node{{ID: "a"}, {ID: "b"}}})
	if err != nil {
		t.Fatal(err)
	}
	changed, err := Compile(Definition{GraphID: "g", Version: "v1", Nodes: []Node{{ID: "a"}, {ID: "b", DependsOn: []string{"a"}}}})
	if err != nil {
		t.Fatal(err)
	}
	if base.Identity.TopologyDigest == changed.Identity.TopologyDigest {
		t.Fatal("changing a dependency edge did not change topology digest")
	}
}

func TestCompileStopsReadingEdgesAtDeclaredLimit(t *testing.T) {
	dependencies := make([]string, MaxEdges+1)
	for i := range dependencies {
		dependencies[i] = "missing-" + strconv.Itoa(i)
	}
	_, err := Compile(Definition{GraphID: "g", Version: "v1", Nodes: []Node{{ID: "a", DependsOn: dependencies}}})
	if !hasCode(err, CodeEdgeLimitExceeded) {
		t.Fatalf("edge bound error = %v", err)
	}
}

func TestCompileRejectsRawDuplicateDependencyOverflow(t *testing.T) {
	dependencies := make([]string, MaxEdges+1)
	for i := range dependencies {
		dependencies[i] = "root"
	}
	_, err := Compile(Definition{GraphID: "g", Version: "v1", Nodes: []Node{{ID: "root"}, {ID: "leaf", DependsOn: dependencies}}})
	if !hasCode(err, CodeEdgeLimitExceeded) {
		t.Fatalf("raw duplicate dependency bound error = %v", err)
	}
}

func TestCompileReportsMixedTopologyErrorsDeterministically(t *testing.T) {
	definition := Definition{GraphID: "g", Version: "v1", Nodes: []Node{
		{ID: "z", DependsOn: []string{"missing"}},
		{ID: "a", DependsOn: []string{"a"}},
	}}
	for attempt := 0; attempt < 128; attempt++ {
		_, err := Compile(definition)
		var validationErr *ValidationError
		if !errors.As(err, &validationErr) {
			t.Fatalf("attempt %d error type = %T, want *ValidationError: %v", attempt, err, err)
		}
		if validationErr.Code != CodeDependencyCycle || validationErr.NodeID != "a" {
			t.Fatalf("attempt %d validation error = %#v, want cycle at node a", attempt, validationErr)
		}
	}
}

func TestCanonicalTopologyFixtureAndDrift(t *testing.T) {
	type fixture struct {
		Schema     string     `json:"schema"`
		Definition Definition `json:"definition"`
		Identity   Identity   `json:"identity"`
		Order      []string   `json:"order"`
	}
	data, err := os.ReadFile(filepath.Join("testdata", "execution_graph.v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	var expected fixture
	if err := json.Unmarshal(data, &expected); err != nil {
		t.Fatal(err)
	}
	if expected.Schema != TopologySchemaV1 {
		t.Fatalf("fixture schema = %q", expected.Schema)
	}
	compiled, err := Compile(expected.Definition)
	if err != nil {
		t.Fatal(err)
	}
	if compiled.Identity != expected.Identity || !reflect.DeepEqual(compiled.Order, expected.Order) {
		t.Fatalf("compiled graph drifted from fixture: %#v", compiled)
	}
	changed := expected.Definition
	changed.Nodes = append([]Node(nil), expected.Definition.Nodes...)
	changed.Nodes[1] = Node{ID: "second", DependsOn: []string{"first"}}
	changedGraph, err := Compile(changed)
	if err != nil {
		t.Fatal(err)
	}
	if changedGraph.Identity.TopologyDigest == expected.Identity.TopologyDigest {
		t.Fatal("fixture edge drift retained the original topology digest")
	}
}

func hasCode(err error, want ErrorCode) bool {
	var validationErr *ValidationError
	return errors.As(err, &validationErr) && validationErr.Code == want
}
