package tools

import "testing"

func TestLooksLikeSourceCode(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected bool
	}{
		// Normal shell commands — should pass through
		{
			name:     "short command",
			command:  "go test ./...",
			expected: false,
		},
		{
			name:     "npm install",
			command:  "npm install express",
			expected: false,
		},
		{
			name:     "git status",
			command:  "git status",
			expected: false,
		},
		{
			name:     "piped command under 200 chars",
			command:  "cat file.txt | grep pattern | sort | uniq -c | head -20",
			expected: false,
		},

		// Source code — should be blocked
		{
			name: "go source code with newlines",
			command: "package main\n\nimport (\n\t\"fmt\"\n)\n\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n" +
				"type Foo struct {\n\tName string\n}\n// padding to reach 200+ chars" +
				"// more padding to reach 200+ chars for the test to work correctly here",
			expected: true,
		},

		// JSON blob — the bug we're fixing (BUG-011 variant)
		{
			name:     "JSON array as command (single line, >500 chars)",
			command:  `[{"json": "created_at", "device": "DeviceInfo", "SessionID": "sessionID", "UserAgent": "userAgent", "Type": "DeviceTypeUnknown", "OS": "DeviceOSUnknown", "Browser": "DeviceBrowserUnknown", "ScreenWidth": 0, "ScreenHeight": 0, "ViewportWidth": 0, "ViewportHeight": 0, "PixelRatio": 1.0, "Orientation": "DeviceOrientationPortrait", "IsTouch": false, "Metadata": "make(map[string]string)", "CreatedAt": "time.Now()", "UpdatedAt": "time.Now()"}]`,
			expected: true,
		},
		{
			name:     "JSON object as command (>500 chars)",
			command:  `{"file_path": "/some/path/to/file.go", "content": "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"hello world\")\n}\n", "extra_field": "` + string(make([]byte, 500)) + `"}`,
			expected: true,
		},
		{
			name:     "very long single-line command (>500 chars, no newlines)",
			command:  "echo " + string(make([]byte, 600)),
			expected: true,
		},
		{
			name:     "JSON array starting with [{ (>500 chars)",
			command:  `[{"key": "value", "nested": {"a": 1}}, {"key2": "value2", "nested2": {"b": 2}}, {"key3": "value3"}]` + string(make([]byte, 500)),
			expected: true,
		},

		// Edge cases
		{
			name:     "long but valid multi-part command (<500 chars on one line)",
			command:  "cd /tmp && mkdir -p test && echo hello > test/file.txt && cat test/file.txt && rm -rf test && echo done",
			expected: false,
		},
		{
			name:     "short JSON-like (under 200 chars)",
			command:  `[{"key": "value"}]`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := looksLikeSourceCode(tt.command)
			if result != tt.expected {
				t.Errorf("looksLikeSourceCode() = %v, want %v (command length: %d)", result, tt.expected, len(tt.command))
			}
		})
	}
}

func TestTruncateForLog(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"hello world", 5, "hello..."},
		{"", 5, ""},
	}

	for _, tt := range tests {
		result := truncateForLog(tt.input, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncateForLog(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
		}
	}
}
