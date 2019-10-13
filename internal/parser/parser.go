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
		valueList astScopeValueNode
		config    *astCmdConfig
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
		ctx.pushLexFn(lexExpectNewline)
		ctx.setLexFn(lexExport)
		name = expectTokenType(p, tokenID, "Expecting tokenID").Value()
		switch {
		// '=' | ':=''
		//
		case tryPeekType(p, tokenEquals):
			p.Next()
			if valueList, ok = tryMatchAssignmentValue(ctx, p); ok {
				ctx.ast.addScopeNode(&astScopeVarAssignment{name: name, value: valueList})
				ctx.ast.addScopeNode(&astScopeExportList{names: []string{name}})
			} else {
				panic("expecting assignment values")
			}
		// '?='
		//
		case tryPeekType(p, tokenQMarkEquals):
			p.Next()
			if valueList, ok = tryMatchAssignmentValue(ctx, p); ok {
				ctx.ast.addScopeNode(&astScopeVarQAssignment{name: name, value: valueList})
				ctx.ast.addScopeNode(&astScopeExportList{names: []string{name}})
			} else {
				panic("expecting assignment values")
			}
		// ','
		//
		default:
			exportList := &astScopeExportList{}
			exportList.names = append(exportList.names, name)
			for tryPeekType(p, tokenComma) {
				p.Next()
				name = expectTokenType(p, tokenID, "Expecting tokenID").Value()
				exportList.names = append(exportList.names, name)
			}
			ctx.ast.addScopeNode(exportList)
		}
		expectTokenType(p, tokenNewline, "expecting end of line")
		p.Clear()
		return parseMain
	}
	// Doc Line
	//
	if tryPeekType(p, tokenConfigDescLine) {
		line := p.Next()
		config = &astCmdConfig{}
		config.desc = append(config.desc, &astScopeValueRunes{value: line.Value()})
		p.Clear()
		tryMatchCmd(ctx, p, config)
		return parseMain
	}
	// Doc Block
	//
	if config, ok = tryMatchDocBlock(ctx, p); ok {
		// Command?
		//
		tryMatchCmd(ctx, p, config)
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
			ctx.ast.addScopeNode(&astScopeAttrAssignment{name: name, value: valueList})
			return parseMain
		}
		panic("expecting assignment value")
	}
	// Variable Assignment
	//
	if name, ok = tryMatchAssignmentStart(p); ok {
		ctx.pushLexFn(ctx.l.fn)
		if valueList, ok = tryMatchAssignmentValue(ctx, p); ok {
			ctx.ast.addScopeNode(&astScopeVarAssignment{name: name, value: valueList})
			return parseMain
		}
		panic("expecting assignment value")
	}
	// Variable QAssignment
	//
	if name, ok = tryMatchQAssignmentStart(p); ok {
		ctx.pushLexFn(ctx.l.fn)
		if valueList, ok = tryMatchAssignmentValue(ctx, p); ok {
			ctx.ast.addScopeNode(&astScopeVarQAssignment{name: name, value: valueList})
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
func tryMatchCmd(ctx *parseContext, p *parser.Parser, config *astCmdConfig) bool {
	var (
		name  string
		shell string
		ok    bool
	)
	if config == nil {
		config = &astCmdConfig{}
	}

	if name, shell, ok = tryMatchCmdHeaderWithShell(p); !ok {
		return false
	}
	ctx.pushLexFn(ctx.l.fn)
	if tryPeekType(p, tokenColon) {
		p.Next()
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

// tryMatchDocBlock
//
func tryMatchDocBlock(ctx *parseContext, p *parser.Parser) (*astCmdConfig, bool) {
	var config *astCmdConfig = nil
	if tryPeekType(p, tokenHashLine) {
		p.Next()
		config = &astCmdConfig{}
		ctx.pushLexFn(ctx.l.fn)
		ctx.setLexFn(lexDocBlockDesc)
		// Desc
		//
		for !tryPeekType(p, tokenConfigDescEnd) {
			line := expectDocNQString(ctx, p)
			config.desc = append(config.desc, line)
		}
		expectTokenType(p, tokenConfigDescEnd, "Expecting tokenConfigDescEnd")
		// Attributes
		//
		ctx.setLexFn(lexDocBlockAttr)
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
			case tokenConfigUsage:
				p.Next()
				ctx.pushLexFn(ctx.l.fn)
				ctx.setLexFn(lexCmdUsage)
				usage := expectDocNQString(ctx, p)
				config.usages = append(config.usages, usage)
				p.Clear()
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
				opt.desc = expectDocNQString(ctx, p)
				config.opts = append(config.opts, opt)
			case tokenConfigExport:
				p.Next()
				ctx.pushLexFn(ctx.l.fn)
				ctx.pushLexFn(lexExpectNewline)
				ctx.setLexFn(lexExport)
				name := expectTokenType(p, tokenID, "Expecting tokenID").Value()
				switch {
				// '=' | ':=''
				//
				case tryPeekType(p, tokenEquals):
					p.Next()
					if valueList, ok := tryMatchAssignmentValue(ctx, p); ok {
						config.vars = append(config.vars, &astScopeVarAssignment{name: name, value: valueList})
						config.exports = append(config.exports, newAstScopeExportList1(name))
					} else {
						panic("expecting assignment value")
					}
				// '?='
				//
				case tryPeekType(p, tokenQMarkEquals):
					p.Next()
					if valueList, ok := tryMatchAssignmentValue(ctx, p); ok {
						config.vars = append(config.vars, &astScopeVarQAssignment{name: name, value: valueList})
						config.exports = append(config.exports, newAstScopeExportList1(name))
					} else {
						panic("expecting assignment value")
					}
				// ','
				//
				default:
					exportList := &astScopeExportList{}
					exportList.names = append(exportList.names, name)
					for tryPeekType(p, tokenComma) {
						p.Next()
						name = expectTokenType(p, tokenID, "Expecting tokenID").Value()
						exportList.names = append(exportList.names, name)
					}
					config.exports = append(config.exports, exportList)
				}
				expectTokenType(p, tokenNewline, "expecting end of line")
				p.Clear()
			default:
				panic(fmt.Sprintf("%d:%d: Expecting cmd config statement", t.Line(), t.Column()))
			}
		}
		expectTokenType(p, tokenConfigEnd, "Expecting tokenConfigEnd")
		p.Clear()
	}
	return config, config != nil
}

// expectDocNQString - Expects lexer.fn == lexDocBlockNQString BEFORE calling.
//
func expectDocNQString(ctx *parseContext, p *parser.Parser) astScopeValueNode {
	values := make([]astScopeValueNode, 0)

	for p.CanPeek(1) && !tryPeekType(p, tokenNewline) {
		switch p.PeekType(1) {
		// Character run
		//
		case tokenRunes:
			values = append(values, &astScopeValueRunes{value: p.Next().Value()})
		// Escape char
		//
		case tokenEscapeSequence:
			values = append(values, &astScopeValueEsc{seq: p.Next().Value()})
		// Var Ref
		//
		case tokenVarRefStart:
			p.Next()
			values = append(values, expectVarRef(ctx, p))
		// End of line
		//
		default:
			panic(parseError(p, "Expecting printable character or newline"))
		}
	}
	// If not eof, expect newline
	//
	if p.CanPeek(1) {
		expectTokenType(p, tokenNewline, "expecting tokeNewline ('\n')")
	}
	p.Clear()
	return &astScopeValueNodeList{values: values}
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
func tryMatchAssignmentValue(ctx *parseContext, p *parser.Parser) (*astScopeValueNodeList, bool) {
	ctx.setLexFn(lexAssignmentValue)
	if !p.CanPeek(1) {
		return newAstScopeValueNodeList([]astScopeValueNode{}), false
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
		return newAstScopeValueNodeList1(expectVarRef(ctx, p)), true
	case tokenSubCmdStart:
		p.Next()
		return newAstScopeValueNodeList1(expectSubCmd(ctx, p)), true
	case tokenDollar:
		t := p.Next()
		panic(fmt.Sprintf("%d:%d: $ must be followed by '{' or '('", t.Line(), t.Column()))
	default:
		value := expectTokenType(p, tokenRunes, "expecting tokenRunes").Value()
		return newAstScopeValueNodeList1(&astScopeValueRunes{value: value}), true
	}
}

// expectVarRef
//
func expectVarRef(ctx *parseContext, p *parser.Parser) *astScopeValueVar {
	ctx.setLexFn(lexVarRef)
	// Dollar
	//
	expectTokenType(p, tokenDollar, "expecting tokenDollar ('$')")
	// Open Brace
	//
	expectTokenType(p, tokenLBrace, "expecting tokenLBrace ('{')")
	// Value
	//
	name := expectTokenType(p, tokenRunes, "expecting tokenRunes").Value()
	// Close Brace
	//
	expectTokenType(p, tokenRBrace, "expecting tokenRBrace ('}')")

	return &astScopeValueVar{name: name}
}

// expectSubCmd
//
func expectSubCmd(ctx *parseContext, p *parser.Parser) *astScopeValueShell {
	ctx.setLexFn(lexSubCmd)
	// Dollar
	//
	expectTokenType(p, tokenDollar, "expecting tokenDollar ('$')")
	// Open Paren
	//
	expectTokenType(p, tokenLParen, "expecting tokenLParen ('(')")

	// Values
	//
	values := make([]astScopeValueNode, 0)

	for p.CanPeek(1) {
		switch p.PeekType(1) {
		// Character run
		//
		case tokenRunes:
			values = append(values, &astScopeValueRunes{value: p.Next().Value()})
		// Escape char
		//
		case tokenEscapeSequence:
			values = append(values, &astScopeValueEsc{seq: p.Next().Value()})
		// Close Paren
		//
		default:
			expectTokenType(p, tokenRParen, "expecting tokenRParen (')')")
			return &astScopeValueShell{cmd: newAstScopeValueNodeList(values)}
		}
	}
	panic("expecting tokenRParen (')')")
}

// expectSQString
//
func expectSQString(ctx *parseContext, p *parser.Parser) *astScopeValueNodeList {
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

	return newAstScopeValueNodeList1(&astScopeValueRunes{value: value})
}

// expectDQString
//
func expectDQString(ctx *parseContext, p *parser.Parser) *astScopeValueNodeList {
	ctx.setLexFn(lexDQString)
	// Open Quote
	//
	expectTokenType(p, tokenDQuote, "expecting tokenDoubleQuote ('\"')")

	// Values
	//
	values := make([]astScopeValueNode, 0)

	for p.CanPeek(1) {
		switch p.PeekType(1) {
		// Character run
		//
		case tokenRunes:
			values = append(values, &astScopeValueRunes{value: p.Next().Value()})
		// Escape char
		//
		case tokenEscapeSequence:
			values = append(values, &astScopeValueEsc{seq: p.Next().Value()})
		case tokenVarRefStart:
			p.Next()
			values = append(values, expectVarRef(ctx, p))

		case tokenSubCmdStart:
			p.Next()
			values = append(values, expectSubCmd(ctx, p))
		case tokenDollar:
			p.Next()
			values = append(values, &astScopeValueRunes{value: "$"})
		// Close quote
		//
		default:
			expectTokenType(p, tokenDQuote, "expecting tokenDoubleQuote ('\"')")
			return newAstScopeValueNodeList(values)
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
	// Colon or Brace - If not present,then error, but don't consume if present
	//
	if !tryPeekType(p, tokenColon) && !tryPeekType(p, tokenLBrace) {
		panic(parseError(p, "Expecting tokenColon (':') or tokenLBrace ('{')"))
	}
	p.Clear()
	return name, shell, true
}

// expectCmdScript
//
func expectCmdScript(ctx *parseContext, p *parser.Parser) []string {
	// Open Brace
	//
	ctx.setLexFn(lexMain)
	usingBraces := tryPeekType(p, tokenLBrace)
	if usingBraces {
		expectTokenType(p, tokenLBrace, "expecting tokenLBrace ('{')")
		ctx.setLexFn(lexCmdScriptAfterLBrace)
	} else {
		expectTokenType(p, tokenNewline, "expecting tokenNewline ('\\n')")
		ctx.setLexFn(lexCmdScriptMaybeLBrace)
		usingBraces = tryPeekType(p, tokenLBrace)
		if usingBraces {
			expectTokenType(p, tokenLBrace, "expecting tokenLBrace ('{')")
		}
	}
	// Script Body
	//
	var scriptText []string
	for p.CanPeek(1) && p.PeekType(1) == tokenScriptLine {
		scriptText = append(scriptText, p.Next().Value())
	}
	if usingBraces || p.CanPeek(1) {
		expectTokenType(p, tokenScriptEnd, "expecting tokenSciptEnd")
	}
	// Close Brace
	//
	if usingBraces {
		ctx.setLexFn(lexCmdScriptMaybeRBrace)
		expectTokenType(p, tokenRBrace, "expecting tokenRBrace ('}')")
	}
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
