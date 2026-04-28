package domain

import "context"

// SpawnAncestorsKey carries the chain of agent names that have spawned the
// current execution. Used to detect spawn cycles (A → B → A).
type spawnAncestorsKey struct{}

// WithSpawnAncestor returns a context that includes `name` as an ancestor of
// any further spawn that uses this context. Order is preserved (root → leaf).
func WithSpawnAncestor(ctx context.Context, name string) context.Context {
	prev, _ := ctx.Value(spawnAncestorsKey{}).([]string)
	next := make([]string, len(prev)+1)
	copy(next, prev)
	next[len(prev)] = name
	return context.WithValue(ctx, spawnAncestorsKey{}, next)
}

// SpawnAncestorsFromContext returns the chain of agent names already spawning
// down to the current execution. Empty slice when ctx has none.
func SpawnAncestorsFromContext(ctx context.Context) []string {
	v, _ := ctx.Value(spawnAncestorsKey{}).([]string)
	return v
}
