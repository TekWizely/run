package lexer

import (
	"bytes"
	"container/list"
	"strings"

	"github.com/tekwizely/go-parsing/lexer"
	"github.com/tekwizely/go-parsing/lexer/token"

	"github.com/tekwizely/run/internal/config"
)

// LexFn is a lexer fun that takes a context
//
type LexFn func(*LexContext, *lexer.Lexer) LexFn

// LexContext allows us to track additional states of the lexer
//
type LexContext struct {
	Fn      LexFn
	fnStack *list.List
	Tokens  token.Nexter
}

// lex delegates incoming lexer calls to the configured fn
//
func (ctx *LexContext) lex(l *lexer.Lexer) lexer.Fn {
	fn := ctx.Fn
	// EOF ?
	//
	if fn == nil {
		if ctx.fnStack.Len() == 0 {
			return nil
		}
		fn = ctx.fnStack.Remove(ctx.fnStack.Back()).(LexFn)
		config.TraceFn("Popped lexer function", fn)
	}
	// assert(fn != nil)
	config.TraceFn("Calling lexer function", fn)
	ctx.Fn = fn(ctx, l)
	return ctx.lex
}

// PushFn stores the specified function on the fn stack.
//
func (ctx *LexContext) PushFn(fn LexFn) {
	ctx.fnStack.PushBack(fn)
	config.TraceFn("Pushed lexer function", fn)
}

// Lex initiates the lexer against a byte array
//
func Lex(fileBytes []byte) *LexContext {
	reader := newReaderIgnoreCR(bytes.NewReader(fileBytes))
	ctx := &LexContext{
		Fn:      LexMain,
		fnStack: list.New(),
	}
	ctx.Tokens = lexer.LexRuneReader(reader, ctx.lex)
	return ctx
}

// LexMain is the primary lexer entry point
//
func LexMain(_ *LexContext, l *lexer.Lexer) LexFn {

	switch {
	// :=
	//
	case l.CanPeek(2) && l.Peek(1) == runeColon && l.Peek(2) == runeEquals:
		l.Next() // :
		l.Next() // =
		l.EmitType(TokenEquals)
	// ?=
	//
	case l.CanPeek(2) && l.Peek(1) == runeQMark && l.Peek(2) == runeEquals:
		l.Next() // ?
		l.Next() // =
		l.EmitType(TokenQMarkEquals)
	// Single-Char Token - Check AFTER multi-char tokens
	//
	case bytes.ContainsRune(singleRunes, l.Peek(1)):
		i := bytes.IndexRune(singleRunes, l.Peek(1))
		l.Next()                    // Match the rune
		l.EmitType(singleTokens[i]) // Emit just the type, discarding the matched rune
	// Comment
	//
	case matchRune(l, runeHash):
		// Might be a hash-line
		//
		if matchOneOrMore(l, isHash) {
			// Might be a single line doc comment
			//
			if matchOneOrMore(l, isSpaceOrTab) {
				l.Clear()
				if l.CanPeek(1) && isPrintNonReturn(l.Peek(1)) {
					l.EmitToken(TokenConfigDescLineStart)
					return LexMain
				}
			}
			if matchNewlineOrEOF(l) {
				l.EmitType(TokenHashLine)
				return LexMain
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
		l.EmitType(TokenNewline)
	// DotID
	//
	case matchDotID(l):
		l.EmitToken(TokenDotID)
	// Keyword / ID / DashID
	//
	case matchDashID(l):
		name := strings.ToUpper(l.PeekToken())
		switch {
		case isMainToken(name):
			l.EmitType(mainTokens[name])
		case strings.ContainsRune(name, runeDash):
			l.EmitToken(TokenDashID)
		default:
			l.EmitToken(TokenID)
		}
	// Unknown
	//
	default:
		l.Next()
		l.EmitToken(TokenUnknownRune)
		return nil
	}

	return LexMain
}

// LexAssignmentValue delegates to other rValue lexers
//
func LexAssignmentValue(_ *LexContext, l *lexer.Lexer) LexFn {
	ignoreSpace(l)
	switch l.Peek(1) {
	case runeSQuote:
		l.EmitType(TokenSQStringStart)
		return LexSQString
	case runeDQuote:
		l.EmitType(TokenDQStringStart)
		return LexDQString
	case runeDollar:
		return lexDollarString
	}
	return lexUQString
}

// lexDollarString
//
func lexDollarString(_ *LexContext, l *lexer.Lexer) LexFn {
	if l.CanPeek(1) {
		if l.Peek(1) == runeDollar {
			if l.CanPeek(2) {
				switch l.Peek(2) {
				case runeLBrace:
					l.EmitType(TokenVarRefStart)
					return nil
				case runeLParen:
					l.EmitType(TokenSubCmdStart)
					return nil
				}
			}
		}
	}
	expectRune(l, runeDollar, "expecting dollar ('$')")
	l.EmitType(TokenDollar)
	return nil
}

// LexVarRef matches: [ '$' '{' [A-Za-z0-9_.]* '}' ]
//
func LexVarRef(_ *LexContext, l *lexer.Lexer) LexFn {
	// Dollar
	//
	expectRune(l, runeDollar, "expecting dollar ('$')")
	l.EmitType(TokenDollar)
	// Open Brace
	//
	expectRune(l, runeLBrace, "expecting l-brace ('{')")
	l.EmitType(TokenLBrace)
	// Variable Name
	//
	matchZeroOrMore(l, isAlphaNumUnderDot)
	l.EmitToken(TokenRunes) // Could be empty
	// Close Brace
	//
	expectRune(l, runeRBrace, "expecting r-brace ('}')")
	l.EmitType(TokenRBrace)

	return nil
}

// LexSubCmd matches: [ '$' '(' [::print::] ')' ]
//
func LexSubCmd(_ *LexContext, l *lexer.Lexer) LexFn {
	// Dollar
	//
	expectRune(l, runeDollar, "expecting dollar ('$')")
	l.EmitType(TokenDollar)
	// Open Paren
	//
	expectRune(l, runeLParen, "expecting l-paren ('(')")
	l.EmitType(TokenLParen)
	// Keep going until we find close paren
	//
	for l.CanPeek(1) {
		switch {
		// Consume a run of printable, non-paren non-escape characters
		//
		case matchOneOrMore(l, isPrintNonParenNonBackslash):
			l.EmitToken(TokenRunes)
		// Back-slash '\'
		//
		case matchRune(l, runeBackSlash):
			// In Shell mode, only '\', '(' and ')' are escapable
			// Anything else is considered two separate characters
			//
			if matchRune(l, runeBackSlash, runeLParen, runeRParen) {
				l.EmitToken(TokenEscapeSequence)
			} else {
				l.EmitToken(TokenRunes)
			}
		// Better be Close Paren ')'
		//
		default:
			expectRune(l, runeRParen, "expecting r-paren (')')")
			l.EmitType(TokenRParen)
			return nil
		}
	}
	return nil
}

// LexAssert assumes 'ASSERT' has already been matched
//
func LexAssert(ctx *LexContext, l *lexer.Lexer) LexFn {
	ignoreSpace(l)
	ctx.PushFn(LexAssertMessage)
	return lexTestString
}

// LexAssertMessage parses an (optional) assertion error message
//
func LexAssertMessage(_ *LexContext, l *lexer.Lexer) LexFn {
	ignoreSpace(l)
	switch {
	// "'"
	//
	case peekRuneEquals(l, runeSQuote):
		l.EmitType(TokenSQStringStart)
		return LexSQString
	// '"'
	//
	case peekRuneEquals(l, runeDQuote):
		l.EmitType(TokenDQStringStart)
		return LexDQString
	default:
		l.EmitType(TokenEmptyAssertMessage)
		return nil
	}
}

// lexTestString
//
func lexTestString(ctx *LexContext, l *lexer.Lexer) LexFn {
	//goland:noinspection GoImportUsedAsName
	var (
		token     token.Type
		elementFn LexFn
		endFn     LexFn
	)
	switch {

	// '[' | '[['
	//
	case matchRune(l, runeLBracket):
		elementFn = lexBracketStringElement
		if matchRune(l, runeLBracket) {
			endFn = lexEndDBracketString
			token = TokenDBracketStringStart
		} else {
			endFn = lexEndBracketString
			token = TokenBracketStringStart
		}
	// '(' | '(('
	//
	case matchRune(l, runeLParen):
		elementFn = lexParenStringElement
		if matchRune(l, runeLParen) {
			endFn = lexEndDParenString
			token = TokenDParenStringStart
		} else {
			endFn = lexEndParenString
			token = TokenParenStringStart
		}
	}
	expectRune(l, ' ', "expecting space (' ')")
	l.EmitType(token)
	ctx.PushFn(endFn)
	return elementFn
}

// lexEndBracketString
//
func lexEndBracketString(_ *LexContext, l *lexer.Lexer) LexFn {
	expectRune(l, ' ', "expecting space (' ')")
	expectRune(l, runeRBracket, "expecting right-bracket (']')")
	l.EmitType(TokenBracketStringEnd)
	return nil
}

// lexEndDBracketString
//
func lexEndDBracketString(_ *LexContext, l *lexer.Lexer) LexFn {
	expectRune(l, ' ', "expecting space (' ')")
	expectRune(l, runeRBracket, "expecting double-right-bracket (']]')")
	expectRune(l, runeRBracket, "expecting double-right-bracket (']]')")
	l.EmitType(TokenDBracketStringEnd)
	return nil
}

// lexBracketStringElement
//
func lexBracketStringElement(_ *LexContext, l *lexer.Lexer) LexFn {
	switch {
	// Space may be end of string
	//
	case l.CanPeek(1) && l.Peek(1) == ' ' && l.CanPeek(2) && l.Peek(2) == runeRBracket:
		// Leave text to be matched by endFn
		//
		return nil
	case matchRune(l, ' '):
		l.EmitToken(TokenRunes)
	// Consume a run of printable, non-bracket non-escape, non-space characters
	//
	case matchOneOrMore(l, isPrintNonBracketNonBackslashNonSpace):
		l.EmitToken(TokenRunes)
	// Back-slash '\'
	//
	case matchRune(l, runeBackSlash):
		// In Bracket String mode, currently only '\', '[' and ']' are escapable
		// Anything else is considered two separate characters
		//
		if matchRune(l, runeBackSlash, runeLBracket, runeRBracket) {
			l.EmitToken(TokenEscapeSequence)
		} else {
			l.EmitToken(TokenRunes)
		}
	default:
		return nil
	}
	return lexBracketStringElement
}

// lexEndParenString
//
func lexEndParenString(_ *LexContext, l *lexer.Lexer) LexFn {
	expectRune(l, ' ', "expecting space (' ')")
	expectRune(l, runeRParen, "expecting right-paren (')')")
	l.EmitType(TokenParenStringEnd)
	return nil
}

// lexEndDParenString
//
func lexEndDParenString(_ *LexContext, l *lexer.Lexer) LexFn {
	expectRune(l, ' ', "expecting space (' ')")
	expectRune(l, runeRParen, "expecting double-right-paren ('))')")
	expectRune(l, runeRParen, "expecting double-right-paren ('))')")
	l.EmitType(TokenDParenStringEnd)
	return nil
}

// lexParenStringElement
//
func lexParenStringElement(_ *LexContext, l *lexer.Lexer) LexFn {
	switch {
	// Space may be end of string
	//
	case l.CanPeek(1) && l.Peek(1) == ' ' && l.CanPeek(2) && l.Peek(2) == runeRParen:
		// Leave text to be matched by endFn
		//
		return nil
	case matchRune(l, ' '):
		l.EmitToken(TokenRunes)
	// Consume a run of printable, non-paren non-escape characters
	//
	case matchOneOrMore(l, isPrintNonParenNonBackslashNonSpace):
		l.EmitToken(TokenRunes)
	// Back-slash '\'
	//
	case matchRune(l, runeBackSlash):
		// In Paren String mode, currently only '\', '(' and ')' are escapable
		// Anything else is considered two separate characters
		//
		if matchRune(l, runeBackSlash, runeLParen, runeRParen) {
			l.EmitToken(TokenEscapeSequence)
		} else {
			l.EmitToken(TokenRunes)
		}
	default:
		return nil
	}
	return lexParenStringElement
}

// LexSQString lexes a Single-Quoted String
// No escapable sequences in SQuotes, not even '\''
//
func LexSQString(_ *LexContext, l *lexer.Lexer) LexFn {
	// Open quote
	//
	expectRune(l, runeSQuote, "expecting single-quote (\"'\")")
	l.EmitType(TokenSQuote)
	// Match quoted value as a one-shot
	//
	matchZeroOrMore(l, isPrintNonSQuote)
	l.EmitToken(TokenRunes) // Could be empty
	// Close quote
	//
	expectRune(l, runeSQuote, "expecting single-quote (\"'\")")
	l.EmitType(TokenSQuote)

	return nil
}

// LexDQString lexes a Double-Quoted String
//
func LexDQString(ctx *LexContext, l *lexer.Lexer) LexFn {
	// Open quote
	//
	expectRune(l, runeDQuote, "expecting double-quote ('\"')")
	l.EmitType(TokenDQuote)
	ctx.PushFn(lexEndDQString)
	return lexDQStringElement
}

// lexEndDQString
//
func lexEndDQString(_ *LexContext, l *lexer.Lexer) LexFn {
	expectRune(l, runeDQuote, "expecting double-quote ('\"')")
	l.EmitType(TokenDQuote)
	return nil
}

// lexDQStringElement
//
func lexDQStringElement(ctx *LexContext, l *lexer.Lexer) LexFn {
	switch {
	// Consume a run of printable, non-quote non-escape characters
	//
	case matchOneOrMore(l, isPrintNonDQuoteNonBackslashNonDollar):
		l.EmitToken(TokenRunes)
	// Back-slash '\'
	//
	case matchRune(l, runeBackSlash):
		// In DQuote mode, currently only '\', '"' and '$' are escapable
		// Anything else is considered two separate characters
		//
		if matchRune(l, runeBackSlash, runeDQuote, runeDollar) {
			l.EmitToken(TokenEscapeSequence)
		} else {
			l.EmitToken(TokenRunes)
		}
	case l.CanPeek(1) && l.Peek(1) == runeDollar:
		ctx.PushFn(lexDQStringElement)
		return lexDollarString

	default:
		return nil
	}
	return lexDQStringElement
}

// lexUQString lexes an Unquoted string (no quotes, no interpolation)
//
func lexUQString(_ *LexContext, l *lexer.Lexer) LexFn {
	matchZeroOrMore(l, isPrintNonSpace)
	l.EmitToken(TokenRunes) // Could be empty
	return nil
}

// LexDocBlockDesc lexes a single dock block description line.
//
func LexDocBlockDesc(ctx *LexContext, l *lexer.Lexer) LexFn {
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
					l.EmitToken(TokenNewline)
					l.EmitType(TokenConfigDescEnd)
					l.EmitType(t)
					return nil
				}
			}
			m.Apply()
			// Desc line
			//
			ctx.PushFn(LexDocBlockDesc)
			return LexDocBlockNQString
		}
		return LexDocBlockDesc
	}
	m.Apply()
	l.EmitType(TokenConfigDescEnd)
	return nil
}

// LexDocBlockNQString lexes a doc block comment line
//
func LexDocBlockNQString(ctx *LexContext, l *lexer.Lexer) LexFn {
	switch {
	// Consume a run of printable, non-escape characters
	//
	case matchOneOrMore(l, isPrintNonBackslashNonDollarNonReturn):
		l.EmitToken(TokenRunes)
	// Back-slash '\'
	//
	case matchRune(l, runeBackSlash):
		// Currently only '\' and '$' are escapable
		// Anything else is considered two separate characters
		//
		if matchRune(l, runeBackSlash, runeDollar) {
			l.EmitToken(TokenEscapeSequence)
		} else {
			l.EmitToken(TokenRunes)
		}
	// Variable reference
	//
	case l.CanPeek(1) && l.Peek(1) == runeDollar:
		if l.CanPeek(2) && l.Peek(2) == runeLBrace {
			ctx.PushFn(LexDocBlockNQString)
			l.EmitType(TokenVarRefStart)
			return LexVarRef
		}
		l.Next() // Consume $
		l.EmitToken(TokenRunes)
	default:
		if matchNewline(l) {
			l.EmitType(TokenNewline)
		}
		return nil
	}
	return LexDocBlockNQString
}

// LexDocBlockAttr lexes a doc block attribute line
//
func LexDocBlockAttr(_ *LexContext, l *lexer.Lexer) LexFn {
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
			return LexDocBlockAttr
		}
		l.Clear() // Clear # and leading space
		// Ignore whitespace-only comment
		//
		if matchNewlineOrEOF(l) {
			l.Clear() // Discard
			return LexDocBlockAttr
		}
		if matchID(l) {
			name := strings.ToUpper(l.PeekToken())
			if t, ok := cmdConfigTokens[name]; ok {
				l.EmitType(t)
				return LexDocBlockAttr
			}
			l.EmitErrorf("Unrecognized command attribute: %s", name)
			return nil
		}
		l.EmitError("expecting command attribute")
		return nil
	}
	m.Apply()
	l.EmitType(TokenConfigEnd)
	return nil
}

// LexCmdConfigShell lexes a doc block SHELL line
//
func LexCmdConfigShell(ctx *LexContext, l *lexer.Lexer) LexFn {
	ignoreSpace(l)
	ctx.PushFn(LexIgnoreNewline)
	return LexCmdShellName
}

// LexCmdConfigUsage lexes a doc block USAGE line
//
func LexCmdConfigUsage(_ *LexContext, l *lexer.Lexer) LexFn {
	ignoreSpace(l)
	return LexDocBlockNQString
}

// LexCmdConfigOpt matches: name [-l] [--long] [<label>] ["desc"]
//
func LexCmdConfigOpt(_ *LexContext, l *lexer.Lexer) LexFn {
	ignoreSpace(l)

	// ID
	//
	if !matchID(l) {
		l.EmitError("expecting option name")
		return nil
	}
	l.EmitToken(TokenConfigOptName)

	// Whitespace
	//
	ignoreSpace(l)

	// First '-' : Might be short or long
	//
	expectRune(l, runeDash, "expecting '-'")
	l.Clear()

	expectLong := true

	// Short flag?
	//
	if matchOne(l, isAlphaNum) {
		l.EmitToken(TokenConfigOptShort)

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
			expectRune(l, runeDash, "expecting '-'")
			l.Clear()
		}
	}

	// Long flag?
	//
	if expectLong {
		// Second '-'
		//
		expectRune(l, runeDash, "expecting '-'")
		l.Clear()
		if matchOneOrMore(l, isAlphaNum) {
			l.EmitToken(TokenConfigOptLong)
		} else {
			l.EmitError("expecting long flag name")
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
		l.EmitToken(TokenConfigOptValue)
		expectRune(l, runeRAngle, "expecting '>'")
		l.Clear()
	}

	// Whitespace
	//
	ignoreSpace(l)

	// Desc?
	//
	return LexDocBlockNQString
}

// LexCmdShellName lexes a command's shell
//
func LexCmdShellName(_ *LexContext, l *lexer.Lexer) LexFn {
	if matchID(l) {
		l.EmitToken(TokenID)
	} else if l.CanPeek(2) && l.Peek(1) == runeHash && l.Peek(2) == runeBang {
		l.Next()             // '#'
		l.Next()             // '!'
		l.EmitToken(TokenID) // HACK : Not really an ID
	}
	return nil
}

// LexExport lexes a global OR doc block EXPORT line
//
func LexExport(_ *LexContext, l *lexer.Lexer) LexFn {
	ignoreSpace(l)
	switch {
	// Variable
	//
	case matchID(l):
		l.EmitToken(TokenID)
		ignoreSpace(l)

		switch {
		// ','
		//
		case peekRuneEquals(l, runeComma):
			for matchRune(l, runeComma) {
				l.EmitType(TokenComma)
				ignoreSpace(l)
				if !matchID(l) {
					l.EmitError("expecting variable name")
					return nil
				}
				l.EmitToken(TokenID)
				ignoreSpace(l)
			}
		// '='
		//
		case matchRune(l, runeEquals):
			l.EmitType(TokenEquals)
		// ':='
		//
		case matchRune(l, runeColon):
			expectRune(l, runeEquals, "expecting '='")
			l.EmitType(TokenEquals)
		// '?='
		//
		case matchRune(l, runeQMark):
			expectRune(l, runeEquals, "expecting '='")
			l.EmitType(TokenQMarkEquals)
		default:
			// No default
		}
	// Attribute
	//
	case matchDotID(l):
		l.EmitToken(TokenDotID)
		ignoreSpace(l)
		// 'AS'
		//
		if l.CanPeek(3) &&
			(l.Peek(1) == 'a' || l.Peek(1) == 'A') &&
			(l.Peek(2) == 's' || l.Peek(2) == 'S') &&
			isSpaceOrTab(l.Peek(3)) {
			l.Next()
			l.Next()
			l.EmitType(TokenAs)
			ignoreSpace(l)
			matchID(l)
			l.EmitToken(TokenID)
		}
	default:
		l.EmitError("expecting attribute or variable name")
	}

	return nil
}

// LexIgnoreNewline matches + ignores whitespace + newline
//
func LexIgnoreNewline(_ *LexContext, l *lexer.Lexer) LexFn {
	ignoreSpace(l)
	ignoreEOL(l)
	return nil
}

// LexExpectNewline matches whitespace + newline or throws an error.
//
func LexExpectNewline(_ *LexContext, l *lexer.Lexer) LexFn {
	ignoreSpace(l)
	if !matchNewlineOrEOF(l) {
		l.EmitError("expecting end of line")
	}
	l.EmitType(TokenNewline)
	return nil
}

// LexCmdScriptMaybeLBrace lexes a script with an optional leading LBrace.
// If command header line ends with ':',
// then first line of script may actually be '{'
//
func LexCmdScriptMaybeLBrace(_ *LexContext, l *lexer.Lexer) LexFn {
	if matchRune(l, runeLBrace) {
		l.EmitType(TokenLBrace)
		return LexCmdScriptAfterLBrace
	}
	return lexCmdScriptLine
}

// LexCmdScriptAfterLBrace finishes a script after the trailing LBrace.
// Presumed to start immediately after '{'
// Consumes remainder of '{' line, so that cmdScriptLine loop always enters
// at the beginning of a line
//
func LexCmdScriptAfterLBrace(_ *LexContext, l *lexer.Lexer) LexFn {
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
func lexCmdScriptLine(_ *LexContext, l *lexer.Lexer) LexFn {
	// Blank line is part of script
	// Need to check this before !whitespace
	//
	if matchNewline(l) {
		l.EmitToken(TokenScriptLine)
		return lexCmdScriptLine
	}
	m := l.Marker()
	// # at beginning of script line = ignore line
	//
	if peekRuneEquals(l, runeHash) {
		// Consume comment block
		//
		for peekRuneEquals(l, runeHash) {
			l.Next()
			// Consume rest of line, including newline
			//
			for !matchNewlineOrEOF(l) {
				l.Next()
			}
		}
		// If the line following the comment block is not part of the script,
		// then assume the comment is also not part of the script
		// NOTE: We have to include RBrace here on the chance that the user
		//       is using braces to wrap the script.
		//
		m2 := l.Marker()
		if matchNewlineOrEOF(l) || matchOneOrMore(l, isSpaceOrTab) || matchRune(l, runeRBrace) {
			m2.Apply()
			l.Clear() // Discard the comment block
			return lexCmdScriptLine
		}
		m.Apply() // Consider script as ending before the comment block, re-parse comment block later
		l.EmitType(TokenScriptEnd)
		return nil
	}
	// !whitespace at beginning of non-blank line terminates script
	//
	if !matchOneOrMore(l, isSpaceOrTab) {
		m.Apply()
		l.EmitType(TokenScriptEnd)
		return nil
	}
	// We have a script line
	// Consume the full line, including eol/eof
	//
	for !matchNewlineOrEOF(l) {
		l.Next()
	}
	l.EmitToken(TokenScriptLine)
	return lexCmdScriptLine
}

// LexCmdScriptMaybeRBrace tries to match an RBrace
//
func LexCmdScriptMaybeRBrace(_ *LexContext, l *lexer.Lexer) LexFn {
	if matchRune(l, runeRBrace) {
		l.EmitType(TokenRBrace)
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

// matchNewlineOrEOF tries to match a newline or EOF, returning success or failure.
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

// matchDashID
//
func matchDashID(l *lexer.Lexer) bool {
	return matchOne(l, isAlphaUnder) && matchZeroOrMore(l, isAlphaNumUnderDash)
}
