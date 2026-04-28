package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestReachable_CycleDetectionGraph verifies the pure BFS helper used by
// agentRelationServiceHTTPAdapter.checkNoCycle to reject edges that would
// close a cycle (Bug 2: circular delegation).
func TestReachable_CycleDetectionGraph(t *testing.T) {
	tests := []struct {
		name string
		adj  map[string][]string
		src  string
		dst  string
		want bool
	}{
		{
			name: "empty graph src==dst",
			adj:  map[string][]string{},
			src:  "A",
			dst:  "A",
			want: true,
		},
		{
			name: "direct edge present",
			adj:  map[string][]string{"A": {"B"}},
			src:  "A",
			dst:  "B",
			want: true,
		},
		{
			name: "no back edge — A->B but B has no outgoing",
			adj:  map[string][]string{"A": {"B"}},
			src:  "B",
			dst:  "A",
			want: false,
		},
		{
			name: "transitive path A->B->C; A reaches C",
			adj:  map[string][]string{"A": {"B"}, "B": {"C"}},
			src:  "A",
			dst:  "C",
			want: true,
		},
		{
			name: "no path in reverse direction A->B->C; C cannot reach A",
			adj:  map[string][]string{"A": {"B"}, "B": {"C"}},
			src:  "C",
			dst:  "A",
			want: false,
		},
		{
			name: "disconnected components",
			adj:  map[string][]string{"A": {"B"}, "X": {"Y"}},
			src:  "A",
			dst:  "Y",
			want: false,
		},
		{
			name: "branching — A->B, A->C, B->D; A reaches D",
			adj:  map[string][]string{"A": {"B", "C"}, "B": {"D"}},
			src:  "A",
			dst:  "D",
			want: true,
		},
		{
			name: "existing cycle tolerated by BFS (no infinite loop)",
			adj:  map[string][]string{"A": {"B"}, "B": {"A"}},
			src:  "A",
			dst:  "C",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := reachable(tt.adj, tt.src, tt.dst)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestCircularDelegation_Scenarios documents the real-world scenarios the
// cycle check blocks. The adjacency maps here mirror what would live in the
// agent_relations table at the moment CreateAgentRelation is called.
//
// The rule: adding source→target closes a cycle iff there is already a path
// target→…→source. checkNoCycle delegates to reachable(adj, target, source).
func TestCircularDelegation_Scenarios(t *testing.T) {
	tests := []struct {
		name        string
		existingAdj map[string][]string
		source      string
		target      string
		wantCycle   bool
	}{
		{
			name:        "A->B on empty schema is allowed",
			existingAdj: map[string][]string{},
			source:      "A",
			target:      "B",
			wantCycle:   false,
		},
		{
			name:        "B->A after A->B closes a 2-cycle",
			existingAdj: map[string][]string{"A": {"B"}},
			source:      "B",
			target:      "A",
			wantCycle:   true,
		},
		{
			name:        "C->A after A->B->C closes a 3-cycle",
			existingAdj: map[string][]string{"A": {"B"}, "B": {"C"}},
			source:      "C",
			target:      "A",
			wantCycle:   true,
		},
		{
			name:        "A->B->C is fine; adding C->D is allowed",
			existingAdj: map[string][]string{"A": {"B"}, "B": {"C"}},
			source:      "C",
			target:      "D",
			wantCycle:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Cycle-check logic, mirroring checkNoCycleExcluding.
			got := reachable(tt.existingAdj, tt.target, tt.source)
			assert.Equal(t, tt.wantCycle, got)
		})
	}
}
