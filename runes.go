package main

import (
	"io"
	"unicode"
	"unicode/utf8"

	"github.com/tekwizely/go-parsing/lexer"
	"github.com/tekwizely/go-parsing/lexer/token"
)

// Runes
//
const (
	runeSpace = ' '
	runeTab   = '\t'
	// NOTE: You probably want matchNewline()
	// runeNewline = '\n'
	// runeReturn    = '\r'
	runeHash      = '#'
	runeDollar    = '$'
	runeDot       = '.'
	runeComma     = ','
	runeDash      = '-'
	runeEquals    = '='
	runeQMark     = '?'
	runeColon     = ':'
	runeBackSlash = '\\'
	runeDQuote    = '"'
	runeSQuote    = '\''
	runeLParen    = '('
	runeRParen    = ')'
	runeLBrace    = '{'
	runeRBrace    = '}'
	runeLAngle    = '<'
	runeRAngle    = '>'
)

// Single-Rune tokens
//
var (
	singleRunes  = []byte{runeColon, runeEquals, runeLParen, runeRParen, runeLBrace, runeRBrace}
	singleTokens = []token.Type{tokenColon, tokenEquals, tokenLParen, tokenRParen, tokenLBrace, tokenRBrace}
)
var mainTokens = map[string]token.Type{
	"COMMAND": tokenCommand,
	"CMD":     tokenCommand,
	"EXPORT":  tokenExport,
}

// Cmd Config Tokens
//
var cmdConfigTokens = map[string]token.Type{
	"SHELL":  tokenConfigShell,
	"USAGE":  tokenConfigUsage,
	"OPTION": tokenConfigOpt,
	"OPT":    tokenConfigOpt,
	"EXPORT": tokenConfigExport,
}

func isAlpha(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

func isAlphaUnder(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_'
}

func isAlphaNum(r rune) bool {
	return (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

func isAlphaNumUnder(r rune) bool {
	return (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_'
}

func isAlphaNumDotUnder(r rune) bool {
	return (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_' || r == '.'
}

func isHash(r rune) bool {
	return r == runeHash
}

func isSpaceOrTab(r rune) bool {
	return r == runeSpace || r == runeTab
}

func isPrintNonSpace(r rune) bool {
	return unicode.IsPrint(r) && !unicode.IsSpace(r)
}

func isPrintNonReturn(r rune) bool {
	return unicode.IsPrint(r) && r != '\r' && r != '\n'
}

func isConfigOptValue(r rune) bool {
	return unicode.IsPrint(r) && r != '\r' && r != '\n' && r != '\t' && r != '<' && r != '>'
}

func isPrintNonSQuote(r rune) bool {
	return r != runeSQuote && unicode.IsPrint(r)
}

func isPrintNonDQuoteNonBackslash(r rune) bool {
	return r != runeDQuote && r != runeBackSlash && unicode.IsPrint(r)
}

func isPrintNonDQuoteNonBackslashNonDollar(r rune) bool {
	return r != runeDQuote && r != runeBackSlash && r != runeDollar && unicode.IsPrint(r)
}

func isPrintNonParenNonBackslash(r rune) bool {
	return r != runeLParen && r != runeRParen && r != runeBackSlash && unicode.IsPrint(r)
}

func isPrintNonBackslashNonDollarNonReturn(r rune) bool {
	return r != runeBackSlash && r != runeDollar && isPrintNonReturn(r)
}

func tryPeekRune(l *lexer.Lexer) (rune, bool) {
	if l.CanPeek(1) {
		return l.Peek(1), true
	}
	return utf8.RuneError, false
}

func peekRuneEquals(l *lexer.Lexer, r rune) bool {
	return l.CanPeek(1) && l.Peek(1) == r
}

func expectRune(l *lexer.Lexer, r rune, msg string) {
	if !l.CanPeek(1) || l.Peek(1) != r {
		l.EmitError(msg)
		return
	}
	l.Next()
}

// ReaderIgnoreCR wraps a RuneReader, filtering errOut '\r'
// Useful for input sources that use '\r'+'\n' for end-of-line
//
func ReaderIgnoreCR(r io.RuneReader) io.RuneReader {
	return &readerIgnoreCR{r: r}
}

type readerIgnoreCR struct {
	r io.RuneReader
}

func (c *readerIgnoreCR) ReadRune() (r rune, size int, err error) {
	r, size, err = c.r.ReadRune()
	if size == 1 && r == '\r' {
		r, size, err = c.r.ReadRune()
	}
	return
}
