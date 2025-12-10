package macro

import (
	"os"
	"path/filepath"
	"testing"

	"go.starlark.net/starlark"
)

func TestLoader_Load_EmptyDirectory(t *testing.T) {
	// Create temp directory
	dir := t.TempDir()
	macrosDir := filepath.Join(dir, "macros")
	if err := os.Mkdir(macrosDir, 0755); err != nil {
		t.Fatal(err)
	}

	loader := NewLoader(macrosDir)
	modules, err := loader.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(modules) != 0 {
		t.Errorf("expected 0 modules, got %d", len(modules))
	}
}

func TestLoader_Load_NonExistentDirectory(t *testing.T) {
	loader := NewLoader("/nonexistent/path/to/macros")
	modules, err := loader.Load()
	if err != nil {
		t.Fatalf("unexpected error for nonexistent dir: %v", err)
	}
	if modules != nil {
		t.Errorf("expected nil modules, got %v", modules)
	}
}

func TestLoader_Load_NotADirectory(t *testing.T) {
	// Create a file instead of directory
	dir := t.TempDir()
	filePath := filepath.Join(dir, "macros")
	if err := os.WriteFile(filePath, []byte("not a dir"), 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewLoader(filePath)
	_, err := loader.Load()
	if err == nil {
		t.Fatal("expected error for non-directory path")
	}
}

func TestLoader_Load_SingleMacro(t *testing.T) {
	dir := t.TempDir()
	macrosDir := filepath.Join(dir, "macros")
	if err := os.Mkdir(macrosDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a simple macro file
	macroContent := `
def greet(name):
    return "Hello, " + name + "!"

def add(a, b):
    return a + b

_private = "should not be exported"
`
	macroPath := filepath.Join(macrosDir, "utils.star")
	if err := os.WriteFile(macroPath, []byte(macroContent), 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewLoader(macrosDir)
	modules, err := loader.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(modules) != 1 {
		t.Fatalf("expected 1 module, got %d", len(modules))
	}

	module := modules[0]
	if module.Namespace != "utils" {
		t.Errorf("expected namespace 'utils', got %q", module.Namespace)
	}

	// Check exports
	if len(module.Exports) != 2 {
		t.Errorf("expected 2 exports, got %d", len(module.Exports))
	}
	if _, ok := module.Exports["greet"]; !ok {
		t.Error("expected 'greet' to be exported")
	}
	if _, ok := module.Exports["add"]; !ok {
		t.Error("expected 'add' to be exported")
	}
	if _, ok := module.Exports["_private"]; ok {
		t.Error("'_private' should not be exported")
	}
}

func TestLoader_Load_MultipleMacros(t *testing.T) {
	dir := t.TempDir()
	macrosDir := filepath.Join(dir, "macros")
	if err := os.Mkdir(macrosDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create multiple macro files
	files := map[string]string{
		"datetime.star": `
def now():
    return "2024-01-01"
`,
		"math.star": `
def square(x):
    return x * x
`,
	}

	for name, content := range files {
		path := filepath.Join(macrosDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	loader := NewLoader(macrosDir)
	modules, err := loader.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(modules) != 2 {
		t.Fatalf("expected 2 modules, got %d", len(modules))
	}

	// Check namespaces
	namespaces := make(map[string]bool)
	for _, m := range modules {
		namespaces[m.Namespace] = true
	}
	if !namespaces["datetime"] {
		t.Error("expected 'datetime' namespace")
	}
	if !namespaces["math"] {
		t.Error("expected 'math' namespace")
	}
}

func TestLoader_Load_SyntaxError(t *testing.T) {
	dir := t.TempDir()
	macrosDir := filepath.Join(dir, "macros")
	if err := os.Mkdir(macrosDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a macro with syntax error
	badContent := `
def broken(:
    return 1
`
	macroPath := filepath.Join(macrosDir, "broken.star")
	if err := os.WriteFile(macroPath, []byte(badContent), 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewLoader(macrosDir)
	_, err := loader.Load()
	if err == nil {
		t.Fatal("expected error for syntax error in macro")
	}

	loadErr, ok := err.(*LoadError)
	if !ok {
		t.Fatalf("expected *LoadError, got %T", err)
	}
	if loadErr.File != macroPath {
		t.Errorf("expected file %q, got %q", macroPath, loadErr.File)
	}
}

func TestLoader_Load_InvalidNamespace(t *testing.T) {
	dir := t.TempDir()
	macrosDir := filepath.Join(dir, "macros")
	if err := os.Mkdir(macrosDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a file with invalid namespace (starts with number)
	macroPath := filepath.Join(macrosDir, "123invalid.star")
	if err := os.WriteFile(macroPath, []byte("x = 1"), 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewLoader(macrosDir)
	_, err := loader.Load()
	if err == nil {
		t.Fatal("expected error for invalid namespace")
	}
}

func TestValidateNamespace(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid lowercase", "datetime", false},
		{"valid with underscore", "date_time", false},
		{"valid start with underscore", "_private", false},
		{"valid with numbers", "utils2", false},
		{"empty", "", true},
		{"starts with number", "123abc", true},
		{"contains hyphen", "date-time", true},
		{"contains space", "date time", true},
		{"contains dot", "date.time", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateNamespace(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateNamespace(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestLoader_ExecuteFunction(t *testing.T) {
	dir := t.TempDir()
	macrosDir := filepath.Join(dir, "macros")
	if err := os.Mkdir(macrosDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a macro with a function we can call
	macroContent := `
def double(x):
    return x * 2
`
	macroPath := filepath.Join(macrosDir, "math.star")
	if err := os.WriteFile(macroPath, []byte(macroContent), 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewLoader(macrosDir)
	modules, err := loader.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	module := modules[0]
	doubleFn := module.Exports["double"]
	if doubleFn == nil {
		t.Fatal("expected 'double' function")
	}

	// Call the function
	thread := &starlark.Thread{Name: "test"}
	result, err := starlark.Call(thread, doubleFn, starlark.Tuple{starlark.MakeInt(5)}, nil)
	if err != nil {
		t.Fatalf("failed to call function: %v", err)
	}

	intResult, ok := result.(starlark.Int)
	if !ok {
		t.Fatalf("expected Int result, got %T", result)
	}

	val, _ := intResult.Int64()
	if val != 10 {
		t.Errorf("expected 10, got %d", val)
	}
}
