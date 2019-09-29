package main

import (
	"github.com/tekwizely/go-parsing/lexer"
)

// We define our lexer tokens starting from the pre-defined START token
//
const (
	tokenNewline = lexer.TStart + iota

	tokenID
	tokenDotID

	tokenColon
	tokenComma
	tokenEquals      // '=' | ':='
	tokenQMarkEquals // ?=

	tokenDQuote
	tokenDQStringStart

	tokenSQuote
	tokenSQStringStart

	tokenRunes
	tokenEscapeSequence

	tokenHash
	tokenDollar
	tokenLBrace
	tokenRBrace
	tokenVarRefStart
	tokenLParen
	tokenRParen
	tokenSubCmdStart

	tokenExport
	tokenCommand

	tokenHashLine

	tokenConfigShell
	tokenConfigDescEnd
	tokenConfigUsage
	tokenConfigOpt
	tokenConfigOptName
	tokenConfigOptShort
	tokenConfigOptLong
	tokenConfigOptValue
	tokenConfigOptEnd
	tokenConfigExport

	tokenConfigEnd

	tokenScriptLine
)
