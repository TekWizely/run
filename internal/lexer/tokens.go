package lexer

import (
	"github.com/tekwizely/go-parsing/lexer"
)

// We define our lexer tokens starting from the pre-defined START token
//
const (
	TokenNewline = lexer.TStart + iota

	TokenNotNewline // Meta token

	TokenID
	TokenDotID
	TokenDashID
	TokenCommandDefID

	TokenAt          // '@'
	TokenBang        // '!'
	TokenColon       // ':'
	TokenComma       // ','
	TokenEquals      // '=' | ':='
	TokenQMarkEquals // ?=

	TokenDQuote // '"'
	TokenDQStringStart

	TokenSQuote // "'"
	TokenSQStringStart

	TokenRunes
	TokenEscapeSequence

	TokenDollar // '$'
	TokenVarRefStart
	TokenSubCmdStart

	TokenLParen   // '('
	TokenRParen   // ')'
	TokenLBrace   // '{'
	TokenRBrace   // '}'
	TokenLBracket // '['
	TokenRBracket // ']'

	TokenParenStringStart  // '( '
	TokenParenStringEnd    // ' )'
	TokenDParenStringStart // '(( '
	TokenDParenStringEnd   // ' ))'

	TokenBracketStringStart  // '[ '
	TokenBracketStringEnd    // ' ]'
	TokenDBracketStringStart // '[[ '
	TokenDBracketStringEnd   // ' ]]'

	TokenExport
	TokenAs
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
	TokenConfigOptExample
	TokenConfigExport
	TokenConfigAssert
	TokenConfigRunBefore
	TokenConfigRunAfter

	TokenConfigEnd

	TokenScriptLine
	TokenScriptEnd

	TokenEmptyAssertMessage

	TokenUnknownRune
)
