package template

import (
	"strings"

	starctx "github.com/user/dbgo/internal/starlark"
	"go.starlark.net/starlark"
)

// Renderer executes a parsed template with a Starlark context.
type Renderer struct {
	ctx    *starctx.ExecutionContext
	locals starlark.StringDict // Local variables (e.g., loop variables)
}

// NewRenderer creates a new renderer with the given execution context.
func NewRenderer(ctx *starctx.ExecutionContext) *Renderer {
	return &Renderer{
		ctx:    ctx,
		locals: nil,
	}
}

// Render executes the template and returns the rendered SQL.
func (r *Renderer) Render(tmpl *Template) (string, error) {
	var buf strings.Builder

	if err := r.renderNodes(tmpl.Nodes, &buf, tmpl.File); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// renderNodes renders a slice of nodes into the buffer.
func (r *Renderer) renderNodes(nodes []Node, buf *strings.Builder, file string) error {
	for _, node := range nodes {
		if err := r.renderNode(node, buf, file); err != nil {
			return err
		}
	}
	return nil
}

// renderNode renders a single node into the buffer.
func (r *Renderer) renderNode(node Node, buf *strings.Builder, file string) error {
	switch n := node.(type) {
	case *TextNode:
		buf.WriteString(n.Text)

	case *ExprNode:
		result, err := r.ctx.EvalExprStringWithLocals(n.Expr, file, n.Pos().Line, r.locals)
		if err != nil {
			return WrapRenderError(n.Pos(), "expression evaluation failed", err)
		}
		buf.WriteString(result)

	case *ForBlock:
		if err := r.renderForBlock(n, buf, file); err != nil {
			return err
		}

	case *IfBlock:
		if err := r.renderIfBlock(n, buf, file); err != nil {
			return err
		}

	default:
		return NewRenderErrorf(node.Pos(), "unknown node type: %T", node)
	}

	return nil
}

// renderForBlock renders a for loop block.
func (r *Renderer) renderForBlock(block *ForBlock, buf *strings.Builder, file string) error {
	// Evaluate the iterator expression
	iterVal, err := r.ctx.EvalExprWithLocals(block.IterExpr, file, block.Pos().Line, r.locals)
	if err != nil {
		return WrapRenderError(block.Pos(), "for loop iterator evaluation failed", err)
	}

	// Get an iterable
	iter := starlark.Iterate(iterVal)
	if iter == nil {
		return NewRenderErrorf(block.Pos(), "for loop: cannot iterate over %s", iterVal.Type())
	}
	defer iter.Done()

	// Iterate and render body for each element
	var elem starlark.Value
	for iter.Next(&elem) {
		// Create locals with the loop variable added
		loopLocals := r.withLocal(block.VarName, elem)

		// Render body with loop context
		loopRenderer := &Renderer{
			ctx:    r.ctx,
			locals: loopLocals,
		}
		if err := loopRenderer.renderNodes(block.Body, buf, file); err != nil {
			return err
		}
	}

	return nil
}

// withLocal creates a new locals dict with an additional variable.
func (r *Renderer) withLocal(name string, value starlark.Value) starlark.StringDict {
	newLocals := make(starlark.StringDict, len(r.locals)+1)
	for k, v := range r.locals {
		newLocals[k] = v
	}
	newLocals[name] = value
	return newLocals
}

// renderIfBlock renders an if/elif/else conditional block.
func (r *Renderer) renderIfBlock(block *IfBlock, buf *strings.Builder, file string) error {
	// Evaluate the main condition
	condVal, err := r.ctx.EvalExprWithLocals(block.Condition, file, block.Pos().Line, r.locals)
	if err != nil {
		return WrapRenderError(block.Pos(), "if condition evaluation failed", err)
	}

	if condVal.Truth() {
		return r.renderNodes(block.Body, buf, file)
	}

	// Check elif branches
	for _, elif := range block.ElseIfs {
		condVal, err := r.ctx.EvalExprWithLocals(elif.Condition, file, elif.pos.Line, r.locals)
		if err != nil {
			return WrapRenderError(elif.pos, "elif condition evaluation failed", err)
		}
		if condVal.Truth() {
			return r.renderNodes(elif.Body, buf, file)
		}
	}

	// Execute else branch if present
	if block.Else != nil {
		return r.renderNodes(block.Else, buf, file)
	}

	return nil
}

// RenderString is a convenience function to render a template string.
func RenderString(input, file string, ctx *starctx.ExecutionContext) (string, error) {
	tmpl, err := ParseString(input, file)
	if err != nil {
		return "", err
	}

	renderer := NewRenderer(ctx)
	return renderer.Render(tmpl)
}
