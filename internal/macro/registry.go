// Package macro provides functionality for loading and managing Starlark macros.
package macro

import (
	"fmt"
	"sort"

	"go.starlark.net/starlark"
)

// ReservedNamespaces are builtin globals that cannot be overridden by macros.
var ReservedNamespaces = []string{"config", "env", "target", "this"}

// Registry stores loaded macro modules and provides lookup functionality.
type Registry struct {
	modules map[string]*LoadedModule
}

// NewRegistry creates a new empty macro registry.
func NewRegistry() *Registry {
	return &Registry{
		modules: make(map[string]*LoadedModule),
	}
}

// Register adds a loaded module to the registry.
// Returns an error if the namespace is reserved or already registered.
func (r *Registry) Register(module *LoadedModule) error {
	// Check for reserved namespace collision
	for _, reserved := range ReservedNamespaces {
		if module.Namespace == reserved {
			return &RegistryError{
				Namespace: module.Namespace,
				Message:   fmt.Sprintf("cannot use reserved namespace '%s'", reserved),
			}
		}
	}

	// Check for duplicate namespace
	if existing, ok := r.modules[module.Namespace]; ok {
		return &RegistryError{
			Namespace: module.Namespace,
			Message: fmt.Sprintf("namespace already registered by %s",
				existing.Path),
		}
	}

	r.modules[module.Namespace] = module
	return nil
}

// RegisterAll registers multiple modules, stopping at the first error.
func (r *Registry) RegisterAll(modules []*LoadedModule) error {
	for _, module := range modules {
		if err := r.Register(module); err != nil {
			return err
		}
	}
	return nil
}

// Get returns the module for a given namespace, or nil if not found.
func (r *Registry) Get(namespace string) *LoadedModule {
	return r.modules[namespace]
}

// Has returns true if a namespace is registered.
func (r *Registry) Has(namespace string) bool {
	_, ok := r.modules[namespace]
	return ok
}

// Namespaces returns a sorted list of all registered namespace names.
func (r *Registry) Namespaces() []string {
	names := make([]string, 0, len(r.modules))
	for name := range r.modules {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Len returns the number of registered modules.
func (r *Registry) Len() int {
	return len(r.modules)
}

// ToStarlarkDict builds a StringDict containing all macro namespaces.
// Each namespace maps to a struct-like dict of its exported functions.
// This can be merged into the execution globals.
func (r *Registry) ToStarlarkDict() starlark.StringDict {
	result := make(starlark.StringDict, len(r.modules))

	for namespace, module := range r.modules {
		// Create a struct-like module for the namespace
		result[namespace] = &starlarkModule{
			name:    namespace,
			exports: module.Exports,
		}
	}

	return result
}

// starlarkModule wraps a module's exports as a Starlark value with attribute access.
type starlarkModule struct {
	name    string
	exports starlark.StringDict
}

// Ensure starlarkModule implements the required interfaces.
var (
	_ starlark.Value    = (*starlarkModule)(nil)
	_ starlark.HasAttrs = (*starlarkModule)(nil)
)

func (m *starlarkModule) String() string        { return fmt.Sprintf("<module %s>", m.name) }
func (m *starlarkModule) Type() string          { return "module" }
func (m *starlarkModule) Freeze()               { m.exports.Freeze() }
func (m *starlarkModule) Truth() starlark.Bool  { return starlark.True }
func (m *starlarkModule) Hash() (uint32, error) { return 0, fmt.Errorf("unhashable type: module") }

// Attr returns an attribute (exported value) by name.
func (m *starlarkModule) Attr(name string) (starlark.Value, error) {
	if v, ok := m.exports[name]; ok {
		return v, nil
	}
	return nil, starlark.NoSuchAttrError(fmt.Sprintf("module '%s' has no attribute '%s'", m.name, name))
}

// AttrNames returns a sorted list of attribute names.
func (m *starlarkModule) AttrNames() []string {
	names := make([]string, 0, len(m.exports))
	for name := range m.exports {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// RegistryError represents an error during module registration.
type RegistryError struct {
	Namespace string
	Message   string
}

func (e *RegistryError) Error() string {
	return fmt.Sprintf("registry: %s: %s", e.Namespace, e.Message)
}

// LoadAndRegister is a convenience function that loads macros from a directory
// and registers them in a new registry.
func LoadAndRegister(macrosDir string) (*Registry, error) {
	loader := NewLoader(macrosDir)
	modules, err := loader.Load()
	if err != nil {
		return nil, err
	}

	registry := NewRegistry()
	if err := registry.RegisterAll(modules); err != nil {
		return nil, err
	}

	return registry, nil
}
