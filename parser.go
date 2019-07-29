package main

import (
	"container/list"
	"fmt"
	"io"
	"strings"

	"github.com/tekwizely/go-parsing/lexer/token"
	"github.com/tekwizely/go-parsing/parser"
)

// parseFn
//
type parseFn func(*parseContext, *parser.Parser) parseFn

// parseContext
//
type parseContext struct {
	l       *lexContext
	ast     *ast
	fn      parseFn
	fnStack *list.List
}

// parse
//
func (ctx *parseContext) parse(p *parser.Parser) parser.Fn {
	fn := ctx.fn
	// EOF?
	//
	if fn == nil {
		if ctx.fnStack.Len() == 0 {
			return nil
		}
		fn = ctx.fnStack.Remove(ctx.fnStack.Front()).(parseFn)
	}
	// assert(fn != nil)
	ctx.fn = fn(ctx, p)
	return ctx.parse
}

// setLexFn
//
func (ctx *parseContext) setLexFn(fn lexFn) {
	ctx.l.fn = fn
}

// pushLexFn
//
func (ctx *parseContext) pushLexFn(fn lexFn) {
	ctx.l.pushFn(fn)
}

// // pushFn
// //
// func (c *parseContext) pushFn(fn parseFn) {
// 	c.fnStack.PushBack(fn)
// }

// parse
//
func parse(l *lexContext) *ast {
	ctx := &parseContext{
		l:       l,
		ast:     newAST(),
		fn:      parseMain,
		fnStack: list.New(),
	}
	_, err := parser.Parse(l.tokens, ctx.parse).Next() // No emits
	if err != nil && err != io.EOF {
		panic(err)
	}
	return ctx.ast
}

// parseMain
//
func parseMain(ctx *parseContext, p *parser.Parser) parseFn {
	// DotAssignment
	//
	if name, ok := tryMatchDotAssignmentStart(p); ok {
		ctx.pushLexFn(ctx.l.fn)
		if valueList, ok := tryMatchAssignmentValue(ctx, p); ok {
			// Let's go ahead and normalize this now
			//
			name = strings.ToUpper(name)
			ctx.ast.add(&astAttrAssignment{name: name, value: valueList})
			return parseMain
		}
		panic("expecting assignment value")
	}
	// Assignment
	//
	if name, ok := tryMatchAssignmentStart(p); ok {
		ctx.pushLexFn(ctx.l.fn)
		if valueList, ok := tryMatchAssignmentValue(ctx, p); ok {
			ctx.ast.add(&astEnvAssignment{name: name, value: valueList})
			return parseMain
		}
		panic("expecting assignment value")
	}
	// Script Header
	//
	if name, shell, ok := tryMatchScriptHeader(p); ok {
		ctx.pushLexFn(ctx.l.fn)
		if body, ok := tryMatchScriptBody(ctx, p); ok {
			// Normalize the script
			//
			body = normalizeCmdText(body)
			ctx.ast.add(&astCmd{name: name, shell: shell, script: body})
			ctx.setLexFn(lexMain)
			return parseMain
		}
		panic("Error parsing script for command '" + name + "'")
	}
	panic("Expecting command header")
}

// tryMatchDotAssignmentStart
//
func tryMatchDotAssignmentStart(p *parser.Parser) (string, bool) {
	if p.CanPeek(2) &&
		p.PeekType(1) == tokenDotID &&
		p.PeekType(2) == tokenEquals {
		// Grab the name
		//
		name := p.Next().Value()
		// Discard the '='
		//
		expectTokenType(p, tokenEquals, "Expecting tokenEquals ('=')")
		p.Clear()
		return name, true
	}
	return "", false
}

// tryMatchAssignmentStart
//
func tryMatchAssignmentStart(p *parser.Parser) (string, bool) {
	if p.CanPeek(2) &&
		p.PeekType(1) == tokenID &&
		p.PeekType(2) == tokenEquals {
		// Grab the name
		//
		name := p.Next().Value()
		// Discard the '='
		//
		expectTokenType(p, tokenEquals, "Expecting tokenEquals ('=')")
		p.Clear()
		return name, true
	}
	return "", false
}

// tryMatchAssignmentValue
//
func tryMatchAssignmentValue(ctx *parseContext, p *parser.Parser) ([]astValueElement, bool) {
	ctx.setLexFn(lexAssignmentValue)
	if !p.CanPeek(1) {
		return []astValueElement{}, false
	}
	switch p.PeekType(1) {
	case tokenSQStringStart:
		p.Next()
		return expectSQString(ctx, p), true
	case tokenDQStringStart:
		p.Next()
		return expectDQString(ctx, p), true
	case tokenVarRefStart:
		p.Next()
		return expectVarRef(ctx, p), true
	case tokenSubCmdStart:
		p.Next()
		return expectSubCmd(ctx, p), true
	case tokenDollar:
		t := p.Next()
		panic(fmt.Sprintf("%d:%d: $ must be followed by '{' or '('", t.Line(), t.Column()))
	default:
		value := expectTokenType(p, tokenRunes, "expecting tokenRunes").Value()
		return []astValueElement{&astValueRunes{runes: value}}, true
	}
}

// func tryMatchDollarString(ctx *parseContext, p *parser.Parser) ([]astValueElement, bool) {
// 	ctx.setLexFn(lexDollarString)
// 	if !p.CanPeek(1) {
// 		return []astValueElement{}, false
// 	}
// 	switch p.PeekType(1) {
// 	case tokenVarRefStart:
// 		p.Next()
// 		return expectVarRef(ctx, p), true
// 	case tokenSubCmdStart:
// 		p.Next()
// 		return expectSubCmd(ctx, p), true
// 	case tokenDollar:
// 		t := p.Next()
// 		panic(fmt.Sprintf("%d:%d: $ must be followed by '{' or '('", t.Line(), t.Column()))
// 	}
// 	return []astValueElement{}, false
// }

func expectVarRef(ctx *parseContext, p *parser.Parser) []astValueElement {
	ctx.setLexFn(lexVarRef)
	// Dollar
	//
	expectTokenType(p, tokenDollar, "expecting tokenDollar ('$')")
	// Open Brace
	//
	expectTokenType(p, tokenLBrace, "expecting tokenLBrace ('{')")
	// Value
	//
	value := expectTokenType(p, tokenRunes, "expecting tokenRunes").Value()
	// Close Brace
	//
	expectTokenType(p, tokenRBrace, "expecting tokenRBrace ('}')")

	return []astValueElement{&astValueVar{varName: value}}
}

func expectSubCmd(ctx *parseContext, p *parser.Parser) []astValueElement {
	ctx.setLexFn(lexSubCmd)
	// Dollar
	//
	expectTokenType(p, tokenDollar, "expecting tokenDollar ('$')")
	// Open Paren
	//
	expectTokenType(p, tokenLParen, "expecting tokenLParen ('(')")

	// Value
	//
	value := make([]astValueElement, 0)

	for p.CanPeek(1) {
		switch p.PeekType(1) {
		// Character run
		//
		case tokenRunes:
			value = append(value, &astValueRunes{runes: p.Next().Value()})
		// Escape char
		//
		case tokenEscapeSequence:
			value = append(value, &astValueEsc{seq: p.Next().Value()})
		// Close Paren
		//
		default:
			expectTokenType(p, tokenRParen, "expecting tokenRParen (')')")
			return []astValueElement{&astValueShell{cmd: value}}
		}
	}
	panic("expecting tokenRParen (')')")
}

func expectSQString(ctx *parseContext, p *parser.Parser) []astValueElement {
	ctx.setLexFn(lexSQString)
	// Open Quote
	//
	expectTokenType(p, tokenSQuote, "expecting tokenSingleQuote (\"'\")")
	// Value
	//
	value := expectTokenType(p, tokenRunes, "expecting tokenRunes").Value()
	// Close Quote
	//
	expectTokenType(p, tokenSQuote, "expecting tokenSingleQuote (\"'\")")

	return []astValueElement{&astValueRunes{runes: value}}
}

func expectDQString(ctx *parseContext, p *parser.Parser) []astValueElement {
	ctx.setLexFn(lexDQString)
	// Open Quote
	//
	expectTokenType(p, tokenDQuote, "expecting tokenDoubleQuote ('\"')")

	// Value
	//
	value := make([]astValueElement, 0)

	for p.CanPeek(1) {
		switch p.PeekType(1) {
		// Character run
		//
		case tokenRunes:
			value = append(value, &astValueRunes{runes: p.Next().Value()})
		// Escape char
		//
		case tokenEscapeSequence:
			value = append(value, &astValueEsc{seq: p.Next().Value()})
		case tokenVarRefStart:
			p.Next()
			value = append(value, expectVarRef(ctx, p)[0])
		case tokenSubCmdStart:
			p.Next()
			value = append(value, expectSubCmd(ctx, p)[0])
		case tokenDollar:
			p.Next()
			value = append(value, &astValueRunes{runes: "$"})
		// Close quote
		//
		default:
			expectTokenType(p, tokenDQuote, "expecting tokenDoubleQuote ('\"')")
			return value
		}
	}
	panic("expecting tokenDoubleQuote ('\"')")
}

// tryMatchScriptHeader matches [ ID ( '(' ID ')' )? ':' ]
//
func tryMatchScriptHeader(p *parser.Parser) (string, string, bool) {
	if p.CanPeek(2) &&
		p.PeekType(1) == tokenID &&
		(p.PeekType(2) == tokenColon || p.PeekType(2) == tokenLParen) {
		// Grab the name
		//
		name := p.Next().Value()
		// Is Shell present?
		//
		var shell string
		if _, ok := tryTokenType(p, tokenLParen); ok {
			shell = expectTokenType(p, tokenID, "Expecting tokenID").Value()
			expectTokenType(p, tokenRParen, "Expecting tokenRParen (')')")
		}
		expectTokenType(p, tokenColon, "Expecting tokenColon (':')")
		p.Clear()
		return name, shell, true
	}
	return "", "", false
}

// tryMatchScriptBody
//
func tryMatchScriptBody(ctx *parseContext, p *parser.Parser) ([]string, bool) {
	ctx.setLexFn(lexScriptBody)
	var scriptText []string
	for p.CanPeek(1) && p.PeekType(1) == tokenScriptLine {
		scriptText = append(scriptText, p.Next().Value())
	}
	// If not EOF, then expect tokenScriptEnd
	//
	if p.CanPeek(1) {
		expectTokenType(p, tokenScriptEnd, "Expecting tokenScriptEnd")
	}
	p.Clear()
	return scriptText, len(scriptText) > 0
}

func tryTokenType(p *parser.Parser, typ token.Type) (token.Token, bool) {
	if p.CanPeek(1) && p.PeekType(1) == typ {
		return p.Next(), true
	}
	return nil, false
}

func expectTokenType(p *parser.Parser, typ token.Type, msg string) token.Token {
	if p.CanPeek(1) {
		if t := p.Peek(1); t.Type() != typ {
			panic(fmt.Sprintf("%d.%d: %s", t.Line(), t.Column(), msg))
		}
		return p.Next()
	}
	panic(fmt.Sprintf("<eof>: %s", msg))
}
