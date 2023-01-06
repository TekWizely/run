package lexer

import (
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
	// runeNewline   = '\n'
	// runeReturn    = '\r'
	// runeAt        = '@'
	runeBang      = '!'
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
	runeLBracket  = '['
	runeRBracket  = ']'
)

// Single-Rune tokens
//
var (
	singleRunes  = []byte{runeQMark, runeColon, runeEquals, runeLParen, runeRParen, runeLBrace, runeRBrace, runeLBracket, runeRBracket}
	singleTokens = []token.Type{TokenQMark, TokenColon, TokenEquals, TokenLParen, TokenRParen, TokenLBrace, TokenRBrace, TokenLBracket, TokenRBracket}
)
var mainTokens = map[string]token.Type{
	"COMMAND":     TokenCommand,
	"CMD":         TokenCommand,
	"EXPORT":      TokenExport,
	"ASSERT":      TokenAssert,
	"INCLUDE":     TokenInclude,
	"INCLUDE.ENV": TokenIncludeEnv,
}

// isMainToken isolates the lookup+check-ok logic.
// This is to appease go-critic and allow the call-site to be a switch statement.
//
func isMainToken(s string) bool {
	_, ok := mainTokens[s]
	return ok
}

// Cmd Config Tokens
//
var cmdConfigTokens = map[string]token.Type{
	"SHELL":      TokenConfigShell,
	"USAGE":      TokenConfigUsage,
	"OPTION":     TokenConfigOpt,
	"OPT":        TokenConfigOpt,
	"EXPORT":     TokenConfigExport,
	"ASSERT":     TokenConfigAssert,
	"RUN":        TokenConfigRunBefore,
	"RUN.BEFORE": TokenConfigRunBefore,
	"RUN.AFTER":  TokenConfigRunAfter,
	"RUN.ENV":    TokenConfigRunEnv,
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

func isAlphaNumUnderDot(r rune) bool {
	return (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_' || r == '.'
}

func isAlphaNumUnderDash(r rune) bool {
	return (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_' || r == '-'
}

func isHash(r rune) bool {
	return r == runeHash
}

func isQMark(r rune) bool {
	return r == runeQMark
}

func isBang(r rune) bool {
	return r == runeBang
}

func isDotOrBang(r rune) bool {
	return r == runeDot || r == runeBang
}

// isSpaceOrTab matches tab or space
//
func isSpaceOrTab(r rune) bool {
	return r == runeSpace || r == runeTab
}

func isPrintNonSpace(r rune) bool {
	return unicode.IsPrint(r) && !unicode.IsSpace(r)
}

func isPrintNonReturn(r rune) bool {
	return unicode.IsPrint(r) && r != '\r' && r != '\n'
}

func isConfigOptExample(r rune) bool {
	return unicode.IsPrint(r) && r != '\r' && r != '\n' && r != '\t' && r != '<' && r != '>'
}

func isPrintNonSQuote(r rune) bool {
	return r != runeSQuote && unicode.IsPrint(r)
}

func isPrintNonDQuoteNonBackslashNonDollar(r rune) bool {
	return r != runeDQuote && r != runeBackSlash && r != runeDollar && unicode.IsPrint(r)
}

func isPrintNonParenNonBackslash(r rune) bool {
	return r != runeLParen && r != runeRParen && r != runeBackSlash && unicode.IsPrint(r)
}

func isPrintNonBracketNonBackslashNonSpace(r rune) bool {
	return r != runeLBracket && r != runeRBracket && r != ' ' && r != runeBackSlash && unicode.IsPrint(r)
}

func isPrintNonParenNonBackslashNonSpace(r rune) bool {
	return r != runeLParen && r != runeRParen && r != ' ' && r != runeBackSlash && unicode.IsPrint(r)
}

func isPrintNonBackslashNonDollarNonReturn(r rune) bool {
	return r != runeBackSlash && r != runeDollar && isPrintNonReturn(r)
}

// tryPeekRune tries to peek the next rune
//
func tryPeekRune(l *lexer.Lexer) (rune, bool) {
	if l.CanPeek(1) {
		return l.Peek(1), true
	}
	return utf8.RuneError, false
}

func peekRuneEquals(l *lexer.Lexer, r rune) bool {
	return l.CanPeek(1) && l.Peek(1) == r
}

func nextIfRuneEquals(l *lexer.Lexer, r rune) bool {
	if !l.CanPeek(1) || l.Peek(1) != r {
		return false
	}
	l.Next()
	return true
}

func expectRune(l *lexer.Lexer, r rune, msg string) {
	if !l.CanPeek(1) || l.Peek(1) != r {
		l.EmitError(msg)
		return
	}
	l.Next()
}
