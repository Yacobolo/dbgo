package macro

import (
	"testing"

	"go.starlark.net/starlark"
)

func TestRegistry_Register(t *testing.T) {
	registry := NewRegistry()

	module := &LoadedModule{
		Namespace: "datetime",
		Path:      "/path/to/datetime.star",
		Exports: starlark.StringDict{
			"now": starlark.String("func"),
		},
	}

	err := registry.Register(module)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !registry.Has("datetime") {
		t.Error("expected registry to have 'datetime'")
	}

	if registry.Len() != 1 {
		t.Errorf("expected len 1, got %d", registry.Len())
	}
}

func TestRegistry_ReservedNamespace(t *testing.T) {
	for _, reserved := range ReservedNamespaces {
		t.Run(reserved, func(t *testing.T) {
			registry := NewRegistry()
			module := &LoadedModule{
				Namespace: reserved,
				Path:      "/path/to/" + reserved + ".star",
				Exports:   starlark.StringDict{},
			}

			err := registry.Register(module)
			if err == nil {
				t.Errorf("expected error for reserved namespace %q", reserved)
			}

			regErr, ok := err.(*RegistryError)
			if !ok {
				t.Errorf("expected *RegistryError, got %T", err)
			}
			if regErr.Namespace != reserved {
				t.Errorf("expected namespace %q, got %q", reserved, regErr.Namespace)
			}
		})
	}
}

func TestRegistry_DuplicateNamespace(t *testing.T) {
	registry := NewRegistry()

	module1 := &LoadedModule{
		Namespace: "utils",
		Path:      "/path/to/utils.star",
		Exports:   starlark.StringDict{},
	}
	module2 := &LoadedModule{
		Namespace: "utils",
		Path:      "/other/path/utils.star",
		Exports:   starlark.StringDict{},
	}

	if err := registry.Register(module1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err := registry.Register(module2)
	if err == nil {
		t.Fatal("expected error for duplicate namespace")
	}

	regErr, ok := err.(*RegistryError)
	if !ok {
		t.Fatalf("expected *RegistryError, got %T", err)
	}
	if regErr.Namespace != "utils" {
		t.Errorf("expected namespace 'utils', got %q", regErr.Namespace)
	}
}

func TestRegistry_RegisterAll(t *testing.T) {
	registry := NewRegistry()

	modules := []*LoadedModule{
		{Namespace: "datetime", Path: "/datetime.star", Exports: starlark.StringDict{}},
		{Namespace: "math", Path: "/math.star", Exports: starlark.StringDict{}},
		{Namespace: "utils", Path: "/utils.star", Exports: starlark.StringDict{}},
	}

	err := registry.RegisterAll(modules)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if registry.Len() != 3 {
		t.Errorf("expected 3 modules, got %d", registry.Len())
	}

	for _, m := range modules {
		if !registry.Has(m.Namespace) {
			t.Errorf("expected registry to have %q", m.Namespace)
		}
	}
}

func TestRegistry_RegisterAll_StopsOnError(t *testing.T) {
	registry := NewRegistry()

	modules := []*LoadedModule{
		{Namespace: "datetime", Path: "/datetime.star", Exports: starlark.StringDict{}},
		{Namespace: "config", Path: "/config.star", Exports: starlark.StringDict{}}, // reserved
		{Namespace: "utils", Path: "/utils.star", Exports: starlark.StringDict{}},
	}

	err := registry.RegisterAll(modules)
	if err == nil {
		t.Fatal("expected error")
	}

	// Only the first one should be registered
	if registry.Len() != 1 {
		t.Errorf("expected 1 module (before error), got %d", registry.Len())
	}
}

func TestRegistry_Get(t *testing.T) {
	registry := NewRegistry()

	module := &LoadedModule{
		Namespace: "datetime",
		Path:      "/path/to/datetime.star",
		Exports: starlark.StringDict{
			"now": starlark.String("func"),
		},
	}
	registry.Register(module)

	got := registry.Get("datetime")
	if got != module {
		t.Errorf("Get returned wrong module")
	}

	got = registry.Get("nonexistent")
	if got != nil {
		t.Errorf("expected nil for nonexistent namespace")
	}
}

func TestRegistry_Namespaces(t *testing.T) {
	registry := NewRegistry()

	modules := []*LoadedModule{
		{Namespace: "zeta", Path: "/zeta.star", Exports: starlark.StringDict{}},
		{Namespace: "alpha", Path: "/alpha.star", Exports: starlark.StringDict{}},
		{Namespace: "beta", Path: "/beta.star", Exports: starlark.StringDict{}},
	}
	registry.RegisterAll(modules)

	namespaces := registry.Namespaces()
	if len(namespaces) != 3 {
		t.Fatalf("expected 3 namespaces, got %d", len(namespaces))
	}

	// Should be sorted
	expected := []string{"alpha", "beta", "zeta"}
	for i, ns := range expected {
		if namespaces[i] != ns {
			t.Errorf("expected %q at index %d, got %q", ns, i, namespaces[i])
		}
	}
}

func TestRegistry_ToStarlarkDict(t *testing.T) {
	registry := NewRegistry()

	module := &LoadedModule{
		Namespace: "utils",
		Path:      "/utils.star",
		Exports: starlark.StringDict{
			"greet": starlark.String("hello_func"),
			"add":   starlark.String("add_func"),
		},
	}
	registry.Register(module)

	dict := registry.ToStarlarkDict()
	if len(dict) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(dict))
	}

	utilsVal, ok := dict["utils"]
	if !ok {
		t.Fatal("expected 'utils' in dict")
	}

	// Check it's a module with HasAttrs
	mod, ok := utilsVal.(starlark.HasAttrs)
	if !ok {
		t.Fatalf("expected HasAttrs, got %T", utilsVal)
	}

	// Check attribute access
	greetVal, err := mod.Attr("greet")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if greetVal.String() != `"hello_func"` {
		t.Errorf("expected 'hello_func', got %s", greetVal.String())
	}

	// Check AttrNames
	attrNames := mod.AttrNames()
	if len(attrNames) != 2 {
		t.Errorf("expected 2 attr names, got %d", len(attrNames))
	}
}

func TestStarlarkModule_NoSuchAttr(t *testing.T) {
	mod := &starlarkModule{
		name:    "test",
		exports: starlark.StringDict{},
	}

	_, err := mod.Attr("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent attr")
	}
}

func TestStarlarkModule_Interface(t *testing.T) {
	mod := &starlarkModule{
		name: "test",
		exports: starlark.StringDict{
			"foo": starlark.String("bar"),
		},
	}

	// Test String()
	if mod.String() != "<module test>" {
		t.Errorf("unexpected String(): %s", mod.String())
	}

	// Test Type()
	if mod.Type() != "module" {
		t.Errorf("unexpected Type(): %s", mod.Type())
	}

	// Test Truth()
	if mod.Truth() != starlark.True {
		t.Error("expected Truth() to return True")
	}

	// Test Hash()
	_, err := mod.Hash()
	if err == nil {
		t.Error("expected error from Hash()")
	}
}

func TestLoadAndRegister(t *testing.T) {
	// Test with nonexistent directory - should return empty registry
	registry, err := LoadAndRegister("/nonexistent/path")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if registry.Len() != 0 {
		t.Errorf("expected empty registry, got %d modules", registry.Len())
	}
}
