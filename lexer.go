package main

import (
	"bytes"
	"unicode"

	"github.com/tekwizely/go-parsing/lexer"
	"github.com/tekwizely/go-parsing/lexer/token"
)

// Runes
//
const (
	runeSpace   = ' '
	runeTab     = '\t'
	runeHash    = '#'
	runeNewLine = '\n'
	runeReturn  = '\r'
	runeColon   = ':'
	runeLParen  = '('
	runeRParen  = ')'
	runeLBrace  = '{'
	runeRBrace  = '}'
)

// Single-Rune tokens
//
var (
	singleRunes  = []byte{runeColon, runeLParen, runeRParen, runeLBrace, runeRBrace}
	singleTokens = []token.Type{tokenColon, tokenLParen, tokenRParen, tokenLBrace, tokenRBrace}
)

type lexContext struct {
	fn     lexer.Fn
	tokens token.Nexter
}

// lex delegates incoming lexer calls to the configured fn
//
func (c *lexContext) lex(l *lexer.Lexer) lexer.Fn {
	// EOF ?
	//
	if c.fn == nil {
		return nil
	}
	c.fn = c.fn(l)
	return c.lex
}

func (c *lexContext) Tokens() token.Nexter {
	return c.tokens
}

// setFn configures the next lexer fn to delegate to
//
func (c *lexContext) SetFn(fn lexer.Fn) {
	c.fn = fn
}

// lex
//
func lex(content string) *lexContext {
	ctx := &lexContext{fn: lexMain}
	ctx.tokens = lexer.LexString(content, ctx.lex)
	return ctx
}

// lexMain
//
func lexMain(l *lexer.Lexer) lexer.Fn {

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
	case tryMatchComment(l):
		l.Clear() // Discard the comment (for now)

	// Whitespace
	//
	case l.CanPeek(1) && unicode.IsSpace(l.Peek(1)):
		l.Next()
		// Consume a run
		//
		if l.CanPeek(1) && unicode.IsSpace(l.Peek(1)) {
			l.Next()
		}
		l.Clear() // Discard them all

	// ID
	//
	case tryMatchID(l):
		l.EmitToken(TokenID)

	// Unknown
	//
	default:
		l.EmitErrorf("unexpected rune '%c'", l.Next())
		return nil
	}

	return lexMain
}

// lexScript Lexes one full line at a time
// Expects to enter fn at start of a line
//
func lexScript(l *lexer.Lexer) lexer.Fn {
	// EOF terminates Script
	//
	if !l.CanPeek(1) {
		l.EmitType(tokenEndScript)
		return nil
	}
	// Blank line is part of script
	//
	if tryMatchNewline(l) {
		l.EmitToken(tokenScriptLine)
		return lexScript
	}
	// !whitespace at beginning of non-blank line terminates script
	//
	if !tryMatchSpaceOrTab(l) {
		l.EmitType(tokenEndScript)
		return nil
	}
	// We have a scrip line
	// Consume the full line, including eol/eof
	//
	for !tryMatchNewline(l) {
		l.Next()
	}
	l.EmitToken(tokenScriptLine)
	return lexScript
}

// tryMatchComment
//
func tryMatchComment(l *lexer.Lexer) bool {
	if l.CanPeek(1) && l.Peek(1) == runeHash {
		l.Next()
		for !tryMatchNewlineOrEOF(l) {
			l.Next()
		}
		return true
	}
	return false
}

// tryMatchNewlineOrEOF
//
func tryMatchNewlineOrEOF(l *lexer.Lexer) bool {
	if tryMatchNewline(l) {
		return true
	}
	return !l.CanPeek(1)
}

// tryMatchNewline
// We will attempt to match 3 newline styles: [ "\n", "\r", "\r\n" ]
// TODO Not sure we need '\r | \r\n' check since this generally a linux tool
//
func tryMatchNewline(l *lexer.Lexer) bool {
	if l.CanPeek(1) {
		switch l.Peek(1) {
		// Newline '\n'
		//
		case runeNewLine:
			l.Next()
			return true

		// Return '\r', optionally followed by newLine '\n'
		//
		case runeReturn:
			l.Next()
			if l.CanPeek(1) && l.Peek(1) == runeNewLine {
				l.Next()
			}
			return true
		}

	}
	return false
}

// tryMatchID
//
func tryMatchID(l *lexer.Lexer) bool {
	if tryMatchAlpha(l) {
		for tryMatchAlphaNum(l) {
		}
		return true
	}
	return false
}

// tryMatchAlpha
//
func tryMatchAlpha(l *lexer.Lexer) bool {
	if l.CanPeek(1) {
		if r := l.Peek(1); (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			l.Next()
			return true
		}
	}
	return false
}

// tryMatchAlphaNum
//
func tryMatchAlphaNum(l *lexer.Lexer) bool {
	if l.CanPeek(1) {
		if r := l.Peek(1); (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
			l.Next()
			return true
		}
	}
	return false
}

// tryMatchSpaceOrTab
//
func tryMatchSpaceOrTab(l *lexer.Lexer) bool {
	if l.CanPeek(1) {
		if r := l.Peek(1); r == runeSpace || r == runeTab {
			l.Next()
			return true
		}
	}
	return false
}
