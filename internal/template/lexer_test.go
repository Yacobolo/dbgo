package template

import (
	"testing"
)

func TestLexer_PlainText(t *testing.T) {
	input := "SELECT * FROM users"
	lexer := NewLexer(input, "test.sql")

	tokens, err := lexer.Tokenize()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tokens) != 2 { // TEXT + EOF
		t.Fatalf("expected 2 tokens, got %d", len(tokens))
	}

	if tokens[0].Type != TokenText {
		t.Errorf("expected TEXT, got %s", tokens[0].Type)
	}
	if tokens[0].Value != input {
		t.Errorf("expected %q, got %q", input, tokens[0].Value)
	}
	if tokens[1].Type != TokenEOF {
		t.Errorf("expected EOF, got %s", tokens[1].Type)
	}
}

func TestLexer_SimpleExpression(t *testing.T) {
	input := "SELECT {{ column }} FROM users"
	lexer := NewLexer(input, "test.sql")

	tokens, err := lexer.Tokenize()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []struct {
		typ TokenType
		val string
	}{
		{TokenText, "SELECT "},
		{TokenExpr, "column"},
		{TokenText, " FROM users"},
		{TokenEOF, ""},
	}

	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}

	for i, exp := range expected {
		if tokens[i].Type != exp.typ {
			t.Errorf("token[%d]: expected type %s, got %s", i, exp.typ, tokens[i].Type)
		}
		if exp.typ != TokenEOF && tokens[i].Value != exp.val {
			t.Errorf("token[%d]: expected value %q, got %q", i, exp.val, tokens[i].Value)
		}
	}
}

func TestLexer_MultipleExpressions(t *testing.T) {
	input := "{{ a }} + {{ b }}"
	lexer := NewLexer(input, "test.sql")

	tokens, err := lexer.Tokenize()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []struct {
		typ TokenType
		val string
	}{
		{TokenExpr, "a"},
		{TokenText, " + "},
		{TokenExpr, "b"},
		{TokenEOF, ""},
	}

	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}

	for i, exp := range expected {
		if tokens[i].Type != exp.typ {
			t.Errorf("token[%d]: expected type %s, got %s", i, exp.typ, tokens[i].Type)
		}
	}
}

func TestLexer_Statement(t *testing.T) {
	input := "{* for x in items: *}"
	lexer := NewLexer(input, "test.sql")

	tokens, err := lexer.Tokenize()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tokens) != 2 { // STMT + EOF
		t.Fatalf("expected 2 tokens, got %d", len(tokens))
	}

	if tokens[0].Type != TokenStmt {
		t.Errorf("expected STMT, got %s", tokens[0].Type)
	}
	if tokens[0].Value != "for x in items:" {
		t.Errorf("expected %q, got %q", "for x in items:", tokens[0].Value)
	}
}

func TestLexer_ForLoop(t *testing.T) {
	input := `SELECT
{* for col in columns: *}
    {{ col }},
{* endfor *}
FROM users`
	lexer := NewLexer(input, "test.sql")

	tokens, err := lexer.Tokenize()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedTypes := []TokenType{
		TokenText, // "SELECT\n"
		TokenStmt, // "for col in columns:"
		TokenText, // "\n    "
		TokenExpr, // "col"
		TokenText, // ",\n"
		TokenStmt, // "endfor"
		TokenText, // "\nFROM users"
		TokenEOF,
	}

	if len(tokens) != len(expectedTypes) {
		t.Fatalf("expected %d tokens, got %d", len(expectedTypes), len(tokens))
	}

	for i, exp := range expectedTypes {
		if tokens[i].Type != exp {
			t.Errorf("token[%d]: expected type %s, got %s", i, exp, tokens[i].Type)
		}
	}
}

func TestLexer_IfElse(t *testing.T) {
	input := `{* if condition: *}
yes
{* else: *}
no
{* endif *}`
	lexer := NewLexer(input, "test.sql")

	tokens, err := lexer.Tokenize()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedTypes := []TokenType{
		TokenStmt, // "if condition:"
		TokenText, // "\nyes\n"
		TokenStmt, // "else:"
		TokenText, // "\nno\n"
		TokenStmt, // "endif"
		TokenEOF,
	}

	if len(tokens) != len(expectedTypes) {
		t.Fatalf("expected %d tokens, got %d", len(expectedTypes), len(tokens))
	}

	for i, exp := range expectedTypes {
		if tokens[i].Type != exp {
			t.Errorf("token[%d]: expected type %s, got %s", i, exp, tokens[i].Type)
		}
	}
}

func TestLexer_UnclosedExpression(t *testing.T) {
	input := "SELECT {{ column FROM users"
	lexer := NewLexer(input, "test.sql")

	_, err := lexer.Tokenize()
	if err == nil {
		t.Fatal("expected error for unclosed expression")
	}

	lexErr, ok := err.(*LexError)
	if !ok {
		t.Fatalf("expected LexError, got %T", err)
	}

	if lexErr.Position().Line != 1 {
		t.Errorf("expected line 1, got %d", lexErr.Position().Line)
	}
}

func TestLexer_UnclosedStatement(t *testing.T) {
	input := "{* for x in items: SELECT"
	lexer := NewLexer(input, "test.sql")

	_, err := lexer.Tokenize()
	if err == nil {
		t.Fatal("expected error for unclosed statement")
	}
}

func TestLexer_NestedBraces(t *testing.T) {
	// Expression with dict literal
	input := `{{ {"key": "value"} }}`
	lexer := NewLexer(input, "test.sql")

	tokens, err := lexer.Tokenize()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tokens) != 2 { // EXPR + EOF
		t.Fatalf("expected 2 tokens, got %d", len(tokens))
	}

	if tokens[0].Type != TokenExpr {
		t.Errorf("expected EXPR, got %s", tokens[0].Type)
	}
	if tokens[0].Value != `{"key": "value"}` {
		t.Errorf("expected %q, got %q", `{"key": "value"}`, tokens[0].Value)
	}
}

func TestLexer_PositionTracking(t *testing.T) {
	input := "line1\nline2\n{{ expr }}"
	lexer := NewLexer(input, "test.sql")

	tokens, err := lexer.Tokenize()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The expression should be on line 3
	exprToken := tokens[1] // Skip first text token
	if exprToken.Type != TokenExpr {
		t.Fatalf("expected EXPR, got %s", exprToken.Type)
	}
	if exprToken.Pos.Line != 3 {
		t.Errorf("expected line 3, got %d", exprToken.Pos.Line)
	}
}

func TestLexer_WhitespaceHandling(t *testing.T) {
	// Whitespace inside delimiters should be trimmed
	tests := []struct {
		input    string
		expected string
	}{
		{"{{  x  }}", "x"},
		{"{{x}}", "x"},
		{"{{  x + y  }}", "x + y"},
		{"{*  for x in y:  *}", "for x in y:"},
	}

	for _, tt := range tests {
		lexer := NewLexer(tt.input, "test.sql")
		tokens, err := lexer.Tokenize()
		if err != nil {
			t.Errorf("input %q: unexpected error: %v", tt.input, err)
			continue
		}

		if tokens[0].Value != tt.expected {
			t.Errorf("input %q: expected %q, got %q", tt.input, tt.expected, tokens[0].Value)
		}
	}
}

func TestLexer_EmptyExpression(t *testing.T) {
	input := "{{ }}"
	lexer := NewLexer(input, "test.sql")

	tokens, err := lexer.Tokenize()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if tokens[0].Value != "" {
		t.Errorf("expected empty string, got %q", tokens[0].Value)
	}
}

func TestLexer_ComplexTemplate(t *testing.T) {
	input := `/*---
name: test
---*/

SELECT
{* for col in ["id", "name", "email"]: *}
    {{ col }},
{* endfor *}
{* if env == "prod": *}
    created_at
{* else: *}
    *
{* endif *}
FROM {{ target.schema }}.users`

	lexer := NewLexer(input, "test.sql")

	tokens, err := lexer.Tokenize()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Count tokens by type
	counts := make(map[TokenType]int)
	for _, tok := range tokens {
		counts[tok.Type]++
	}

	// Expressions: {{ col }}, {{ target.schema }} = 2
	if counts[TokenExpr] != 2 {
		t.Errorf("expected 2 expressions, got %d", counts[TokenExpr])
	}

	// Statements: for, endfor, if, else, endif = 5
	if counts[TokenStmt] != 5 {
		t.Errorf("expected 5 statements, got %d", counts[TokenStmt])
	}
}
