package template

import (
	"testing"
)

func TestParser_PlainText(t *testing.T) {
	input := "SELECT * FROM users"
	tmpl, err := ParseString(input, "test.sql")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tmpl.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(tmpl.Nodes))
	}

	text, ok := tmpl.Nodes[0].(*TextNode)
	if !ok {
		t.Fatalf("expected TextNode, got %T", tmpl.Nodes[0])
	}
	if text.Text != input {
		t.Errorf("expected %q, got %q", input, text.Text)
	}
}

func TestParser_SimpleExpression(t *testing.T) {
	input := "SELECT {{ column }} FROM users"
	tmpl, err := ParseString(input, "test.sql")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tmpl.Nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(tmpl.Nodes))
	}

	// First node: text
	text1, ok := tmpl.Nodes[0].(*TextNode)
	if !ok {
		t.Fatalf("node[0]: expected TextNode, got %T", tmpl.Nodes[0])
	}
	if text1.Text != "SELECT " {
		t.Errorf("node[0]: expected %q, got %q", "SELECT ", text1.Text)
	}

	// Second node: expression
	expr, ok := tmpl.Nodes[1].(*ExprNode)
	if !ok {
		t.Fatalf("node[1]: expected ExprNode, got %T", tmpl.Nodes[1])
	}
	if expr.Expr != "column" {
		t.Errorf("node[1]: expected %q, got %q", "column", expr.Expr)
	}

	// Third node: text
	text2, ok := tmpl.Nodes[2].(*TextNode)
	if !ok {
		t.Fatalf("node[2]: expected TextNode, got %T", tmpl.Nodes[2])
	}
	if text2.Text != " FROM users" {
		t.Errorf("node[2]: expected %q, got %q", " FROM users", text2.Text)
	}
}

func TestParser_ForLoop(t *testing.T) {
	input := `{* for col in columns: *}
{{ col }}
{* endfor *}`

	tmpl, err := ParseString(input, "test.sql")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tmpl.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(tmpl.Nodes))
	}

	forBlock, ok := tmpl.Nodes[0].(*ForBlock)
	if !ok {
		t.Fatalf("expected ForBlock, got %T", tmpl.Nodes[0])
	}

	if forBlock.VarName != "col" {
		t.Errorf("expected var name 'col', got %q", forBlock.VarName)
	}
	if forBlock.IterExpr != "columns" {
		t.Errorf("expected iter expr 'columns', got %q", forBlock.IterExpr)
	}

	// Check body
	if len(forBlock.Body) != 3 { // text, expr, text
		t.Fatalf("expected 3 body nodes, got %d", len(forBlock.Body))
	}

	expr, ok := forBlock.Body[1].(*ExprNode)
	if !ok {
		t.Fatalf("body[1]: expected ExprNode, got %T", forBlock.Body[1])
	}
	if expr.Expr != "col" {
		t.Errorf("body[1]: expected %q, got %q", "col", expr.Expr)
	}
}

func TestParser_ForLoopWithList(t *testing.T) {
	input := `{* for x in ["a", "b", "c"]: *}{{ x }}{* endfor *}`

	tmpl, err := ParseString(input, "test.sql")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	forBlock, ok := tmpl.Nodes[0].(*ForBlock)
	if !ok {
		t.Fatalf("expected ForBlock, got %T", tmpl.Nodes[0])
	}

	if forBlock.VarName != "x" {
		t.Errorf("expected var name 'x', got %q", forBlock.VarName)
	}
	if forBlock.IterExpr != `["a", "b", "c"]` {
		t.Errorf("expected iter expr '[\"a\", \"b\", \"c\"]', got %q", forBlock.IterExpr)
	}
}

func TestParser_IfElse(t *testing.T) {
	input := `{* if condition: *}
yes
{* else: *}
no
{* endif *}`

	tmpl, err := ParseString(input, "test.sql")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tmpl.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(tmpl.Nodes))
	}

	ifBlock, ok := tmpl.Nodes[0].(*IfBlock)
	if !ok {
		t.Fatalf("expected IfBlock, got %T", tmpl.Nodes[0])
	}

	if ifBlock.Condition != "condition" {
		t.Errorf("expected condition 'condition', got %q", ifBlock.Condition)
	}

	// Check if body
	if len(ifBlock.Body) != 1 {
		t.Fatalf("expected 1 if body node, got %d", len(ifBlock.Body))
	}

	// Check else body
	if ifBlock.Else == nil {
		t.Fatal("expected else body")
	}
	if len(ifBlock.Else) != 1 {
		t.Fatalf("expected 1 else body node, got %d", len(ifBlock.Else))
	}
}

func TestParser_IfElif(t *testing.T) {
	input := `{* if a: *}
A
{* elif b: *}
B
{* elif c: *}
C
{* endif *}`

	tmpl, err := ParseString(input, "test.sql")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ifBlock, ok := tmpl.Nodes[0].(*IfBlock)
	if !ok {
		t.Fatalf("expected IfBlock, got %T", tmpl.Nodes[0])
	}

	if ifBlock.Condition != "a" {
		t.Errorf("expected condition 'a', got %q", ifBlock.Condition)
	}

	if len(ifBlock.ElseIfs) != 2 {
		t.Fatalf("expected 2 elif branches, got %d", len(ifBlock.ElseIfs))
	}

	if ifBlock.ElseIfs[0].Condition != "b" {
		t.Errorf("elif[0]: expected condition 'b', got %q", ifBlock.ElseIfs[0].Condition)
	}
	if ifBlock.ElseIfs[1].Condition != "c" {
		t.Errorf("elif[1]: expected condition 'c', got %q", ifBlock.ElseIfs[1].Condition)
	}
}

func TestParser_IfElifElse(t *testing.T) {
	input := `{* if a: *}
A
{* elif b: *}
B
{* else: *}
C
{* endif *}`

	tmpl, err := ParseString(input, "test.sql")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ifBlock, ok := tmpl.Nodes[0].(*IfBlock)
	if !ok {
		t.Fatalf("expected IfBlock, got %T", tmpl.Nodes[0])
	}

	if ifBlock.Condition != "a" {
		t.Errorf("expected condition 'a', got %q", ifBlock.Condition)
	}

	if len(ifBlock.ElseIfs) != 1 {
		t.Fatalf("expected 1 elif branch, got %d", len(ifBlock.ElseIfs))
	}

	if ifBlock.Else == nil {
		t.Fatal("expected else body")
	}
}

func TestParser_NestedBlocks(t *testing.T) {
	input := `{* for x in items: *}
{* if x > 0: *}
{{ x }}
{* endif *}
{* endfor *}`

	tmpl, err := ParseString(input, "test.sql")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	forBlock, ok := tmpl.Nodes[0].(*ForBlock)
	if !ok {
		t.Fatalf("expected ForBlock, got %T", tmpl.Nodes[0])
	}

	// Find the IfBlock in the for body
	var foundIf bool
	for _, node := range forBlock.Body {
		if _, ok := node.(*IfBlock); ok {
			foundIf = true
			break
		}
	}
	if !foundIf {
		t.Error("expected nested IfBlock in ForBlock body")
	}
}

func TestParser_UnmatchedFor(t *testing.T) {
	input := `{* for x in items: *}
{{ x }}`

	_, err := ParseString(input, "test.sql")
	if err == nil {
		t.Fatal("expected error for unmatched for")
	}

	unmatchedErr, ok := err.(*UnmatchedBlockError)
	if !ok {
		t.Fatalf("expected UnmatchedBlockError, got %T: %v", err, err)
	}
	if unmatchedErr.BlockKind != StmtFor {
		t.Errorf("expected StmtFor, got %s", unmatchedErr.BlockKind)
	}
}

func TestParser_UnmatchedEndFor(t *testing.T) {
	input := `{{ x }}
{* endfor *}`

	_, err := ParseString(input, "test.sql")
	if err == nil {
		t.Fatal("expected error for unmatched endfor")
	}
}

func TestParser_UnmatchedIf(t *testing.T) {
	input := `{* if condition: *}
yes`

	_, err := ParseString(input, "test.sql")
	if err == nil {
		t.Fatal("expected error for unmatched if")
	}
}

func TestParser_UnmatchedElse(t *testing.T) {
	input := `yes
{* else: *}
no`

	_, err := ParseString(input, "test.sql")
	if err == nil {
		t.Fatal("expected error for unmatched else")
	}
}

func TestParser_InvalidStatement(t *testing.T) {
	input := `{* while true: *}`

	_, err := ParseString(input, "test.sql")
	if err == nil {
		t.Fatal("expected error for invalid statement")
	}
}

func TestParser_ForWithoutColon(t *testing.T) {
	// Both with and without colon should work
	inputs := []string{
		`{* for x in items: *}{{ x }}{* endfor *}`,
		`{* for x in items *}{{ x }}{* endfor *}`,
	}

	for _, input := range inputs {
		tmpl, err := ParseString(input, "test.sql")
		if err != nil {
			t.Errorf("input %q: unexpected error: %v", input, err)
			continue
		}

		forBlock, ok := tmpl.Nodes[0].(*ForBlock)
		if !ok {
			t.Errorf("input %q: expected ForBlock, got %T", input, tmpl.Nodes[0])
			continue
		}

		if forBlock.VarName != "x" {
			t.Errorf("input %q: expected var 'x', got %q", input, forBlock.VarName)
		}
	}
}

func TestParser_ComplexExpression(t *testing.T) {
	input := `{{ target.schema + "." + this.name }}`

	tmpl, err := ParseString(input, "test.sql")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expr, ok := tmpl.Nodes[0].(*ExprNode)
	if !ok {
		t.Fatalf("expected ExprNode, got %T", tmpl.Nodes[0])
	}

	expected := `target.schema + "." + this.name`
	if expr.Expr != expected {
		t.Errorf("expected %q, got %q", expected, expr.Expr)
	}
}
