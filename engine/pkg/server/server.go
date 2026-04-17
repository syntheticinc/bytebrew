// Package server exposes the CE server entry point as a public API.
//
// EE builds import this package, build a Config with their ee.Extension,
// and call Run — the CE binary does the same with a nil extension.
package server

import "github.com/syntheticinc/bytebrew/engine/internal/app"

// Config is the CE server configuration. EE sets the EEExtension field.
type Config = app.ServerConfig

// Run starts the CE server with the given configuration and blocks until
// the server exits. EE callers populate Config.EEExtension before calling.
func Run(sc Config) error {
	return app.Run(sc)
}
