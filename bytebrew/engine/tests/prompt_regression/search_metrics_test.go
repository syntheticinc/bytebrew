//go:build prompt

package prompt_regression

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCountDuplicateQueries(t *testing.T) {
	tests := []struct {
		name     string
		calls    []SmartSearchCall
		expected int
	}{
		{
			name:     "empty — no duplicates",
			calls:    nil,
			expected: 0,
		},
		{
			name:     "single call — no duplicates",
			calls:    []SmartSearchCall{{Query: "error handling"}},
			expected: 0,
		},
		{
			name: "identical pair — one duplicate",
			calls: []SmartSearchCall{
				{Query: "error handling"},
				{Query: "error handling"},
			},
			expected: 1,
		},
		{
			name: "similar pair with >70% overlap — one duplicate",
			// Jaccard("error handling patterns", "error handling in go"):
			// words1: {error, handling, patterns} words2: {error, handling, in, go}
			// intersection: {error, handling} = 2
			// union: {error, handling, patterns, in, go} = 5
			// similarity = 2/5 = 0.4 — NOT similar
			// Use a genuinely similar pair instead:
			// words1: {error, handling, go, code} words2: {error, handling, go, patterns}
			// intersection: {error, handling, go} = 3, union: {error, handling, go, code, patterns} = 5 → 0.6 < 0.7
			// words1: {error, handling, go} words2: {error, handling, go, code}
			// intersection: 3, union: 4 → 0.75 > 0.7 — duplicate!
			calls: []SmartSearchCall{
				{Query: "error handling go"},
				{Query: "error handling go code"},
			},
			expected: 1,
		},
		{
			name: "all unique queries — no duplicates",
			calls: []SmartSearchCall{
				{Query: "kubernetes deployment"},
				{Query: "docker compose"},
				{Query: "error handling"},
			},
			expected: 0,
		},
		{
			name: "three queries, two similar — one duplicate",
			// "error handling" and "error handling code" share: {error, handling}/union{error,handling,code} = 2/3 ≈ 0.67 < 0.7
			// "error handling go" vs "error handling go patterns": intersection{error,handling,go}=3, union{error,handling,go,patterns}=4 → 0.75 > 0.7
			calls: []SmartSearchCall{
				{Query: "error handling go"},
				{Query: "unique kubernetes deployment"},
				{Query: "error handling go patterns"},
			},
			expected: 1,
		},
		{
			name: "three identical queries — two duplicates (second and third marked)",
			calls: []SmartSearchCall{
				{Query: "same query"},
				{Query: "same query"},
				{Query: "same query"},
			},
			// i=0,j=1: similar → seen[1]=true, duplicates=1
			// i=0,j=2: similar → seen[2]=true, duplicates=2
			// i=1: seen[1]=true → skip
			// i=2: seen[2]=true → skip
			expected: 2,
		},
		{
			name: "two pairs of similar queries — two duplicates",
			calls: []SmartSearchCall{
				{Query: "error handling go"},
				{Query: "error handling go patterns"},
				{Query: "kubernetes deployment cluster"},
				{Query: "kubernetes deployment cluster nodes"},
			},
			// Pair 1: "error handling go" vs "error handling go patterns" → 3/4 = 0.75 > 0.7
			// Pair 2: "kubernetes deployment cluster" vs "kubernetes deployment cluster nodes" → 3/4 = 0.75 > 0.7
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := countDuplicateQueries(tt.calls)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestQuerySimilarity(t *testing.T) {
	tests := []struct {
		name    string
		q1      string
		q2      string
		wantMin float64
		wantMax float64
	}{
		{
			name:    "identical queries — similarity is 1.0",
			q1:      "error handling in go",
			q2:      "error handling in go",
			wantMin: 1.0,
			wantMax: 1.0,
		},
		{
			name:    "completely different queries — similarity is 0.0",
			q1:      "kubernetes deployment",
			q2:      "authentication tokens",
			wantMin: 0.0,
			wantMax: 0.0,
		},
		{
			name: "partial overlap — similarity between 0 and 1",
			// words1: {error, handling, go} words2: {error, handling, patterns}
			// intersection: {error, handling} = 2
			// union: {error, handling, go, patterns} = 4
			// similarity = 2/4 = 0.5
			q1:      "error handling go",
			q2:      "error handling patterns",
			wantMin: 0.4,
			wantMax: 0.6,
		},
		{
			name: "high overlap above threshold — similarity > 0.7",
			// words1: {error, handling, go} words2: {error, handling, go, code}
			// intersection: 3, union: 4 → 0.75
			q1:      "error handling go",
			q2:      "error handling go code",
			wantMin: 0.7,
			wantMax: 1.0,
		},
		{
			name:    "empty first query — similarity is 0",
			q1:      "",
			q2:      "error handling",
			wantMin: 0.0,
			wantMax: 0.0,
		},
		{
			name:    "empty second query — similarity is 0",
			q1:      "error handling",
			q2:      "",
			wantMin: 0.0,
			wantMax: 0.0,
		},
		{
			name:    "both empty — similarity is 0",
			q1:      "",
			q2:      "",
			wantMin: 0.0,
			wantMax: 0.0,
		},
		{
			name: "case insensitive — uppercase treated same as lowercase",
			// splitWords normalizes to lowercase, so "Error" == "error"
			q1:      "Error Handling",
			q2:      "error handling",
			wantMin: 1.0,
			wantMax: 1.0,
		},
		{
			name: "punctuation ignored",
			q1:      "error-handling, go!",
			q2:      "error handling go",
			wantMin: 1.0,
			wantMax: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := querySimilarity(tt.q1, tt.q2)
			assert.GreaterOrEqual(t, got, tt.wantMin, "similarity should be >= %f", tt.wantMin)
			assert.LessOrEqual(t, got, tt.wantMax, "similarity should be <= %f", tt.wantMax)
		})
	}
}
