// Package embedded provides default configuration files for managed mode.
// These files are embedded into the binary so the server can self-bootstrap
// without requiring external config files.
package embedded

import _ "embed"

//go:embed prompts.yaml
var DefaultPrompts []byte

//go:embed flows.yaml
var DefaultFlows []byte
