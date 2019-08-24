package main

import (
	"github.com/tekwizely/go-parsing/lexer"
)

// We define our lexer tokens starting from the pre-defined START token
//
const (
	tokenID = lexer.TStart + iota
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

	tokenConfigShell
	tokenConfigDescEnd
	tokenConfigUsage
	tokenConfigOpt
	tokenConfigOptName
	tokenConfigOptShort
	tokenConfigOptLong
	tokenConfigOptValue
	tokenConfigExport

	tokenConfigEnd

	tokenScriptLine
)
