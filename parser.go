package main

import (
	"io"

	"github.com/tekwizely/go-parsing/lexer"
	"github.com/tekwizely/go-parsing/lexer/token"
	"github.com/tekwizely/go-parsing/parser"
)

// parseFn
//
type parseFn func(*parseContext, *parser.Parser) parseFn

// parseContext
//
type parseContext struct {
	l   *lexContext
	ast *ast
	fn  parseFn
}

// parse
//
func (ctx *parseContext) parse(p *parser.Parser) parser.Fn {
	fn := ctx.fn
	// EOF?
	//
	if fn == nil {
		return nil
	}
	ctx.fn = fn(ctx, p)
	return ctx.parse
}

// SetLexFn
//
func (ctx *parseContext) SetLexFn(fn lexer.Fn) {
	ctx.l.SetFn(fn)
}

// AST
//
func (ctx *parseContext) AST() *ast {
	return ctx.ast
}

// parse
//
func parse(l *lexContext) *ast {
	ctx := &parseContext{l: l, ast: newAST(), fn: parseMain}
	_, err := parser.Parse(l.tokens, ctx.parse).Next() // No emits
	if err != nil && err != io.EOF {
		panic(err)
	}
	return ctx.AST()
}

// parseMain
//
func parseMain(ctx *parseContext, p *parser.Parser) parseFn {
	if id, ok := tryMatchScriptHeader(p); ok {
		ctx.SetLexFn(lexScript)
		if scriptText, ok := tryMatchScript(p); ok {
			// Normalize the script
			//
			scriptText = normalizeCmdText(scriptText)
			ctx.AST().commands[id] = &script{text: scriptText}
			ctx.SetLexFn(lexMain)
			return parseMain
		}
		panic("Error parsing script for command '" + id + "'")
	}
	panic("Expecting command header")
}

// tryMatchScriptHeader matches [ ID ':' ]
//
func tryMatchScriptHeader(p *parser.Parser) (string, bool) {
	if p.CanPeek(2) && p.PeekType(1) == TokenID && p.PeekType(2) == tokenColon {
		tID := p.Next()
		id := tID.Value()
		expectTokenType(p, tokenColon, "Expecting tokenColon (':')")
		p.Clear()
		return id, true
	}
	return "", false
}

// tryMatchScript
//
func tryMatchScript(p *parser.Parser) ([]string, bool) {
	var scriptText []string
	for p.CanPeek(1) && p.PeekType(1) == tokenScriptLine {
		scriptText = append(scriptText, p.Next().Value())
	}
	// If not EOF, then expecdt tokenEndScript
	//
	if p.CanPeek(1) {
		expectTokenType(p, tokenEndScript, "Expecting tokenEndScript")
	}
	p.Clear()
	return scriptText, len(scriptText) > 0
}

func expectTokenType(p *parser.Parser, typ token.Type, msg string) {
	if p.CanPeek(1) && p.PeekType(1) == typ {
		p.Next()
	} else {
		panic(msg)
	}
}
