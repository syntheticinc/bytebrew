package kit

import "fmt"

// Registry stores registered kits by name.
type Registry struct {
	kits map[string]Kit
}

// NewRegistry creates an empty Registry.
func NewRegistry() *Registry {
	return &Registry{kits: make(map[string]Kit)}
}

// Register adds a kit to the registry, keyed by its Name().
func (r *Registry) Register(k Kit) {
	r.kits[k.Name()] = k
}

// Get returns a kit by name or an error if not found.
func (r *Registry) Get(name string) (Kit, error) {
	k, ok := r.kits[name]
	if !ok {
		return nil, fmt.Errorf("kit %q not registered", name)
	}
	return k, nil
}

// List returns the names of all registered kits.
func (r *Registry) List() []string {
	names := make([]string, 0, len(r.kits))
	for name := range r.kits {
		names = append(names, name)
	}
	return names
}

// Has reports whether a kit with the given name is registered.
func (r *Registry) Has(name string) bool {
	_, ok := r.kits[name]
	return ok
}
