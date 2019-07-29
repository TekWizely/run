package main

import (
	"bytes"
	"container/list"

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
	}
	// assert(fn != nil)
	ctx.fn = fn(ctx, l)
	return ctx.lex
}

// pushFn
//
func (ctx *lexContext) pushFn(fn lexFn) {
	ctx.fnStack.PushBack(fn)
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

	// Single-char token?
	//
	if i := bytes.IndexRune(singleRunes, l.Peek(1)); i >= 0 {
		l.Next()                    // Match the rune
		l.EmitType(singleTokens[i]) // Emit just the type, discarding the matched rune
		return lexMain
	}

	switch {
	// Comment
	//
	case matchComment(l):
		l.Clear() // Discard the comment (for now)
	// Whitespace run
	//
	case matchOneOrMore(l, isSpaceOrTab) || matchNewline(l):
		l.Clear() // Discard them all
	// DotID
	//
	case matchDotID(l):
		l.EmitToken(tokenDotID)
	// ID
	//
	case matchID(l):
		l.EmitToken(tokenID)
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
	case '\'':
		l.EmitType(tokenSQStringStart)
	case '"':
		l.EmitType(tokenDQStringStart)
	case '$':
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
func lexEndDQString(ctx *lexContext, l *lexer.Lexer) lexFn {
	expectRune(l, runeDQuote, "expecting double-quote ('\"')")
	l.EmitType(tokenDQuote)
	return nil
}

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

// lexUQString
//
func lexUQString(ctx *lexContext, l *lexer.Lexer) lexFn {
	matchZeroOrMore(l, isPrintNonSpace)
	l.EmitToken(tokenRunes) // Could be empty
	return nil
}

// lexScriptBody Lexes one full line at a time
// Expects to enter fn at start of a line
//
func lexScriptBody(ctx *lexContext, l *lexer.Lexer) lexFn {
	// EOF terminates Script
	//
	if !l.CanPeek(1) {
		l.EmitType(tokenScriptEnd)
		return nil
	}
	// Blank line is part of script
	// Need to check this before !whitespace
	//
	if matchNewline(l) {
		l.EmitToken(tokenScriptLine)
		return lexScriptBody
	}
	// !whitespace at beginning of non-blank line terminates script
	//
	if !matchRune(l, runeSpace, runeTab) {
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
	return lexScriptBody
}

// matchComment
//
func matchComment(l *lexer.Lexer) bool {
	if matchRune(l, runeHash) {
		for !matchNewlineOrEOF(l) {
			l.Next()
		}
		return true
	}
	return false
}

// matchNewline
// We will attempt to match 3 newline styles: [ "\n", "\r", "\r\n" ]
// TODO Not sure we need [ '\r' | '\r\n' ] check since this generally a linux tool
//
func matchNewline(l *lexer.Lexer) bool {
	if matchRune(l, runeNewline) {
		return true
	}
	if matchRune(l, runeReturn) {
		matchRuneOrNone(l, runeNewline)
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
