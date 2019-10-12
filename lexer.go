package main

import (
	"bytes"
	"container/list"
	"strings"

	"github.com/tekwizely/go-parsing/lexer"
	"github.com/tekwizely/go-parsing/lexer/token"
)

// lexFn
//
type lexFn func(*lexContext, *lexer.Lexer) lexFn

// lexContext
//
type lexContext struct {
	fn      lexFn
	fnStack *list.List
	tokens  token.Nexter
}

// lex delegates incoming lexer calls to the configured fn
//
func (ctx *lexContext) lex(l *lexer.Lexer) lexer.Fn {
	fn := ctx.fn
	// EOF ?
	//
	if fn == nil {
		if ctx.fnStack.Len() == 0 {
			return nil
		}
		fn = ctx.fnStack.Remove(ctx.fnStack.Back()).(lexFn)
		traceFn("Popped lexer function", fn)
	}
	// assert(fn != nil)
	traceFn("Calling lexer function", fn)
	ctx.fn = fn(ctx, l)
	return ctx.lex
}

// pushFn
//
func (ctx *lexContext) pushFn(fn lexFn) {
	ctx.fnStack.PushBack(fn)
	traceFn("Pushed lexer function", fn)
}

// lex
//
func lex(fileBytes []byte) *lexContext {
	reader := &readerIgnoreCR{r: bytes.NewReader(fileBytes)}
	ctx := &lexContext{
		fn:      lexMain,
		fnStack: list.New(),
	}
	ctx.tokens = lexer.LexRuneReader(reader, ctx.lex)
	return ctx
}

// lexMain
//
func lexMain(ctx *lexContext, l *lexer.Lexer) lexFn {

	switch {
	// :=
	//
	case l.CanPeek(2) && l.Peek(1) == runeColon && l.Peek(2) == runeEquals:
		l.Next() // :
		l.Next() // =
		l.EmitType(tokenEquals)
	// ?=
	//
	case l.CanPeek(2) && l.Peek(1) == runeQMark && l.Peek(2) == runeEquals:
		l.Next() // ?
		l.Next() // =
		l.EmitType(tokenQMarkEquals)
	// Single-Char Token - Check AFTER multi-char tokens
	//
	case bytes.ContainsRune(singleRunes, l.Peek(1)):
		i := bytes.IndexRune(singleRunes, l.Peek(1))
		l.Next()                    // Match the rune
		l.EmitType(singleTokens[i]) // Emit just the type, discarding the matched rune
	// Comment
	//
	case matchRune(l, runeHash):
		// May be a hash-line
		//
		if matchOneOrMore(l, isHash) {
			// May be a single line doc comment
			//
			if matchOneOrMore(l, isSpaceOrTab) {
				l.Clear()
				if matchOneOrMore(l, isPrintNonReturn) {
					l.EmitToken(tokenConfigDescLine)
					if matchNewline(l) {
						l.Clear()
					}
					return lexMain
				}
			}
			if matchNewlineOrEOF(l) {
				l.EmitType(tokenHashLine)
				return lexMain
			}
		}
		// Consume rest of line as a standard comment
		//
		for !matchNewlineOrEOF(l) {
			l.Next()
		}
		l.Clear() // Discard the comment (for now)
	// Leading Whitespace
	//
	case matchOneOrMore(l, isSpaceOrTab):
		l.Clear() // Discard
	// Newline
	case matchNewline(l):
		l.EmitType(tokenNewline)
	// DotID
	//
	case matchDotID(l):
		l.EmitToken(tokenDotID)
	// ID
	//
	case matchID(l):
		name := strings.ToUpper(l.PeekToken())
		if t, ok := mainTokens[name]; ok {
			l.EmitType(t)
		} else {
			l.EmitToken(tokenID)
		}
	// Unknown
	//
	default:
		l.EmitErrorf("unexpected rune '%c'", l.Next())
		return nil
	}

	return lexMain
}

// lexAssignmentValue delegates to other rValue lexers
//
func lexAssignmentValue(ctx *lexContext, l *lexer.Lexer) lexFn {
	// Consume leading space
	//
	if matchOneOrMore(l, isSpaceOrTab) {
		l.Clear() // Discard
	}

	switch l.Peek(1) {
	// Does it look like a single-quoted string?
	//
	case runeSQuote:
		l.EmitType(tokenSQStringStart)
		return lexSQString
	case runeDQuote:
		l.EmitType(tokenDQStringStart)
		return lexDQString
	case runeDollar:
		return lexDollarString
	}
	return lexUQString
}

// lexDollarString
//
func lexDollarString(ctx *lexContext, l *lexer.Lexer) lexFn {
	if l.CanPeek(1) {
		if l.Peek(1) == runeDollar {
			if l.CanPeek(2) {
				switch l.Peek(2) {
				case runeLBrace:
					l.EmitType(tokenVarRefStart)
					return nil
				case runeLParen:
					l.EmitType(tokenSubCmdStart)
					return nil
				}
			}
		}
	}
	expectRune(l, runeDollar, "expecting dollar ('$')")
	l.EmitType(tokenDollar)
	return nil
}

// lexVarRef [ '$' '{' [A-Za-z0-9_.]* '}' ]
//
func lexVarRef(ctx *lexContext, l *lexer.Lexer) lexFn {
	// Dollar
	//
	expectRune(l, runeDollar, "expecting dollar ('$')")
	l.EmitType(tokenDollar)
	// Open Brace
	//
	expectRune(l, runeLBrace, "expecting l-brace ('{')")
	l.EmitType(tokenLBrace)
	// Variable Name
	//
	matchZeroOrMore(l, isAlphaNumDotUnder)
	l.EmitToken(tokenRunes) // Could be empty
	// Close Brace
	//
	expectRune(l, runeRBrace, "expecting r-brace ('}')")
	l.EmitType(tokenRBrace)

	return nil
}

// lexSubCmd - [ '$' '(' [::print::] ')' ]
//
func lexSubCmd(ctx *lexContext, l *lexer.Lexer) lexFn {
	// Dollar
	//
	expectRune(l, runeDollar, "expecting dollar ('$')")
	l.EmitType(tokenDollar)
	// Open Paren
	//
	expectRune(l, runeLParen, "expecting l-paren ('(')")
	l.EmitType(tokenLParen)
	// Keep going until we find close paren
	//
	for l.CanPeek(1) {
		switch {
		// Consume a run of printable, non-paren non-escape characters
		//
		case matchOneOrMore(l, isPrintNonParenNonBackslash):
			l.EmitToken(tokenRunes)
		// Back-slash '\'
		//
		case matchRune(l, runeBackSlash):
			// In Shell mode, only '\', '(' and ')' are escapable
			// Anything else is considered two separate characters
			//
			if matchRune(l, runeBackSlash, runeLParen, runeRParen) {
				l.EmitToken(tokenEscapeSequence)
			} else {
				l.EmitToken(tokenRunes)
			}
		// Better be Close Paren ')'
		//
		default:
			expectRune(l, runeRParen, "expecting r-paren (')')")
			l.EmitType(tokenRParen)
			return nil
		}
	}
	return nil
}

// lexSQString lexes a Single-Quoted String
// No escapable sequences in SQuotes, not even '\''
//
func lexSQString(ctx *lexContext, l *lexer.Lexer) lexFn {
	// Open quote
	//
	expectRune(l, runeSQuote, "expecting single-quote (\"'\")")
	l.EmitType(tokenSQuote)
	// Match quoted value as a one-shot
	//
	matchZeroOrMore(l, isPrintNonSQuote)
	l.EmitToken(tokenRunes) // Could be empty
	// Close quote
	//
	expectRune(l, runeSQuote, "expecting single-quote (\"'\")")
	l.EmitType(tokenSQuote)

	return nil
}

// lexDQString lexes a Double-Quoted String
//
func lexDQString(ctx *lexContext, l *lexer.Lexer) lexFn {
	// Open quote
	//
	expectRune(l, runeDQuote, "expecting double-quote ('\"')")
	l.EmitType(tokenDQuote)
	ctx.pushFn(lexEndDQString)
	return lexDQStringElement
}

// lexEndDQString
//
func lexEndDQString(ctx *lexContext, l *lexer.Lexer) lexFn {
	expectRune(l, runeDQuote, "expecting double-quote ('\"')")
	l.EmitType(tokenDQuote)
	return nil
}

// lexDQStringElement
//
func lexDQStringElement(ctx *lexContext, l *lexer.Lexer) lexFn {
	switch {
	// Consume a run of printable, non-quote non-escape characters
	//
	case matchOneOrMore(l, isPrintNonDQuoteNonBackslashNonDollar):
		l.EmitToken(tokenRunes)
	// Back-slash '\'
	//
	case matchRune(l, runeBackSlash):
		// In DQuote mode, currently only '\', '"' and '$' are escapable
		// Anything else is considered two separate characters
		//
		if matchRune(l, runeBackSlash, runeDQuote, runeDollar) {
			l.EmitToken(tokenEscapeSequence)
		} else {
			l.EmitToken(tokenRunes)
		}
	case l.CanPeek(1) && l.Peek(1) == runeDollar:
		ctx.pushFn(lexDQStringElement)
		return lexDollarString

	default:
		return nil
	}
	return lexDQStringElement
}

// lexUQString lexes an Unquoted string (no quotes, no interpolation)
//
func lexUQString(ctx *lexContext, l *lexer.Lexer) lexFn {
	matchZeroOrMore(l, isPrintNonSpace)
	l.EmitToken(tokenRunes) // Could be empty
	return nil
}

// lexDocBlockDesc
//
func lexDocBlockDesc(ctx *lexContext, l *lexer.Lexer) lexFn {
	m := l.Marker()
	if matchOne(l, isHash) {
		matchZeroOrMore(l, isSpaceOrTab)
		// 2+ # = ignored as a comment
		//
		if matchOne(l, isHash) {
			// Consume rest of line, including newline
			//
			for !matchNewlineOrEOF(l) {
				l.Next()
			}
			l.Clear() // Discard
		} else {
			l.Clear() // Clear # and leading space
			m = l.Marker()
			// Possible attribute
			//
			if matchID(l) {
				id := strings.ToUpper(l.PeekToken())
				if t, ok := cmdConfigTokens[id]; ok {
					// We've gone this far, let's go ahead and emit
					// the attribute (vs rewind and re-scan)
					//
					l.Clear()
					l.EmitToken(tokenNewline)
					l.EmitType(tokenConfigDescEnd)
					l.EmitType(t)
					return nil
				}
			}
			m.Apply()
			// Desc line
			//
			ctx.pushFn(lexDocBlockDesc)
			return lexDocBlockNQString
		}
		return lexDocBlockDesc
	}
	m.Apply()
	l.EmitType(tokenConfigDescEnd)
	return nil
}

// lexDocBlockNQString
//
func lexDocBlockNQString(ctx *lexContext, l *lexer.Lexer) lexFn {
	switch {
	// Consume a run of printable, non-escape characters
	//
	case matchOneOrMore(l, isPrintNonBackslashNonDollarNonReturn):
		l.EmitToken(tokenRunes)
	// Back-slash '\'
	//
	case matchRune(l, runeBackSlash):
		// Currently only '\' and '$' are escapable
		// Anything else is considered two separate characters
		//
		if matchRune(l, runeBackSlash, runeDollar) {
			l.EmitToken(tokenEscapeSequence)
		} else {
			l.EmitToken(tokenRunes)
		}
	// Variable reference
	//
	case l.CanPeek(1) && l.Peek(1) == runeDollar:
		if l.CanPeek(2) && l.Peek(2) == runeLBrace {
			ctx.pushFn(lexDocBlockNQString)
			l.EmitType(tokenVarRefStart)
			return lexVarRef
		}
		l.Next() // Consume $
		l.EmitToken(tokenRunes)
	default:
		if matchNewline(l) {
			l.EmitType(tokenNewline)
		}
		return nil
	}
	return lexDocBlockNQString
}

// lexDocBlockAttr
//
func lexDocBlockAttr(ctx *lexContext, l *lexer.Lexer) lexFn {
	m := l.Marker()
	if matchOne(l, isHash) {
		matchZeroOrMore(l, isSpaceOrTab)
		// 2+ # = ignored as a comment
		//
		if matchOne(l, isHash) {
			// Consume rest of line, including newline
			//
			for !matchNewlineOrEOF(l) {
				l.Next()
			}
			l.Clear() // Discard
			return lexDocBlockAttr
		}
		l.Clear() // Clear # and leading space
		// Ignore whitespace-only comment
		//
		if matchNewlineOrEOF(l) {
			l.Clear() // Discard
			return lexDocBlockAttr
		}
		if matchID(l) {
			name := strings.ToUpper(l.PeekToken())
			if t, ok := cmdConfigTokens[name]; ok {
				l.EmitType(t)
				return lexDocBlockAttr
			}
			l.EmitErrorf("Unrecognized command attribute: %s", name)
			return nil
		}
		l.EmitError("Expecting command attribute")
		return nil
	}
	m.Apply()
	l.EmitType(tokenConfigEnd)
	return nil
}

// lexCmdShell
//
func lexCmdShell(ctx *lexContext, l *lexer.Lexer) lexFn {
	ignoreSpace(l)
	if matchID(l) {
		l.EmitToken(tokenID)
	}
	ignoreSpace(l)
	ignoreEOL(l)
	return nil
}

// lexCmdUsage
//
func lexCmdUsage(ctx *lexContext, l *lexer.Lexer) lexFn {
	ignoreSpace(l)
	return lexDocBlockNQString
}

// lexCmdOpt: name [-l] [--long] [<label>] ["desc"]
//
func lexCmdOpt(ctx *lexContext, l *lexer.Lexer) lexFn {
	ignoreSpace(l)

	// ID
	//
	if !matchID(l) {
		l.EmitError("Expecting option name")
		return nil
	}
	l.EmitToken(tokenConfigOptName)

	// Whitespace
	//
	ignoreSpace(l)

	// First '-' : Might be short or long
	//
	expectRune(l, runeDash, "Expecting '-'")
	l.Clear()

	expectLong := true

	// Short flag?
	//
	if matchOne(l, isAlphaNum) {
		l.EmitToken(tokenConfigOptShort)

		// Whitespace
		//
		ignoreSpace(l)

		expectLong = matchRune(l, runeComma)
		l.Clear()

		if expectLong {
			// Whitespace
			//
			ignoreSpace(l)
			// First '-'
			//
			expectRune(l, runeDash, "Expecting '-'")
			l.Clear()
		}
	}

	// Long flag?
	//
	if expectLong {
		// Second '-'
		//
		expectRune(l, runeDash, "Expecting '-'")
		l.Clear()
		if matchOneOrMore(l, isAlphaNum) {
			l.EmitToken(tokenConfigOptLong)
		} else {
			l.EmitError("Expecting long flag name")
		}
	}

	// Whitespace
	//
	ignoreSpace(l)

	// Value?
	//
	if matchRune(l, runeLAngle) {
		l.Clear()
		matchOneOrMore(l, isConfigOptValue)
		l.EmitToken(tokenConfigOptValue)
		expectRune(l, runeRAngle, "Expecting '>'")
		l.Clear()
	}

	// Whitespace
	//
	ignoreSpace(l)

	// Desc?
	//
	return lexDocBlockNQString
}

func lexCmdOptEnd(ctx *lexContext, l *lexer.Lexer) lexFn {
	ignoreSpace(l)
	ignoreEOL(l)
	l.EmitType(tokenConfigOptEnd)
	return nil
}

func lexExport(ctx *lexContext, l *lexer.Lexer) lexFn {
	ignoreSpace(l)
	if !matchID(l) {
		l.EmitError("Expecting variable name")
		return nil
	}
	l.EmitToken(tokenID)
	ignoreSpace(l)

	switch {
	// ','
	//
	case peekRuneEquals(l, runeComma):
		for matchRune(l, runeComma) {
			l.EmitType(tokenComma)
			ignoreSpace(l)
			if !matchID(l) {
				l.EmitError("Expecting variable name")
				return nil
			}
			l.EmitToken(tokenID)
			ignoreSpace(l)
		}
	// '='
	//
	case matchRune(l, runeEquals):
		l.EmitType(tokenEquals)
	// ':='
	//
	case matchRune(l, runeColon):
		expectRune(l, runeEquals, "Expecting '='")
		l.EmitType(tokenEquals)
	// '?='
	//
	case matchRune(l, runeQMark):
		expectRune(l, runeEquals, "Expecting '='")
		l.EmitType(tokenQMarkEquals)
	default:
		// No default
	}

	return nil
}

// lexExpectNewline
//
func lexExpectNewline(ctx *lexContext, l *lexer.Lexer) lexFn {
	ignoreSpace(l)
	if !matchNewlineOrEOF(l) {
		l.EmitError("expecting end of line")
	}
	l.EmitType(tokenNewline)
	return nil
}

// lexCmdScriptMaybeLBrace
// If command header line ends with ':',
// then first line of script may actually be '{'
//
func lexCmdScriptMaybeLBrace(ctx *lexContext, l *lexer.Lexer) lexFn {
	if matchRune(l, runeLBrace) {
		l.EmitType(tokenLBrace)
		return lexCmdScriptAfterLBrace
	}
	return lexCmdScriptLine
}

// lexCmdScriptAfterLBrace
// Presumed to start immediately after '{'
// Consumes remainder of '{' line, so that cmdScriptLine loop always enters
// at the beginning of a line
//
func lexCmdScriptAfterLBrace(ctx *lexContext, l *lexer.Lexer) lexFn {
	// Discard whitespace to EOL
	//
	matchZeroOrMore(l, isSpaceOrTab)
	matchNewlineOrEOF(l)
	l.Clear()
	return lexCmdScriptLine
}

// lexCmdScriptLine Lexes one full line at a time
// Expects to enter fn at start of a line
//
func lexCmdScriptLine(ctx *lexContext, l *lexer.Lexer) lexFn {
	// Blank line is part of script
	// Need to check this before !whitespace
	//
	if matchNewline(l) {
		l.EmitToken(tokenScriptLine)
		return lexCmdScriptLine
	}
	m := l.Marker()
	// !whitespace at beginning of non-blank line terminates script
	//
	if !matchOneOrMore(l, isSpaceOrTab) {
		m.Apply()
		l.EmitType(tokenScriptEnd)
		return nil
	}
	// We have a script line
	// Consume the full line, including eol/eof
	//
	for !matchNewline(l) {
		l.Next()
	}
	l.EmitToken(tokenScriptLine)
	return lexCmdScriptLine
}

// lexCmdScriptMaybeRBrace
//
func lexCmdScriptMaybeRBrace(ctx *lexContext, l *lexer.Lexer) lexFn {
	if matchRune(l, runeRBrace) {
		l.EmitType(tokenRBrace)
	}
	return nil
}

// matchNewline
// We will attempt to match 3 newline styles: [ "\n", "\r", "\r\n" ]
// TODO Not sure we need [ '\r' | '\r\n' ] check since this generally a linux tool
//
func matchNewline(l *lexer.Lexer) bool {
	if matchRune(l, '\n') {
		return true
	}
	if matchRune(l, '\r') {
		matchRuneOrNone(l, '\n')
		return true
	}
	return false
}

// matchNewlineOrEOF
//
func matchNewlineOrEOF(l *lexer.Lexer) bool {
	if matchNewline(l) {
		return true
	}
	return !l.CanPeek(1)
}

// matchDotID matches [ '\.' [a-zA-Z] [a-zA-Z0-9_]* ( \. [a-zA-Z0-9_]+ )*
//
func matchDotID(l *lexer.Lexer) (ok bool) {
	m := l.Marker()
	// If we don't match then reset
	//
	defer func() {
		if !ok {
			m.Apply()
		}
	}()
	if matchRune(l, runeDot) && matchOne(l, isAlpha) {
		matchZeroOrMore(l, isAlphaNum)
		for matchRune(l, runeDot) {
			if !matchOneOrMore(l, isAlphaNum) {
				return ok
			}
		}
		return true
	}
	return false
}

// matchID
//
func matchID(l *lexer.Lexer) bool {
	return matchOne(l, isAlphaUnder) && matchZeroOrMore(l, isAlphaNumUnder)
}
