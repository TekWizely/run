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
	TokenLBrace
	TokenRBrace
	TokenVarRefStart
	TokenLParen
	TokenRParen
	TokenSubCmdStart

	TokenExport
	TokenCommand

	TokenHashLine

	TokenConfigShell
	TokenConfigDescLine
	TokenConfigDescEnd
	TokenConfigUsage
	TokenConfigOpt
	TokenConfigOptName
	TokenConfigOptShort
	TokenConfigOptLong
	TokenConfigOptValue
	tokenConfigOptEnd
	TokenConfigExport

	TokenConfigEnd

	TokenScriptLine
	TokenScriptEnd
)
