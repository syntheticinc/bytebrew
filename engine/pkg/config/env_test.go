package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExpandEnvVars(t *testing.T) {
	tests := []struct {
		name  string
		input string
		env   map[string]string
		want  string
	}{
		{
			name:  "no placeholders",
			input: "plain string",
			want:  "plain string",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "single var",
			input: "host=${DB_HOST}",
			env:   map[string]string{"DB_HOST": "localhost"},
			want:  "host=localhost",
		},
		{
			name:  "multiple vars",
			input: "postgresql://${DB_USER}:${DB_PASS}@${DB_HOST}:5432/mydb",
			env:   map[string]string{"DB_USER": "admin", "DB_PASS": "secret", "DB_HOST": "pg.local"},
			want:  "postgresql://admin:secret@pg.local:5432/mydb",
		},
		{
			name:  "undefined var resolves to empty",
			input: "prefix-${UNDEFINED_VAR}-suffix",
			want:  "prefix--suffix",
		},
		{
			name:  "var at start",
			input: "${PORT}",
			env:   map[string]string{"PORT": "8080"},
			want:  "8080",
		},
		{
			name:  "adjacent vars",
			input: "${A}${B}",
			env:   map[string]string{"A": "hello", "B": "world"},
			want:  "helloworld",
		},
		{
			name:  "dollar without braces is not expanded",
			input: "$NOT_EXPANDED",
			want:  "$NOT_EXPANDED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			got := expandEnvVars(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSplitAndTrim(t *testing.T) {
	tests := []struct {
		name  string
		input string
		sep   string
		want  []string
	}{
		{
			name:  "simple comma separated",
			input: "a,b,c",
			sep:   ",",
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "with spaces around values",
			input: " https://example.com , https://app.example.com ",
			sep:   ",",
			want:  []string{"https://example.com", "https://app.example.com"},
		},
		{
			name:  "empty elements excluded",
			input: "a,,b, ,c",
			sep:   ",",
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "empty string",
			input: "",
			sep:   ",",
			want:  []string{},
		},
		{
			name:  "single value",
			input: "https://example.com",
			sep:   ",",
			want:  []string{"https://example.com"},
		},
		{
			name:  "only whitespace and separators",
			input: " , , ",
			sep:   ",",
			want:  []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitAndTrim(tt.input, tt.sep)
			assert.Equal(t, tt.want, got)
		})
	}
}
