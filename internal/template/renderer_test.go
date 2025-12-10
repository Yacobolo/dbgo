package template

import (
	"strings"
	"testing"

	starctx "github.com/user/dbgo/internal/starlark"
	"go.starlark.net/starlark"
)

func newTestContext() *starctx.ExecutionContext {
	config := starlark.NewDict(1)
	config.SetKey(starlark.String("materialized"), starlark.String("table"))

	target := &starctx.TargetInfo{
		Type:     "duckdb",
		Schema:   "analytics",
		Database: "test_db",
	}

	this := &starctx.ThisInfo{
		Name:   "test_model",
		Schema: "public",
	}

	return starctx.NewExecutionContext(config, "dev", target, this)
}

func TestRenderer_PlainText(t *testing.T) {
	input := "SELECT * FROM users"
	ctx := newTestContext()

	result, err := RenderString(input, "test.sql", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != input {
		t.Errorf("expected %q, got %q", input, result)
	}
}

func TestRenderer_SimpleExpression(t *testing.T) {
	input := `SELECT * FROM {{ target.schema }}.users`
	ctx := newTestContext()

	result, err := RenderString(input, "test.sql", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "SELECT * FROM analytics.users"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestRenderer_MultipleExpressions(t *testing.T) {
	input := `{{ target.schema }}.{{ this.name }}`
	ctx := newTestContext()

	result, err := RenderString(input, "test.sql", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "analytics.test_model"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestRenderer_EnvVariable(t *testing.T) {
	input := `{{ env }}`
	ctx := newTestContext()

	result, err := RenderString(input, "test.sql", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "dev" {
		t.Errorf("expected 'dev', got %q", result)
	}
}

func TestRenderer_ConfigAccess(t *testing.T) {
	input := `{{ config["materialized"] }}`
	ctx := newTestContext()

	result, err := RenderString(input, "test.sql", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "table" {
		t.Errorf("expected 'table', got %q", result)
	}
}

func TestRenderer_ForLoop(t *testing.T) {
	input := `SELECT
{* for col in ["id", "name", "email"]: *}
    {{ col }},
{* endfor *}
FROM users`

	ctx := newTestContext()

	result, err := RenderString(input, "test.sql", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check that all columns are present
	for _, col := range []string{"id", "name", "email"} {
		if !strings.Contains(result, col) {
			t.Errorf("expected result to contain %q, got %q", col, result)
		}
	}
}

func TestRenderer_ForLoopInline(t *testing.T) {
	input := `{* for x in [1, 2, 3]: *}{{ x }}{* endfor *}`
	ctx := newTestContext()

	result, err := RenderString(input, "test.sql", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "123" {
		t.Errorf("expected '123', got %q", result)
	}
}

func TestRenderer_IfTrue(t *testing.T) {
	input := `{* if env == "dev": *}DEV{* endif *}`
	ctx := newTestContext()

	result, err := RenderString(input, "test.sql", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "DEV" {
		t.Errorf("expected 'DEV', got %q", result)
	}
}

func TestRenderer_IfFalse(t *testing.T) {
	input := `{* if env == "prod": *}PROD{* endif *}`
	ctx := newTestContext()

	result, err := RenderString(input, "test.sql", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "" {
		t.Errorf("expected empty, got %q", result)
	}
}

func TestRenderer_IfElse(t *testing.T) {
	input := `{* if env == "prod": *}PROD{* else: *}NOT_PROD{* endif *}`
	ctx := newTestContext()

	result, err := RenderString(input, "test.sql", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "NOT_PROD" {
		t.Errorf("expected 'NOT_PROD', got %q", result)
	}
}

func TestRenderer_IfElif(t *testing.T) {
	input := `{* if env == "prod": *}PROD{* elif env == "dev": *}DEV{* else: *}OTHER{* endif *}`
	ctx := newTestContext()

	result, err := RenderString(input, "test.sql", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "DEV" {
		t.Errorf("expected 'DEV', got %q", result)
	}
}

func TestRenderer_NestedForIf(t *testing.T) {
	input := `{* for x in [1, 2, 3]: *}{* if x > 1: *}{{ x }}{* endif *}{* endfor *}`
	ctx := newTestContext()

	result, err := RenderString(input, "test.sql", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "23" {
		t.Errorf("expected '23', got %q", result)
	}
}

func TestRenderer_StringConcatenation(t *testing.T) {
	input := `{{ target.schema + "." + this.name }}`
	ctx := newTestContext()

	result, err := RenderString(input, "test.sql", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "analytics.test_model"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestRenderer_IntegerExpression(t *testing.T) {
	input := `{{ 1 + 2 }}`
	ctx := newTestContext()

	result, err := RenderString(input, "test.sql", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "3" {
		t.Errorf("expected '3', got %q", result)
	}
}

func TestRenderer_BooleanExpression(t *testing.T) {
	input := `{{ True }}`
	ctx := newTestContext()

	result, err := RenderString(input, "test.sql", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "True" {
		t.Errorf("expected 'True', got %q", result)
	}
}

func TestRenderer_ErrorInExpression(t *testing.T) {
	input := `{{ undefined_variable }}`
	ctx := newTestContext()

	_, err := RenderString(input, "test.sql", ctx)
	if err == nil {
		t.Fatal("expected error for undefined variable")
	}
}

func TestRenderer_ErrorInForIterator(t *testing.T) {
	input := `{* for x in undefined: *}{{ x }}{* endfor *}`
	ctx := newTestContext()

	_, err := RenderString(input, "test.sql", ctx)
	if err == nil {
		t.Fatal("expected error for undefined iterator")
	}
}

func TestRenderer_ErrorInCondition(t *testing.T) {
	input := `{* if undefined: *}yes{* endif *}`
	ctx := newTestContext()

	_, err := RenderString(input, "test.sql", ctx)
	if err == nil {
		t.Fatal("expected error for undefined condition")
	}
}

func TestRenderer_NonIterableFor(t *testing.T) {
	input := `{* for x in 42: *}{{ x }}{* endfor *}`
	ctx := newTestContext()

	_, err := RenderString(input, "test.sql", ctx)
	if err == nil {
		t.Fatal("expected error for non-iterable")
	}
}

func TestRenderer_FullExample(t *testing.T) {
	input := `SELECT
{* for col in ["id", "name", "created_at"]: *}
    {{ col }},
{* endfor *}
{* if env == "prod": *}
    updated_at
{* else: *}
    *
{* endif *}
FROM {{ target.schema }}.users`

	ctx := newTestContext()

	result, err := RenderString(input, "test.sql", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should contain all column names
	for _, col := range []string{"id", "name", "created_at"} {
		if !strings.Contains(result, col) {
			t.Errorf("expected result to contain %q", col)
		}
	}

	// Should contain * since env is "dev"
	if !strings.Contains(result, "*") {
		t.Error("expected result to contain '*' for dev env")
	}

	// Should not contain "updated_at" since env is "dev"
	if strings.Contains(result, "updated_at") {
		t.Error("expected result NOT to contain 'updated_at' for dev env")
	}

	// Should have correct table reference
	if !strings.Contains(result, "analytics.users") {
		t.Error("expected result to contain 'analytics.users'")
	}
}

func TestRenderer_LoopWithIndex(t *testing.T) {
	// Testing that nested variables work correctly
	input := `{* for i in [0, 1, 2]: *}
{* for j in [0, 1]: *}
({{ i }}, {{ j }})
{* endfor *}
{* endfor *}`

	ctx := newTestContext()

	result, err := RenderString(input, "test.sql", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check for some expected pairs
	expected := []string{"(0, 0)", "(0, 1)", "(1, 0)", "(1, 1)", "(2, 0)", "(2, 1)"}
	for _, exp := range expected {
		if !strings.Contains(result, exp) {
			t.Errorf("expected result to contain %q, got %q", exp, result)
		}
	}
}

func TestRenderer_EmptyLoop(t *testing.T) {
	input := `before{* for x in []: *}{{ x }}{* endfor *}after`
	ctx := newTestContext()

	result, err := RenderString(input, "test.sql", ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != "beforeafter" {
		t.Errorf("expected 'beforeafter', got %q", result)
	}
}

func TestRenderer_TruthyFalsy(t *testing.T) {
	tests := []struct {
		condition string
		expected  string
	}{
		{`True`, "yes"},
		{`False`, "no"},
		{`1`, "yes"},
		{`0`, "no"},
		{`""`, "no"},
		{`"hello"`, "yes"},
		{`[]`, "no"},
		{`[1]`, "yes"},
	}

	for _, tt := range tests {
		input := `{* if ` + tt.condition + `: *}yes{* else: *}no{* endif *}`
		ctx := newTestContext()

		result, err := RenderString(input, "test.sql", ctx)
		if err != nil {
			t.Errorf("condition %s: unexpected error: %v", tt.condition, err)
			continue
		}

		if result != tt.expected {
			t.Errorf("condition %s: expected %q, got %q", tt.condition, tt.expected, result)
		}
	}
}
