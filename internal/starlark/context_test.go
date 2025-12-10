package starlark

import (
	"testing"

	"github.com/leapstack-labs/leapsql/internal/macro"
	"go.starlark.net/starlark"
)

func TestNewExecutionContext(t *testing.T) {
	config := starlark.NewDict(1)
	config.SetKey(starlark.String("name"), starlark.String("test_model"))

	target := &TargetInfo{
		Type:     "duckdb",
		Schema:   "main",
		Database: "test.db",
	}

	this := &ThisInfo{
		Name:   "test_model",
		Schema: "analytics",
	}

	ctx := NewExecutionContext(config, "dev", target, this)

	if ctx == nil {
		t.Fatal("NewExecutionContext returned nil")
	}

	globals := ctx.Globals()

	// Check all expected globals are present
	expectedKeys := []string{"config", "env", "target", "this"}
	for _, key := range expectedKeys {
		if _, ok := globals[key]; !ok {
			t.Errorf("global %q not found", key)
		}
	}
}

func TestExecutionContext_EvalExpr(t *testing.T) {
	config := starlark.NewDict(2)
	config.SetKey(starlark.String("name"), starlark.String("my_model"))
	config.SetKey(starlark.String("materialized"), starlark.String("table"))

	ctx := NewExecutionContext(config, "prod", nil, nil)

	tests := []struct {
		name    string
		expr    string
		want    string
		wantErr bool
	}{
		{
			name: "simple string",
			expr: `"hello"`,
			want: "hello",
		},
		{
			name: "env variable",
			expr: `env`,
			want: "prod",
		},
		{
			name: "config access",
			expr: `config["name"]`,
			want: "my_model",
		},
		{
			name: "string concatenation",
			expr: `"prefix_" + config["name"]`,
			want: "prefix_my_model",
		},
		{
			name: "conditional expression",
			expr: `"production" if env == "prod" else "development"`,
			want: "production",
		},
		{
			name: "arithmetic",
			expr: `str(1 + 2)`,
			want: "3",
		},
		{
			name:    "undefined variable",
			expr:    `undefined_var`,
			wantErr: true,
		},
		{
			name:    "syntax error",
			expr:    `if`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ctx.EvalExprString(tt.expr, "test.sql", 1)

			if (err != nil) != tt.wantErr {
				t.Errorf("EvalExprString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && result != tt.want {
				t.Errorf("EvalExprString() = %q, want %q", result, tt.want)
			}
		})
	}
}

func TestExecutionContext_EvalExpr_WithTarget(t *testing.T) {
	config := starlark.NewDict(0)
	target := &TargetInfo{
		Type:     "duckdb",
		Schema:   "analytics",
		Database: "mydb",
	}

	ctx := NewExecutionContext(config, "dev", target, nil)

	// Test target.schema access
	result, err := ctx.EvalExprString(`target.schema`, "test.sql", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "analytics" {
		t.Errorf("target.schema = %q, want \"analytics\"", result)
	}

	// Test target.type access
	result, err = ctx.EvalExprString(`target.type`, "test.sql", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "duckdb" {
		t.Errorf("target.type = %q, want \"duckdb\"", result)
	}
}

func TestExecutionContext_EvalExpr_WithThis(t *testing.T) {
	config := starlark.NewDict(0)
	this := &ThisInfo{
		Name:   "orders",
		Schema: "staging",
	}

	ctx := NewExecutionContext(config, "dev", nil, this)

	// Test this.name access
	result, err := ctx.EvalExprString(`this.name`, "test.sql", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "orders" {
		t.Errorf("this.name = %q, want \"orders\"", result)
	}

	// Test this.schema access
	result, err = ctx.EvalExprString(`this.schema`, "test.sql", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "staging" {
		t.Errorf("this.schema = %q, want \"staging\"", result)
	}
}

func TestExecutionContext_AddMacros(t *testing.T) {
	config := starlark.NewDict(0)
	ctx := NewExecutionContext(config, "dev", nil, nil)

	// Create a simple macro namespace
	macros := starlark.StringDict{
		"utils": starlark.String("mock_utils_module"),
	}

	err := ctx.AddMacros(macros)
	if err != nil {
		t.Fatalf("AddMacros() error = %v", err)
	}

	globals := ctx.Globals()
	if _, ok := globals["utils"]; !ok {
		t.Error("utils macro not found in globals")
	}
}

func TestExecutionContext_AddMacros_ConflictWithBuiltin(t *testing.T) {
	config := starlark.NewDict(0)
	ctx := NewExecutionContext(config, "dev", nil, nil)

	tests := []struct {
		name      string
		macroName string
	}{
		{"config conflict", "config"},
		{"env conflict", "env"},
		{"target conflict", "target"},
		{"this conflict", "this"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			macros := starlark.StringDict{
				tt.macroName: starlark.String("conflict"),
			}

			err := ctx.AddMacros(macros)
			if err == nil {
				t.Errorf("expected error for conflicting macro name %q", tt.macroName)
			}
		})
	}
}

func TestEvalError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  EvalError
		want string
	}{
		{
			name: "with line",
			err: EvalError{
				File:    "model.sql",
				Line:    10,
				Expr:    "undefined",
				Message: "undefined variable",
			},
			want: `model.sql:10: error evaluating "undefined": undefined variable`,
		},
		{
			name: "without line",
			err: EvalError{
				File:    "model.sql",
				Expr:    "bad",
				Message: "syntax error",
			},
			want: `model.sql: error evaluating "bad": syntax error`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewContext_WithOptions(t *testing.T) {
	config := starlark.NewDict(0)
	macros := starlark.StringDict{
		"datetime": starlark.String("datetime_module"),
	}

	ctx := NewContext(config, "prod", nil, nil, WithMacros(macros))

	globals := ctx.Globals()
	if _, ok := globals["datetime"]; !ok {
		t.Error("datetime macro not found in globals")
	}
}

func TestNewContext_WithMacroRegistry(t *testing.T) {
	config := starlark.NewDict(0)

	// Create a registry with a module
	registry := macro.NewRegistry()
	module := &macro.LoadedModule{
		Namespace: "utils",
		Path:      "/test/utils.star",
		Exports: starlark.StringDict{
			"greet": starlark.String("greet_func"),
		},
	}
	if err := registry.Register(module); err != nil {
		t.Fatalf("failed to register module: %v", err)
	}

	ctx := NewContext(config, "prod", nil, nil, WithMacroRegistry(registry))

	globals := ctx.Globals()
	utilsVal, ok := globals["utils"]
	if !ok {
		t.Fatal("utils macro not found in globals")
	}

	// Check it's a module with attribute access
	mod, ok := utilsVal.(starlark.HasAttrs)
	if !ok {
		t.Fatalf("expected HasAttrs, got %T", utilsVal)
	}

	greet, err := mod.Attr("greet")
	if err != nil {
		t.Fatalf("failed to get greet attr: %v", err)
	}
	if greet.String() != `"greet_func"` {
		t.Errorf("expected greet_func, got %s", greet.String())
	}
}

func TestNewContext_WithMacroRegistry_Nil(t *testing.T) {
	config := starlark.NewDict(0)

	// Nil registry should not cause panic
	ctx := NewContext(config, "prod", nil, nil, WithMacroRegistry(nil))

	globals := ctx.Globals()
	// Should have standard globals
	if _, ok := globals["config"]; !ok {
		t.Error("config not found")
	}
	if _, ok := globals["env"]; !ok {
		t.Error("env not found")
	}
}
