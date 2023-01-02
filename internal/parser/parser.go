package parser

import (
	"container/list"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/tekwizely/go-parsing/lexer/token"
	"github.com/tekwizely/go-parsing/parser"

	"github.com/tekwizely/run/internal/ast"
	"github.com/tekwizely/run/internal/config"
	"github.com/tekwizely/run/internal/lexer"
	"github.com/tekwizely/run/internal/runfile"
)

// ParseBytes attempts to parse the specified byte array.
//
func ParseBytes(runfile []byte) *ast.Ast {
	return Parse(lexer.Lex(runfile))
}

// parseFn
//
type parseFn func(*parseContext, *parser.Parser) parseFn

// parseContext
//
type parseContext struct {
	l       *lexer.LexContext
	ast     *ast.Ast
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
		config.TraceFn("Popped parser function", fn)
	}
	// assert(fn != nil)
	config.TraceFn("Calling parser function", fn)
	ctx.fn = fn(ctx, p)
	return ctx.parse
}

// setLexFn
//
func (ctx *parseContext) setLexFn(fn lexer.LexFn) {
	ctx.l.Fn = fn
	config.TraceFn("Set lexer function", fn)
}

// pushLexFn
//
func (ctx *parseContext) pushLexFn(fn lexer.LexFn) {
	ctx.l.PushFn(fn)
}

// pushFn
//
func (ctx *parseContext) pushFn(fn parseFn) {
	ctx.fnStack.PushBack(fn)
	config.TraceFn("Pushed parser function", fn)
}

// Parse delegates incoming parser calls to the configured fn
//
func Parse(l *lexer.LexContext) *ast.Ast {
	ctx := &parseContext{
		l:       l,
		ast:     ast.NewAST(),
		fn:      parseMain,
		fnStack: list.New(),
	}
	_, err := parser.Parse(l.Tokens, ctx.parse).Next() // No emits
	if err != nil && err != io.EOF {
		panic(err)
	}
	return ctx.ast
}

// parseMain
//
func parseMain(ctx *parseContext, p *parser.Parser) parseFn {
	var (
		varName   string
		attrName  string
		valueList ast.ScopeValueNode
		cmdConfig *ast.CmdConfig
		ok        bool
	)
	// Newline
	//
	if tryPeekType(p, lexer.TokenNewline) {
		p.Next()
		p.Clear()
		return parseMain
	}
	// Export
	//
	if tryPeekType(p, lexer.TokenExport) {
		p.Next()
		ctx.pushLexFn(ctx.l.Fn)
		ctx.pushLexFn(lexer.LexExpectNewline)
		ctx.setLexFn(lexer.LexExport)
		commaMode := false
		for hasNext := true; hasNext; {
			hasNext = false
			if tryPeekType(p, lexer.TokenID) {
				varName = p.Next().Value()
				switch {
				// '=' | ':=''
				//
				case !commaMode && tryPeekType(p, lexer.TokenEquals):
					p.Next()
					valueList = expectAssignmentValue(ctx, p)
					ctx.ast.AddScopeNode(&ast.ScopeVarAssignment{Name: varName, Value: valueList})
					ctx.ast.AddScopeNode(ast.NewVarExport(varName))
				// '?='
				//
				case !commaMode && tryPeekType(p, lexer.TokenQMarkEquals):
					p.Next()
					valueList = expectAssignmentValue(ctx, p)
					ctx.ast.AddScopeNode(&ast.ScopeVarQAssignment{Name: varName, Value: valueList})
					ctx.ast.AddScopeNode(ast.NewVarExport(varName))
				// Export existing variable
				//
				default:
					ctx.ast.AddScopeNode(ast.NewVarExport(varName))
				}
			} else {
				attrName = expectTokenType(p, lexer.TokenDotID, "expecting TokenID or TokenDotID").Value()
				// Let's go ahead and normalize this now
				//
				attrName = strings.ToUpper(attrName)
				// 'AS'
				//
				if tryPeekType(p, lexer.TokenAs) {
					p.Next()
					varName = expectTokenType(p, lexer.TokenID, "expecting TokenID").Value()
				} else {
					// Variable name based on munged attribute name
					//
					varName = attrName[1:]                          // Strip leading '.'
					varName = strings.ReplaceAll(varName, ".", "_") // s/\./_/
				}
				ctx.ast.AddScopeNode(ast.NewAttrExport(attrName, varName))
			}
			// ','
			//
			if tryPeekTypes(p, lexer.TokenComma) {
				p.Next()
				commaMode = true
				hasNext = true
			}
		}
		expectTokenType(p, lexer.TokenNewline, "expecting end of line")
		p.Clear()
		return parseMain
	}
	// Assert
	//
	if tryPeekType(p, lexer.TokenAssert) {
		t := p.Next()
		ctx.pushLexFn(ctx.l.Fn)
		ctx.l.PushFn(lexer.LexExpectNewline)
		ctx.setLexFn(lexer.LexAssert)
		assert := &ast.ScopeAssert{}
		assert.Runfile = config.CurrentRunfile
		assert.Line = t.Line()
		assert.Test = expectTestString(ctx, p)
		assert.Message = expectAssertMessage(ctx, p)
		ctx.ast.AddScopeNode(assert)
		expectTokenType(p, lexer.TokenNewline, "expecting end of line")
		p.Clear()
		return parseMain
	}
	// Include
	//
	if tryPeekType(p, lexer.TokenInclude) {
		p.Next()
		ctx.pushLexFn(ctx.l.Fn)
		ctx.setLexFn(lexer.LexMaybeBangOrQMark)
		t := p.Next()
		var (
			missingSingleOk   = t.Type() == lexer.TokenQMark
			missingMatchersOk = t.Type() != lexer.TokenBang
		)
		valueList = expectAssignmentValue(ctx, p)
		ctx.ast.Add(&ast.ScopeInclude{FilePattern: valueList, MissingSingleOk: missingSingleOk, MissingMatchersOk: missingMatchersOk})
		expectTokenType(p, lexer.TokenNewline, "expecting end of line")
		p.Clear()
		return parseMain
	}
	// Include.Env
	//
	if tryPeekType(p, lexer.TokenIncludeEnv) {
		p.Next()
		ctx.pushLexFn(ctx.l.Fn)
		ctx.setLexFn(lexer.LexMaybeBangOrQMark)
		t := p.Next()
		var (
			missingSingleOk   = t.Type() != lexer.TokenBang
			missingMatchersOk = t.Type() != lexer.TokenBang
		)
		valueList = expectAssignmentValue(ctx, p)
		ctx.ast.Add(&ast.ScopeIncludeEnv{FilePattern: valueList, MissingSingleOk: missingSingleOk, MissingMatchersOk: missingMatchersOk})
		expectTokenType(p, lexer.TokenNewline, "expecting end of line")
		p.Clear()
		return parseMain
	}
	// Doc Line
	//
	if tryPeekType(p, lexer.TokenConfigDescLineStart) {
		p.Next()
		ctx.pushLexFn(ctx.l.Fn)
		ctx.setLexFn(lexer.LexDocBlockNQString)
		line := expectDocNQString(ctx, p)
		cmdConfig = &ast.CmdConfig{}
		cmdConfig.Desc = append(cmdConfig.Desc, line)
		p.Clear()
		tryMatchCmd(ctx, p, cmdConfig)
		return parseMain
	}
	// Doc Block
	//
	if cmdConfig, ok = tryMatchDocBlock(ctx, p); ok {
		// Command?
		//
		tryMatchCmd(ctx, p, cmdConfig)
		return parseMain
	}
	// DotAssignment
	//
	if attrName, ok = tryMatchDotAssignmentStart(p); ok {
		// Let's go ahead and normalize this now
		//
		attrName = strings.ToUpper(attrName)
		ctx.pushLexFn(ctx.l.Fn)
		valueList = expectAssignmentValue(ctx, p)
		ctx.ast.AddScopeNode(&ast.ScopeAttrAssignment{Name: attrName, Value: valueList})
		return parseMain
	}
	// Variable Assignment
	//
	if varName, ok = tryMatchAssignmentStart(p); ok {
		ctx.pushLexFn(ctx.l.Fn)
		valueList = expectAssignmentValue(ctx, p)
		ctx.ast.AddScopeNode(&ast.ScopeVarAssignment{Name: varName, Value: valueList})
		return parseMain
	}
	// Variable QAssignment
	//
	if varName, ok = tryMatchQAssignmentStart(p); ok {
		ctx.pushLexFn(ctx.l.Fn)
		valueList = expectAssignmentValue(ctx, p)
		ctx.ast.AddScopeNode(&ast.ScopeVarQAssignment{Name: varName, Value: valueList})
		return parseMain
	}
	// Command
	//
	if ok = tryMatchCmd(ctx, p, nil); ok {
		return parseMain
	}
	panic(parseError(p, "expecting runfile statement"))
}

// tryMatchCmd
//
func tryMatchCmd(ctx *parseContext, p *parser.Parser, cmdConfig *ast.CmdConfig) bool {
	var (
		flags config.CmdFlags
		name  string
		shell string
		ok    bool
		line  int
	)
	if flags, name, shell, line, ok = tryMatchCmdHeaderWithShell(ctx, p); !ok {
		return false
	}
	if cmdConfig == nil {
		cmdConfig = &ast.CmdConfig{}
	}
	ctx.pushLexFn(ctx.l.Fn)
	if tryPeekType(p, lexer.TokenColon) {
		p.Next()
	}
	if len(shell) > 0 {
		if len(cmdConfig.Shell) > 0 && shell != cmdConfig.Shell {
			panic(parseError(p, fmt.Sprintf("Shell '%s' defined in cmd header, shell '%s' defined in attributes", shell, cmdConfig.Shell)))
		}
		cmdConfig.Shell = shell
	}
	// Script
	//
	script := expectCmdScript(ctx, p)
	// Normalize the script
	//
	script = runfile.NormalizeCmdScript(script)
	// Should not be empty
	//
	if len(script) == 0 {
		panic(parseError(p, "command '"+name+"' contains an empty script."))
	}
	ctx.ast.Add(&ast.Cmd{
		Flags:   flags,
		Name:    name,
		Config:  cmdConfig,
		Script:  script,
		Runfile: config.CurrentRunfile,
		Line:    line,
	})
	return true
}

// tryMatchDocBlock
//
func tryMatchDocBlock(ctx *parseContext, p *parser.Parser) (*ast.CmdConfig, bool) {
	var cmdConfig *ast.CmdConfig = nil
	if tryPeekType(p, lexer.TokenHashLine) {
		p.Next()
		cmdConfig = &ast.CmdConfig{}
		ctx.pushLexFn(ctx.l.Fn)
		ctx.setLexFn(lexer.LexDocBlockDesc)
		// Desc
		//
		for !tryPeekType(p, lexer.TokenConfigDescEnd) {
			line := expectDocNQString(ctx, p)
			cmdConfig.Desc = append(cmdConfig.Desc, line)
		}
		expectTokenType(p, lexer.TokenConfigDescEnd, "expecting TokenConfigDescEnd")
		// Attributes
		//
		ctx.setLexFn(lexer.LexDocBlockAttr)
		for !tryPeekType(p, lexer.TokenConfigEnd) {
			t := p.Peek(1)
			switch t.Type() {
			case lexer.TokenConfigShell:
				p.Next()
				if cmdConfig.Shell != "" {
					panic(fmt.Sprintf("%d:%d: SHELL already defined", t.Line(), t.Column()))
				}
				ctx.pushLexFn(ctx.l.Fn)
				ctx.setLexFn(lexer.LexCmdConfigShell)
				shell := expectTokenType(p, lexer.TokenID, "expecting TokenID")
				cmdConfig.Shell = shell.Value()
			case lexer.TokenConfigUsage:
				p.Next()
				ctx.pushLexFn(ctx.l.Fn)
				ctx.setLexFn(lexer.LexCmdConfigUsage)
				usage := expectDocNQString(ctx, p)
				cmdConfig.Usages = append(cmdConfig.Usages, usage)
				p.Clear()
			case lexer.TokenConfigOpt:
				p.Next()
				opt := &ast.CmdOpt{}
				ctx.pushLexFn(ctx.l.Fn)
				ctx.setLexFn(lexer.LexCmdConfigOpt)
				opt.Name = expectTokenType(p, lexer.TokenConfigOptName, "expecting TokenConfigOptName").Value()
				if tryPeekType(p, lexer.TokenBang) {
					opt.Required = true
					p.Next()
				} else if tryPeekType(p, lexer.TokenQMarkEquals) {
					p.Next()
					ctx.pushLexFn(ctx.l.Fn)
					opt.Default = expectAssignmentValue(ctx, p)
				}
				if tryPeekType(p, lexer.TokenConfigOptShort) {
					opt.Short = []rune(p.Next().Value())[0]
				}
				if tryPeekType(p, lexer.TokenConfigOptLong) {
					opt.Long = p.Next().Value()
				}
				if tryPeekType(p, lexer.TokenConfigOptExample) {
					opt.Example = p.Next().Value()
				}
				opt.Desc = expectDocNQString(ctx, p)
				cmdConfig.Opts = append(cmdConfig.Opts, opt)
			case lexer.TokenConfigExport:
				p.Next()
				ctx.pushLexFn(ctx.l.Fn)
				ctx.pushLexFn(lexer.LexExpectNewline)
				ctx.setLexFn(lexer.LexExport)
				commaMode := false
				for hasNext := true; hasNext; {
					hasNext = false
					if tryPeekType(p, lexer.TokenID) {
						varName := expectTokenType(p, lexer.TokenID, "expecting TokenID").Value()
						switch {
						// '=' | ':=''
						//
						case !commaMode && tryPeekType(p, lexer.TokenEquals):
							p.Next()
							valueList := expectAssignmentValue(ctx, p)
							cmdConfig.Vars = append(cmdConfig.Vars, &ast.ScopeVarAssignment{Name: varName, Value: valueList})
							cmdConfig.VarExports = append(cmdConfig.VarExports, ast.NewVarExport(varName))
						// '?='
						//
						case !commaMode && tryPeekType(p, lexer.TokenQMarkEquals):
							p.Next()
							valueList := expectAssignmentValue(ctx, p)
							cmdConfig.Vars = append(cmdConfig.Vars, &ast.ScopeVarQAssignment{Name: varName, Value: valueList})
							cmdConfig.VarExports = append(cmdConfig.VarExports, ast.NewVarExport(varName))
						// Export existing variable
						//
						default:
							cmdConfig.VarExports = append(cmdConfig.VarExports, ast.NewVarExport(varName))
						}
					} else {
						attrName := expectTokenType(p, lexer.TokenDotID, "expecting TokenID or TokenDotID").Value()
						// Let's go ahead and normalize this now
						//
						attrName = strings.ToUpper(attrName)
						// 'AS'
						//
						var varName string
						if tryPeekType(p, lexer.TokenAs) {
							p.Next()
							varName = expectTokenType(p, lexer.TokenID, "expecting TokenID").Value()
						} else {
							// Variable name based on munged attribute name
							//
							varName = attrName[1:]                          // Strip leading '.'
							varName = strings.ReplaceAll(varName, ".", "_") // s/\./_/
						}
						cmdConfig.AttrExports = append(cmdConfig.AttrExports, ast.NewAttrExport(attrName, varName))
					}
					// ','
					//
					if tryPeekTypes(p, lexer.TokenComma) {
						p.Next()
						commaMode = true
						hasNext = true
					}
				}
				expectTokenType(p, lexer.TokenNewline, "expecting end of line")
				p.Clear()
			case lexer.TokenConfigAssert:
				t = p.Next()
				ctx.pushLexFn(ctx.l.Fn)
				ctx.l.PushFn(lexer.LexExpectNewline)
				ctx.setLexFn(lexer.LexAssert)
				assert := &ast.CmdAssert{}
				assert.Runfile = config.CurrentRunfile
				assert.Line = t.Line()
				assert.Test = expectTestString(ctx, p)
				assert.Message = expectAssertMessage(ctx, p)
				cmdConfig.Asserts = append(cmdConfig.Asserts, assert)
				expectTokenType(p, lexer.TokenNewline, "expecting end of line")
				p.Clear()
			case lexer.TokenConfigRunEnv, lexer.TokenConfigRunBefore, lexer.TokenConfigRunAfter:
				t = p.Next()
				ctx.pushLexFn(ctx.l.Fn)
				ctx.setLexFn(lexer.LexExpectCommandName)
				command := p.Next().Value()
				var args []ast.ScopeValueNode
				for {
					ctx.setLexFn(lexer.LexMaybeNewline)
					if tryPeekType(p, lexer.TokenNotNewline) {
						p.Next()
						args = append(args, expectAssignmentValue(ctx, p))
					} else {
						break
					}
				}
				cmdRun := &ast.CmdRun{Command: command, Args: args}
				switch t.Type() {
				case lexer.TokenConfigRunEnv:
					cmdConfig.EnvRuns = append(cmdConfig.EnvRuns, cmdRun)
				case lexer.TokenConfigRunBefore:
					cmdConfig.BeforeRuns = append(cmdConfig.BeforeRuns, cmdRun)
				case lexer.TokenConfigRunAfter:
					cmdConfig.AfterRuns = append(cmdConfig.AfterRuns, cmdRun)
				default:
					// NOTE: Unreachable unless we add a token and forget to implement it
					//
					panic(fmt.Sprintf("%d:%d: Unknown cmd config RUN token: %d", t.Line(), t.Column(), t.Type()))
				}
				expectTokenType(p, lexer.TokenNewline, "expecting end of line")
				p.Clear()
			default:
				panic(fmt.Sprintf("%d:%d: Expecting cmd config statement", t.Line(), t.Column()))
			}
		}
		expectTokenType(p, lexer.TokenConfigEnd, "expecting TokenConfigEnd")
		p.Clear()
	}
	return cmdConfig, cmdConfig != nil
}

// expectDocNQString - Expects lexer.fn == lexDocBlockNQString BEFORE calling.
//
func expectDocNQString(ctx *parseContext, p *parser.Parser) ast.ScopeValueNode {
	values := make([]ast.ScopeValueNode, 0)

	for p.CanPeek(1) && !tryPeekType(p, lexer.TokenNewline) {
		switch p.PeekType(1) {
		// Character run
		//
		case lexer.TokenRunes:
			values = append(values, &ast.ScopeValueRunes{Value: p.Next().Value()})
		// Escape char
		//
		case lexer.TokenEscapeSequence:
			values = append(values, &ast.ScopeValueEsc{Seq: p.Next().Value()})
		// Var Ref
		//
		case lexer.TokenVarRefStart:
			p.Next()
			values = append(values, expectVarRef(ctx, p))
		// End of line
		//
		default:
			panic(parseError(p, "expecting printable character or newline"))
		}
	}
	// If not eof, expect newline
	//
	if p.CanPeek(1) {
		expectTokenType(p, lexer.TokenNewline, "expecting TokenNewline ('\n')")
	}
	p.Clear()
	return &ast.ScopeValueNodeList{Values: values}
}

// tryMatchDotAssignmentStart
//
func tryMatchDotAssignmentStart(p *parser.Parser) (string, bool) {
	if p.CanPeek(2) &&
		p.PeekType(1) == lexer.TokenDotID &&
		p.PeekType(2) == lexer.TokenEquals {
		name := p.Next().Value()
		expectTokenType(p, lexer.TokenEquals, "expecting TokenEquals ('=' | ':=')")
		p.Clear()
		return name, true
	}
	return "", false
}

// tryMatchAssignmentStart
//
func tryMatchAssignmentStart(p *parser.Parser) (string, bool) {
	if p.CanPeek(2) &&
		p.PeekType(1) == lexer.TokenID &&
		p.PeekType(2) == lexer.TokenEquals {
		name := p.Next().Value()
		expectTokenType(p, lexer.TokenEquals, "expecting TokenEquals ('=' | ':=')")
		p.Clear()
		return name, true
	}
	return "", false
}

// tryMatchQAssignmentStart
//
func tryMatchQAssignmentStart(p *parser.Parser) (string, bool) {
	if p.CanPeek(2) &&
		p.PeekType(1) == lexer.TokenID &&
		p.PeekType(2) == lexer.TokenQMarkEquals {
		name := p.Next().Value()
		expectTokenType(p, lexer.TokenQMarkEquals, "expecting TokenQMarkEquals ('?=')")
		p.Clear()
		return name, true
	}
	return "", false
}

// expectAssignmentValue
//
func expectAssignmentValue(ctx *parseContext, p *parser.Parser) *ast.ScopeValueNodeList {
	ctx.setLexFn(lexer.LexAssignmentValue)
	if !p.CanPeek(1) {
		return ast.NewScopeValueNodeList([]ast.ScopeValueNode{})
	}
	switch p.PeekType(1) {
	case lexer.TokenSQStringStart:
		p.Next()
		return expectSQString(ctx, p)
	case lexer.TokenDQStringStart:
		p.Next()
		return expectDQString(ctx, p)
	case lexer.TokenVarRefStart:
		p.Next()
		return ast.NewScopeValueNodeList1(expectVarRef(ctx, p))
	case lexer.TokenSubCmdStart:
		p.Next()
		return ast.NewScopeValueNodeList1(expectSubCmd(ctx, p))
	case lexer.TokenDollar:
		t := p.Next()
		panic(fmt.Sprintf("%d:%d: $ must be followed by '{' or '('", t.Line(), t.Column()))
	default:
		value := expectTokenType(p, lexer.TokenRunes, "expecting TokenRunes").Value()
		return ast.NewScopeValueNodeList1(&ast.ScopeValueRunes{Value: value})
	}
}

// expectAssertMessage
// Expects lexer.fn == LexAssertMessage BEFORE calling.
//
func expectAssertMessage(ctx *parseContext, p *parser.Parser) *ast.ScopeValueNodeList {
	// LexAssertMessage always returns a token
	switch p.PeekType(1) {
	case lexer.TokenSQStringStart:
		p.Next()
		return expectSQString(ctx, p)
	case lexer.TokenDQStringStart:
		p.Next()
		return expectDQString(ctx, p)
	default:
		expectTokenType(p, lexer.TokenEmptyAssertMessage, "expecting quoted assert message or eol")
		return &ast.ScopeValueNodeList{}
	}
}

// expectVarRef
//
func expectVarRef(ctx *parseContext, p *parser.Parser) *ast.ScopeValueVar {
	ctx.setLexFn(lexer.LexVarRef)
	// Dollar
	//
	expectTokenType(p, lexer.TokenDollar, "expecting TokenDollar ('$')")
	// Open Brace
	//
	expectTokenType(p, lexer.TokenLBrace, "expecting TokenLBrace ('{')")
	// Value
	//
	name := expectTokenType(p, lexer.TokenRunes, "expecting TokenRunes").Value()
	// Let's go ahead and normalize this now if it's an attribute
	//
	if strings.HasPrefix(name, ".") {
		name = strings.ToUpper(name)
	}
	// Close Brace
	//
	expectTokenType(p, lexer.TokenRBrace, "expecting TokenRBrace ('}')")

	return &ast.ScopeValueVar{Name: name}
}

// expectSubCmd
//
func expectSubCmd(ctx *parseContext, p *parser.Parser) *ast.ScopeValueShell {
	ctx.setLexFn(lexer.LexSubCmd)
	// Dollar
	//
	expectTokenType(p, lexer.TokenDollar, "expecting TokenDollar ('$')")
	// Open Paren
	//
	expectTokenType(p, lexer.TokenLParen, "expecting TokenLParen ('(')")

	// Values
	//
	values := make([]ast.ScopeValueNode, 0)

	for p.CanPeek(1) {
		switch p.PeekType(1) {
		// Character run
		//
		case lexer.TokenRunes:
			values = append(values, &ast.ScopeValueRunes{Value: p.Next().Value()})
		// Escape char
		//
		case lexer.TokenEscapeSequence:
			values = append(values, &ast.ScopeValueEsc{Seq: p.Next().Value()})
		// Close Paren
		//
		default:
			expectTokenType(p, lexer.TokenRParen, "expecting TokenRParen (')')")
			return &ast.ScopeValueShell{Cmd: ast.NewScopeValueNodeList(values)}
		}
	}
	panic(parseError(p, "expecting tokenRParen (')')"))
}

// expectSQString
//
func expectSQString(ctx *parseContext, p *parser.Parser) *ast.ScopeValueNodeList {
	ctx.setLexFn(lexer.LexSQString)
	// Open Quote
	//
	expectTokenType(p, lexer.TokenSQuote, "expecting TokenSingleQuote (\"'\")")
	// Value
	//
	value := expectTokenType(p, lexer.TokenRunes, "expecting TokenRunes").Value()
	// Close Quote
	//
	expectTokenType(p, lexer.TokenSQuote, "expecting TokenSingleQuote (\"'\")")

	return ast.NewScopeValueNodeList1(&ast.ScopeValueRunes{Value: value})
}

// expectDQString
//
func expectDQString(ctx *parseContext, p *parser.Parser) *ast.ScopeValueNodeList {
	ctx.setLexFn(lexer.LexDQString)
	// Open Quote
	//
	expectTokenType(p, lexer.TokenDQuote, "expecting TokenDoubleQuote ('\"')")

	// Values
	//
	values := make([]ast.ScopeValueNode, 0)

	for p.CanPeek(1) {
		switch p.PeekType(1) {
		// Character run
		//
		case lexer.TokenRunes:
			values = append(values, &ast.ScopeValueRunes{Value: p.Next().Value()})
		// Escape char
		//
		case lexer.TokenEscapeSequence:
			values = append(values, &ast.ScopeValueEsc{Seq: p.Next().Value()})
		case lexer.TokenVarRefStart:
			p.Next()
			values = append(values, expectVarRef(ctx, p))

		case lexer.TokenSubCmdStart:
			p.Next()
			values = append(values, expectSubCmd(ctx, p))
		case lexer.TokenDollar:
			p.Next()
			values = append(values, &ast.ScopeValueRunes{Value: "$"})
		// Close quote
		//
		default:
			expectTokenType(p, lexer.TokenDQuote, "expecting TokenDoubleQuote ('\"')")
			return ast.NewScopeValueNodeList(values)
		}
	}
	panic(parseError(p, "expecting TokenDoubleQuote ('\"')"))
}

// expectTestString - Expects lexer.fn == lexTestString BEFORE calling.
//
func expectTestString(_ *parseContext, p *parser.Parser) ast.ScopeValueNode {
	// Values
	//
	values := make([]ast.ScopeValueNode, 0)
	var (
		endType token.Type
		endErr  string
		fn      func(ast.ScopeValueNode) ast.ScopeValueNode
	)
	switch {
	// [ ... ]
	//
	case tryPeekType(p, lexer.TokenBracketStringStart):
		endType, fn, endErr = lexer.TokenBracketStringEnd, ast.NewScopeBracketString, "expecting TokenBracketStringEnd"
	// [[ ... ]]
	//
	case tryPeekType(p, lexer.TokenDBracketStringStart):
		endType, fn, endErr = lexer.TokenDBracketStringEnd, ast.NewScopeDBracketString, "expecting TokenDBracketStringEnd"
	// ( ... )
	//
	case tryPeekType(p, lexer.TokenParenStringStart):
		endType, fn, endErr = lexer.TokenParenStringEnd, ast.NewScopeParenString, "expecting TokenParenStringEnd"
	// (( ... ))
	//
	case tryPeekType(p, lexer.TokenDParenStringStart):
		endType, fn, endErr = lexer.TokenDParenStringEnd, ast.NewScopeDParenString, "expecting TokenDParenStringEnd"
	default:
		panic(parseError(p, "expecting test string start token"))
	}
	p.Next()

	for p.CanPeek(1) {
		switch p.PeekType(1) {
		// Character run
		//
		case lexer.TokenRunes:
			values = append(values, &ast.ScopeValueRunes{Value: p.Next().Value()})
		// Escape char
		//
		case lexer.TokenEscapeSequence:
			values = append(values, &ast.ScopeValueEsc{Seq: p.Next().Value()})
		// Close string
		//
		default:
			expectTokenType(p, endType, endErr)
			return fn(ast.NewScopeValueNodeList(values))
		}
	}
	panic(parseError(p, "expecting test string end token"))
}

// tryMatchCmdHeaderWithShell matches [ [ 'CMD' ] [ '@' ] DASH_ID ( '(' ID ')' )? ( ':' | '{' ) ]
//
func tryMatchCmdHeaderWithShell(ctx *parseContext, p *parser.Parser) (config.CmdFlags, string, string, int, bool) {
	expectCommand := tryPeekType(p, lexer.TokenCommand)
	if expectCommand {
		expectTokenType(p, lexer.TokenCommand, "expecting TokenCommand")
	} else {
		expectCommand =
			tryPeekType(p, lexer.TokenDashID) ||
				(tryPeekType(p, lexer.TokenDotID) && !strings.ContainsRune(strings.TrimPrefix(p.Peek(1).Value(), "."), '.')) ||
				tryPeekType(p, lexer.TokenCommandDefID) ||
				tryPeekTypes(p, lexer.TokenID, lexer.TokenColon) ||
				tryPeekTypes(p, lexer.TokenID, lexer.TokenLParen) ||
				tryPeekTypes(p, lexer.TokenID, lexer.TokenLBrace)
	}
	if !expectCommand {
		return 0, "", "", -1, false
	}
	// Name + Line
	//
	var t token.Token
	switch {
	case tryPeekType(p, lexer.TokenDashID):
		t = expectTokenType(p, lexer.TokenDashID, "expecting command name")
	case tryPeekType(p, lexer.TokenDotID):
		t = expectTokenType(p, lexer.TokenDotID, "expecting command name")
	case tryPeekType(p, lexer.TokenCommandDefID):
		t = expectTokenType(p, lexer.TokenCommandDefID, "expecting command name")
	default:
		t = expectTokenType(p, lexer.TokenID, "expecting command name")
	}
	name := t.Value()
	line := t.Line()
	flags := config.CmdFlags(0)
	// Hidden / Private
	//
	if strings.HasPrefix(name, ".") {
		name = strings.TrimPrefix(name, ".")
		flags |= config.FlagHidden
	} else if strings.HasPrefix(name, "!") {
		name = strings.TrimPrefix(name, "!")
		flags |= config.FlagPrivate
	}
	// Shell
	//
	shell := ""
	if tryPeekType(p, lexer.TokenLParen) {
		expectTokenType(p, lexer.TokenLParen, "expecting TokenLParen ('(')")
		ctx.pushLexFn(ctx.l.Fn)
		ctx.setLexFn(lexer.LexCmdShellName)
		shell = expectTokenType(p, lexer.TokenID, "expecting shell name").Value()
		expectTokenType(p, lexer.TokenRParen, "expecting TokenRParen (')')")
	}
	// Colon or Brace - If not present,then error, but don't consume if present
	//
	if !tryPeekType(p, lexer.TokenColon) && !tryPeekType(p, lexer.TokenLBrace) {
		panic(parseError(p, "expecting TokenColon (':') or TokenLBrace ('{')"))
	}
	p.Clear()
	return flags, name, shell, line, true
}

// expectCmdScript
//
func expectCmdScript(ctx *parseContext, p *parser.Parser) []string {
	// Open Brace
	//
	ctx.setLexFn(lexer.LexMain)
	usingBraces := tryPeekType(p, lexer.TokenLBrace)
	if usingBraces {
		expectTokenType(p, lexer.TokenLBrace, "expecting TokenLBrace ('{')")
		ctx.setLexFn(lexer.LexCmdScriptAfterLBrace)
	} else {
		expectTokenType(p, lexer.TokenNewline, "expecting TokenNewline ('\\n')")
		ctx.setLexFn(lexer.LexCmdScriptMaybeLBrace)
		usingBraces = tryPeekType(p, lexer.TokenLBrace)
		if usingBraces {
			expectTokenType(p, lexer.TokenLBrace, "expecting TokenLBrace ('{')")
		}
	}
	// Script Body
	//
	var scriptText []string
	for p.CanPeek(1) && p.PeekType(1) == lexer.TokenScriptLine {
		scriptText = append(scriptText, p.Next().Value())
	}
	if usingBraces || p.CanPeek(1) {
		expectTokenType(p, lexer.TokenScriptEnd, "expecting TokenScriptEnd")
	}
	// Close Brace
	//
	if usingBraces {
		ctx.setLexFn(lexer.LexCmdScriptMaybeRBrace)
		expectTokenType(p, lexer.TokenRBrace, "expecting TokenRBrace ('}')")
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
	panic(parseError(p, msg)) // Do NOT copy this into a parsePanic method - see parseError for notes
}

// tokenMsg
//
func tokenMsg(t token.Token, msg string) string {
	return fmt.Sprintf("%s:%d.%d: %s", config.Runfile, t.Line(), t.Column(), msg)
}

// tokenError
//
func tokenError(t token.Token, msg string) error {
	return errors.New(tokenMsg(t, msg))
}

// parseError
// NOTE: Do NOT create a parsePanic() method to auto-panic
//       as it throws off the required return value at the call site.
//       Just use panic(parseError(p, "error"))
//
func parseError(p *parser.Parser, msg string) error {
	// If a token is available, use it for line/column
	//
	if p.CanPeek(1) {
		t := p.Peek(1)
		return tokenError(t, msg)
	}
	return fmt.Errorf("%s: <eof>: %s", config.Runfile, msg)
}
