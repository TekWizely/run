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
		traceFn("Popped parser function", fn)
	}
	// assert(fn != nil)
	traceFn("Calling parser function", fn)
	ctx.fn = fn(ctx, p)
	return ctx.parse
}

// setLexFn
//
func (ctx *parseContext) setLexFn(fn lexFn) {
	ctx.l.fn = fn
	traceFn("Set lexer function", fn)
}

// pushLexFn
//
func (ctx *parseContext) pushLexFn(fn lexFn) {
	ctx.l.pushFn(fn)
}

// pushFn
//
func (ctx *parseContext) pushFn(fn parseFn) {
	ctx.fnStack.PushBack(fn)
	traceFn("Pushed parser function", fn)
}

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
	var (
		name      string
		valueList *astValue
		ok        bool
	)
	// Newline
	//
	if tryPeekType(p, tokenNewline) {
		p.Next()
		p.Clear()
		return parseMain
	}
	// Export
	//
	if tryPeekType(p, tokenExport) {
		p.Next()
		ctx.pushLexFn(ctx.l.fn)
		ctx.setLexFn(lexExport)
		name = expectTokenType(p, tokenID, "Expecting tokenID").Value()
		switch {
		// '=' | ':=''
		//
		case tryPeekType(p, tokenEquals):
			p.Next()
			if valueList, ok = tryMatchAssignmentValue(ctx, p); ok {
				ctx.ast.add(&astVarAssignment{name: name, value: valueList})
				ctx.ast.add(&astExport{names: []string{name}})
			} else {
				panic("expecting assignment value")
			}
		// '?='
		//
		case tryPeekType(p, tokenQMarkEquals):
			p.Next()
			if valueList, ok = tryMatchAssignmentValue(ctx, p); ok {
				ctx.ast.add(&astVarQAssignment{name: name, value: valueList})
				ctx.ast.add(&astExport{names: []string{name}})
			} else {
				panic("expecting assignment value")
			}
		// ','
		//
		default:
			export := &astExport{}
			export.names = append(export.names, name)
			for tryPeekType(p, tokenComma) {
				p.Next()
				name = expectTokenType(p, tokenID, "Expecting tokenID").Value()
				export.names = append(export.names, name)
			}
			ctx.ast.add(export)
		}
		p.Clear()
		return parseMain
	}
	// Doc Block
	//
	if tryPeekType(p, tokenHashLine) {
		p.Next()
		var docBlock []*astCmdValue
		ctx.pushLexFn(ctx.l.fn)
		ctx.setLexFn(lexDocBlock)
		for !tryPeekType(p, tokenDocBlockEnd) {
			expectTokenType(p, tokenHash, "Expecting tokenHash ('#')")
			line := expectTokenType(p, tokenRunes, "Expecting tokenRunes")
			docBlock = append(docBlock, newAstCmdAstValue1(newAstValueRunes(line.Value())))
			p.Clear()
		}
		p.Next()
		p.Clear()
		// Command?
		//
		tryMatchCmd(ctx, p, docBlock)
		return parseMain
	}
	// DotAssignment
	//
	if name, ok = tryMatchDotAssignmentStart(p); ok {
		ctx.pushLexFn(ctx.l.fn)
		if valueList, ok = tryMatchAssignmentValue(ctx, p); ok {
			// Let's go ahead and normalize this now
			//
			name = strings.ToUpper(name)
			ctx.ast.add(&astAttrAssignment{name: name, value: valueList})
			return parseMain
		}
		panic("expecting assignment value")
	}
	// Variable Assignment
	//
	if name, ok = tryMatchAssignmentStart(p); ok {
		ctx.pushLexFn(ctx.l.fn)
		if valueList, ok = tryMatchAssignmentValue(ctx, p); ok {
			ctx.ast.add(&astVarAssignment{name: name, value: valueList})
			return parseMain
		}
		panic("expecting assignment value")
	}
	// Variable QAssignment
	//
	if name, ok = tryMatchQAssignmentStart(p); ok {
		ctx.pushLexFn(ctx.l.fn)
		if valueList, ok = tryMatchAssignmentValue(ctx, p); ok {
			ctx.ast.add(&astVarQAssignment{name: name, value: valueList})
			return parseMain
		}
		panic("expecting assignment value")
	}
	// Command
	//
	if ok = tryMatchCmd(ctx, p, nil); ok {
		return parseMain
	}
	if p.CanPeek(1) {
		t := p.Peek(1)
		panic(fmt.Sprintf("%d:%d: Expecting command header", t.Line(), t.Column()))
	} else {
		panic("Expecting command header")
	}
}

// tryMatchCmd
//
func tryMatchCmd(ctx *parseContext, p *parser.Parser, docBlock []*astCmdValue) bool {
	var (
		name  string
		shell string
		ok    bool
	)
	if name, shell, ok = tryMatchCmdHeaderWithShell(p); !ok {
		return false
	}
	ctx.pushLexFn(ctx.l.fn)
	// Config
	//
	var config *astCmdConfig
	if tryPeekType(p, tokenColon) {
		p.Next() // Consume ':'
		config = expectCmdConfig(ctx, p)
	} else {
		config = &astCmdConfig{}
	}
	if docBlock != nil {
		config.desc = docBlock
	}
	if len(shell) > 0 {
		if len(config.shell) > 0 && shell != config.shell {
			panic(fmt.Sprintf("Shell '%s' defined in cmd header, shell '%s' defined in attributes", shell, config.shell))
		}
		config.shell = shell
	}
	// Script
	//
	script := expectCmdScript(ctx, p)
	// Normalize the script
	//
	script = normalizeCmdScript(script)
	ctx.ast.add(&astCmd{name: name, config: config, script: script})
	return true
}

// tryMatchDotAssignmentStart
//
func tryMatchDotAssignmentStart(p *parser.Parser) (string, bool) {
	if p.CanPeek(2) &&
		p.PeekType(1) == tokenDotID &&
		p.PeekType(2) == tokenEquals {
		name := p.Next().Value()
		expectTokenType(p, tokenEquals, "Expecting tokenEquals ('=' | ':=')")
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
		name := p.Next().Value()
		expectTokenType(p, tokenEquals, "Expecting tokenEquals ('=' | ':=')")
		p.Clear()
		return name, true
	}
	return "", false
}

// tryMatchQAssignmentStart
//
func tryMatchQAssignmentStart(p *parser.Parser) (string, bool) {
	if p.CanPeek(2) &&
		p.PeekType(1) == tokenID &&
		p.PeekType(2) == tokenQMarkEquals {
		name := p.Next().Value()
		expectTokenType(p, tokenQMarkEquals, "Expecting tokenQMarkEquals ('?=')")
		p.Clear()
		return name, true
	}
	return "", false
}

// tryMatchAssignmentValue
//
func tryMatchAssignmentValue(ctx *parseContext, p *parser.Parser) (*astValue, bool) {
	ctx.setLexFn(lexAssignmentValue)
	if !p.CanPeek(1) {
		return newAstValue([]astValueElement{}), false
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
		return newAstValue([]astValueElement{&astValueRunes{value: value}}), true
	}
}

// tryMatchDollarString
//
func tryMatchDollarString(ctx *parseContext, p *parser.Parser) (*astValue, bool) {
	ctx.setLexFn(lexDollarString)
	if !p.CanPeek(1) {
		return newAstValue([]astValueElement{}), false
	}
	switch p.PeekType(1) {
	case tokenVarRefStart:
		p.Next()
		return expectVarRef(ctx, p), true
	case tokenSubCmdStart:
		p.Next()
		return expectSubCmd(ctx, p), true
	case tokenDollar:
		t := p.Next()
		panic(fmt.Sprintf("%d:%d: $ must be followed by '{' or '('", t.Line(), t.Column()))
	}
	return newAstValue([]astValueElement{}), false
}

// expectVarRef
//
func expectVarRef(ctx *parseContext, p *parser.Parser) *astValue {
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

	return newAstValue([]astValueElement{&astValueVar{name: value}})
}

// expectSubCmd
//
func expectSubCmd(ctx *parseContext, p *parser.Parser) *astValue {
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
			value = append(value, &astValueRunes{value: p.Next().Value()})
		// Escape char
		//
		case tokenEscapeSequence:
			value = append(value, &astValueEsc{seq: p.Next().Value()})
		// Close Paren
		//
		default:
			expectTokenType(p, tokenRParen, "expecting tokenRParen (')')")
			return newAstValue([]astValueElement{&astValueShell{cmd: newAstValue(value)}})
		}
	}
	panic("expecting tokenRParen (')')")
}

// expectSQString
//
func expectSQString(ctx *parseContext, p *parser.Parser) *astValue {
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

	return newAstValue([]astValueElement{&astValueRunes{value: value}})
}

// expectDQString
//
func expectDQString(ctx *parseContext, p *parser.Parser) *astValue {
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
			value = append(value, &astValueRunes{value: p.Next().Value()})
		// Escape char
		//
		case tokenEscapeSequence:
			value = append(value, &astValueEsc{seq: p.Next().Value()})
		case tokenVarRefStart:
			p.Next()
			value = append(value, expectVarRef(ctx, p))
		case tokenSubCmdStart:
			p.Next()
			value = append(value, expectSubCmd(ctx, p))
		case tokenDollar:
			p.Next()
			value = append(value, &astValueRunes{value: "$"})
		// Close quote
		//
		default:
			expectTokenType(p, tokenDQuote, "expecting tokenDoubleQuote ('\"')")
			return newAstValue(value)
		}
	}
	panic("expecting tokenDoubleQuote ('\"')")
}

// tryMatchCmdHeaderWithShell matches [ [ 'CMD' ] ID ( '(' ID ')' )? ( ':' | '{' ) ]
//
func tryMatchCmdHeaderWithShell(p *parser.Parser) (string, string, bool) {
	expectCommand := tryPeekType(p, tokenCommand)
	if expectCommand {
		expectTokenType(p, tokenCommand, "Expecting tokenCommand")
	} else {
		expectCommand =
			tryPeekTypes(p, tokenID, tokenColon) ||
				tryPeekTypes(p, tokenID, tokenLParen) ||
				tryPeekTypes(p, tokenID, tokenLBrace)
	}
	if !expectCommand {
		return "", "", false
	}
	// Name
	//
	name := expectTokenType(p, tokenID, "Expecting tokenId").Value()
	// Shell
	//
	shell := ""
	if tryPeekType(p, tokenLParen) {
		expectTokenType(p, tokenLParen, "Expecting tokenLParen ('(')")
		shell = expectTokenType(p, tokenID, "Expecting shell name").Value()
		expectTokenType(p, tokenRParen, "Expecting tokenRParen (')')")
	}
	// Colon or Brace - If not present,then error, but don't consume if pesent
	//
	if !tryPeekType(p, tokenColon) && !tryPeekType(p, tokenLBrace) {
		panic(parseError(p, "Expecting tokenColon (':') or tokenLBrace ('{')"))
	}
	p.Clear()
	return name, shell, true
}

// expectCmdConfig
//
func expectCmdConfig(ctx *parseContext, p *parser.Parser) *astCmdConfig {
	config := &astCmdConfig{}
	// // Desc is always first, if present
	// //
	// ctx.setLexFn(lexCmdDesc)
	// for !tryPeekType(p, tokenConfigDescEnd) {
	// 	expectTokenType(p, tokenHash, "Expecting tokenHash ('#')")
	// 	desc := expectTokenType(p, tokenRunes, "Expecting tokenRunes")
	// 	config.desc = append(config.desc, newAstCmdAstValue1(newAstValueRunes(desc.Value())))
	// 	p.Clear()
	// }
	// expectTokenType(p, tokenConfigDescEnd, "Expecting tokenConfigDescEnd")

	ctx.setLexFn(lexCmdAttr)
	for !tryPeekType(p, tokenConfigEnd) {
		t := p.Peek(1)
		switch t.Type() {
		case tokenConfigShell:
			p.Next()
			if config.shell != "" {
				panic(fmt.Sprintf("%d:%d: SHELL already defined", t.Line(), t.Column()))
			}
			ctx.pushLexFn(ctx.l.fn)
			ctx.setLexFn(lexCmdShell)
			shell := expectTokenType(p, tokenID, "Expecting tokenID")
			config.shell = shell.Value()
		// case tokenConfigDesc:
		// 	p.Next()
		// 	if len(config.desc) > 0 {
		// 		panic(fmt.Sprintf("%d:%d: DESC already defined", t.Line(), t.Column()))
		// 	}
		// 	ctx.pushLexFn(ctx.l.fn)
		// 	ctx.setLexFn(lexCmdDesc)
		// 	desc := expectTokenType(p, tokenRunes, "Expecting tokenRunes")
		// 	config.desc = desc.Value()
		case tokenConfigUsage:
			p.Next()
			ctx.pushLexFn(ctx.l.fn)
			ctx.setLexFn(lexCmdUsage)
			usage := expectTokenType(p, tokenRunes, "Expecting tokenRunes")
			config.usages = append(config.usages, newAstCmdAstValue1(newAstValueRunes(usage.Value())))
		case tokenConfigOpt:
			p.Next()
			opt := &astCmdOpt{}
			ctx.pushLexFn(ctx.l.fn)
			ctx.setLexFn(lexCmdOpt)
			opt.name = expectTokenType(p, tokenConfigOptName, "Expecting tokenConfigOptName").Value()
			if tryPeekType(p, tokenConfigOptShort) {
				opt.short = []rune(p.Next().Value())[0]
			}
			if tryPeekType(p, tokenConfigOptLong) {
				opt.long = p.Next().Value()
			}
			if tryPeekType(p, tokenConfigOptValue) {
				opt.value = p.Next().Value()
			}
			if tryPeekType(p, tokenDQStringStart) {
				p.Next()
				opt.desc = expectDQString(ctx, p)
			}
			config.opts = append(config.opts, opt)
		case tokenConfigExport:
			p.Next()
			ctx.pushLexFn(ctx.l.fn)
			ctx.setLexFn(lexExport)
			name := expectTokenType(p, tokenID, "Expecting tokenID")
			switch {
			// '=' | ':=''
			//
			case tryPeekType(p, tokenEquals):
				p.Next()
				// ctx.pushLexFn(ctx.l.fn)
				if valueList, ok := tryMatchAssignmentValue(ctx, p); ok {
					config.env = append(config.env, &astCmdEnvAssignment{name: name.Value(), value: valueList})
				} else {
					panic("expecting assignment value")
				}
			// '?='
			//
			case tryPeekType(p, tokenQMarkEquals):
				p.Next()
				// ctx.pushLexFn(ctx.l.fn)
				if valueList, ok := tryMatchAssignmentValue(ctx, p); ok {
					config.env = append(config.env, &astCmdEnvQAssignment{name: name.Value(), value: valueList})
				} else {
					panic("expecting assignment value")
				}
			// ','
			//
			default:
				export := &astCmdExport{}
				export.names = append(export.names, name.Value())
				for tryPeekType(p, tokenComma) {
					p.Next()
					name = expectTokenType(p, tokenID, "Expecting tokenID")
					export.names = append(export.names, name.Value())
				}
				config.exports = append(config.exports, export)
			}
		default:
			panic(fmt.Sprintf("%d:%d: Expecting cmd config statement", t.Line(), t.Column()))
		}
	}
	expectTokenType(p, tokenConfigEnd, "Expecting tokenConfigEnd")
	p.Clear()
	return config
}

// expectCmdScript
//
func expectCmdScript(ctx *parseContext, p *parser.Parser) []string {
	ctx.setLexFn(lexMain)
	expectTokenType(p, tokenLBrace, "expecting tokenLBrace ('{')")
	ctx.setLexFn(lexCmdScript)
	// Script Body
	//
	var scriptText []string
	for p.CanPeek(1) && p.PeekType(1) == tokenScriptLine {
		scriptText = append(scriptText, p.Next().Value())
	}
	// Close Brace
	//
	expectTokenType(p, tokenRBrace, "expecting tokenRBrace ('}')")
	p.Clear()
	return scriptText
}

// tryPeekType
//
func tryPeekType(p *parser.Parser, typ token.Type) bool {
	return p.CanPeek(1) && p.PeekType(1) == typ
}

// tryPeekTypes
//
func tryPeekTypes(p *parser.Parser, types ...token.Type) bool {
	for i, typ := range types {
		if !p.CanPeek(i+1) || p.PeekType(i+1) != typ {
			return false
		}
	}
	return true
}

// expectTokenType
//
func expectTokenType(p *parser.Parser, typ token.Type, msg string) token.Token {
	if p.CanPeek(1) && p.Peek(1).Type() == typ {
		return p.Next()
	}
	panic(parseError(p, msg))
}

// parseError
//
func parseError(p *parser.Parser, msg string) error {
	// If a token is available, use it for line/column
	//
	if p.CanPeek(1) {
		t := p.Peek(1)
		return fmt.Errorf("%d.%d: %s", t.Line(), t.Column(), msg)
	}
	return fmt.Errorf("<eof>: %s", msg)
}
