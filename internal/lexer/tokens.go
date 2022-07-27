package lexer

import (
	"github.com/tekwizely/go-parsing/lexer"
)

// We define our lexer tokens starting from the pre-defined START token
//
const (
	TokenNewline = lexer.TStart + iota

	TokenID
	TokenDotID
	TokenDashID

	TokenColon
	TokenComma
	TokenEquals      // '=' | ':='
	TokenQMarkEquals // ?=

	TokenDQuote
	TokenDQStringStart

	TokenSQuote
	TokenSQStringStart

	TokenRunes
	TokenEscapeSequence

	TokenDollar
	TokenVarRefStart
	TokenSubCmdStart

	TokenLParen
	TokenRParen
	TokenLBrace
	TokenRBrace
	TokenLBracket
	TokenRBracket

	TokenParenStringStart  // '( '
	TokenParenStringEnd    // ' )'
	TokenDParenStringStart // '(( '
	TokenDParenStringEnd   // ' ))'

	TokenBracketStringStart  // '[ '
	TokenBracketStringEnd    // ' ]'
	TokenDBracketStringStart // '[[ '
	TokenDBracketStringEnd   // ' ]]'

	TokenExport
	TokenAssert
	TokenInclude
	TokenCommand

	TokenHashLine

	TokenConfigShell
	TokenConfigDescLineStart
	TokenConfigDescEnd
	TokenConfigUsage
	TokenConfigOpt
	TokenConfigOptName
	TokenConfigOptShort
	TokenConfigOptLong
	TokenConfigOptValue
	TokenConfigExport
	TokenConfigAssert

	TokenConfigEnd

	TokenScriptLine
	TokenScriptEnd

	TokenEmptyAssertMessage

	TokenUnknownRune
)
