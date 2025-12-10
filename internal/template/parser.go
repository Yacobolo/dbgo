package template

import (
	"regexp"
	"strings"
)

// Parser builds an AST from a token stream.
type Parser struct {
	tokens []Token
	pos    int
	file   string
}

// NewParser creates a new parser for the given tokens.
func NewParser(tokens []Token, file string) *Parser {
	return &Parser{
		tokens: tokens,
		pos:    0,
		file:   file,
	}
}

// Parse parses the tokens into a Template AST.
func (p *Parser) Parse() (*Template, error) {
	nodes, err := p.parseNodes(nil)
	if err != nil {
		return nil, err
	}
	return &Template{Nodes: nodes, File: p.file}, nil
}

// parseNodes parses nodes until EOF or a closing statement (endfor/endif/else/elif).
// The stopOn parameter specifies which statement kinds should stop parsing.
func (p *Parser) parseNodes(stopOn []StmtKind) ([]Node, error) {
	var nodes []Node

	for p.pos < len(p.tokens) {
		tok := p.current()

		switch tok.Type {
		case TokenEOF:
			return nodes, nil

		case TokenText:
			nodes = append(nodes, &TextNode{
				nodeBase: nodeBase{pos: tok.Pos},
				Text:     tok.Value,
			})
			p.advance()

		case TokenExpr:
			nodes = append(nodes, &ExprNode{
				nodeBase: nodeBase{pos: tok.Pos},
				Expr:     tok.Value,
			})
			p.advance()

		case TokenStmt:
			// First peek at the statement kind to check if we should stop
			kind := peekStmtKind(tok.Value)

			// Check if this is a stopping statement BEFORE consuming
			if containsKind(stopOn, kind) {
				// Don't consume; let caller handle it
				return nodes, nil
			}

			stmt, err := p.parseStmt(tok)
			if err != nil {
				return nil, err
			}

			// Handle block statements
			switch stmt.Kind {
			case StmtFor:
				block, err := p.parseForBlock(stmt)
				if err != nil {
					return nil, err
				}
				nodes = append(nodes, block)

			case StmtIf:
				block, err := p.parseIfBlock(stmt)
				if err != nil {
					return nil, err
				}
				nodes = append(nodes, block)

			case StmtEndFor, StmtEndIf, StmtElse, StmtElif:
				// Unexpected closing statement
				return nil, NewUnmatchedBlockError(tok.Pos, stmt.Kind)

			default:
				return nil, NewParseErrorf(tok.Pos, "unexpected statement: %s", tok.Value)
			}

		default:
			return nil, NewParseErrorf(tok.Pos, "unexpected token: %s", tok.Type)
		}
	}

	return nodes, nil
}

// Regex patterns for parsing statements
var (
	forPattern  = regexp.MustCompile(`^for\s+(\w+)\s+in\s+(.+?)\s*:?\s*$`)
	ifPattern   = regexp.MustCompile(`^if\s+(.+?)\s*:?\s*$`)
	elifPattern = regexp.MustCompile(`^elif\s+(.+?)\s*:?\s*$`)
)

// peekStmtKind determines the statement kind without advancing the parser.
func peekStmtKind(value string) StmtKind {
	value = strings.TrimSpace(value)

	switch value {
	case "endfor":
		return StmtEndFor
	case "endif":
		return StmtEndIf
	case "else", "else:":
		return StmtElse
	}

	if forPattern.MatchString(value) {
		return StmtFor
	}
	if ifPattern.MatchString(value) {
		return StmtIf
	}
	if elifPattern.MatchString(value) {
		return StmtElif
	}

	return StmtUnknown
}

// parseStmt parses a statement token into a StmtNode.
func (p *Parser) parseStmt(tok Token) (*StmtNode, error) {
	value := strings.TrimSpace(tok.Value)

	// Check for simple keywords first
	switch value {
	case "endfor":
		p.advance()
		return &StmtNode{
			nodeBase: nodeBase{pos: tok.Pos},
			Kind:     StmtEndFor,
		}, nil
	case "endif":
		p.advance()
		return &StmtNode{
			nodeBase: nodeBase{pos: tok.Pos},
			Kind:     StmtEndIf,
		}, nil
	case "else", "else:":
		p.advance()
		return &StmtNode{
			nodeBase: nodeBase{pos: tok.Pos},
			Kind:     StmtElse,
		}, nil
	}

	// Check for 'for' loop
	if match := forPattern.FindStringSubmatch(value); match != nil {
		p.advance()
		return &StmtNode{
			nodeBase: nodeBase{pos: tok.Pos},
			Kind:     StmtFor,
			VarName:  match[1],
			Expr:     match[2],
		}, nil
	}

	// Check for 'if' conditional
	if match := ifPattern.FindStringSubmatch(value); match != nil {
		p.advance()
		return &StmtNode{
			nodeBase: nodeBase{pos: tok.Pos},
			Kind:     StmtIf,
			Expr:     match[1],
		}, nil
	}

	// Check for 'elif' conditional
	if match := elifPattern.FindStringSubmatch(value); match != nil {
		p.advance()
		return &StmtNode{
			nodeBase: nodeBase{pos: tok.Pos},
			Kind:     StmtElif,
			Expr:     match[1],
		}, nil
	}

	return nil, NewParseErrorf(tok.Pos, "invalid statement syntax: %s", value)
}

// parseForBlock parses a complete for loop including body and endfor.
func (p *Parser) parseForBlock(stmt *StmtNode) (*ForBlock, error) {
	// Parse body until endfor
	body, err := p.parseNodes([]StmtKind{StmtEndFor})
	if err != nil {
		return nil, err
	}

	// Expect endfor
	if p.pos >= len(p.tokens) || p.current().Type == TokenEOF {
		return nil, NewUnmatchedBlockError(stmt.Pos(), StmtFor)
	}

	endTok := p.current()
	if endTok.Type != TokenStmt {
		return nil, NewUnmatchedBlockError(stmt.Pos(), StmtFor)
	}

	endStmt, err := p.parseStmt(endTok)
	if err != nil {
		return nil, err
	}
	if endStmt.Kind != StmtEndFor {
		return nil, NewUnmatchedBlockError(stmt.Pos(), StmtFor)
	}

	return &ForBlock{
		nodeBase: nodeBase{pos: stmt.Pos()},
		VarName:  stmt.VarName,
		IterExpr: stmt.Expr,
		Body:     body,
	}, nil
}

// parseIfBlock parses a complete if/elif/else conditional including body and endif.
func (p *Parser) parseIfBlock(stmt *StmtNode) (*IfBlock, error) {
	block := &IfBlock{
		nodeBase:  nodeBase{pos: stmt.Pos()},
		Condition: stmt.Expr,
	}

	// Parse body until elif/else/endif
	body, err := p.parseNodes([]StmtKind{StmtElif, StmtElse, StmtEndIf})
	if err != nil {
		return nil, err
	}
	block.Body = body

	// Process elif/else/endif chain
	for p.pos < len(p.tokens) && p.current().Type != TokenEOF {
		tok := p.current()
		if tok.Type != TokenStmt {
			return nil, NewUnmatchedBlockError(stmt.Pos(), StmtIf)
		}

		nextStmt, err := p.parseStmt(tok)
		if err != nil {
			return nil, err
		}

		switch nextStmt.Kind {
		case StmtEndIf:
			return block, nil

		case StmtElif:
			// Parse elif body
			elifBody, err := p.parseNodes([]StmtKind{StmtElif, StmtElse, StmtEndIf})
			if err != nil {
				return nil, err
			}
			block.ElseIfs = append(block.ElseIfs, Branch{
				Condition: nextStmt.Expr,
				Body:      elifBody,
				pos:       nextStmt.Pos(),
			})

		case StmtElse:
			// Parse else body
			elseBody, err := p.parseNodes([]StmtKind{StmtEndIf})
			if err != nil {
				return nil, err
			}
			block.Else = elseBody

			// Must be followed by endif
			if p.pos >= len(p.tokens) || p.current().Type == TokenEOF {
				return nil, NewUnmatchedBlockError(stmt.Pos(), StmtIf)
			}
			endTok := p.current()
			endStmt, err := p.parseStmt(endTok)
			if err != nil {
				return nil, err
			}
			if endStmt.Kind != StmtEndIf {
				return nil, NewUnmatchedBlockError(stmt.Pos(), StmtIf)
			}
			return block, nil

		default:
			return nil, NewUnmatchedBlockError(stmt.Pos(), StmtIf)
		}
	}

	return nil, NewUnmatchedBlockError(stmt.Pos(), StmtIf)
}

// Helper methods

func (p *Parser) current() Token {
	if p.pos >= len(p.tokens) {
		return Token{Type: TokenEOF}
	}
	return p.tokens[p.pos]
}

func (p *Parser) advance() {
	p.pos++
}

func containsKind(kinds []StmtKind, k StmtKind) bool {
	for _, kind := range kinds {
		if kind == k {
			return true
		}
	}
	return false
}

// ParseString is a convenience function to parse a template string.
func ParseString(input, file string) (*Template, error) {
	lexer := NewLexer(input, file)
	tokens, err := lexer.Tokenize()
	if err != nil {
		return nil, err
	}

	parser := NewParser(tokens, file)
	return parser.Parse()
}
