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
	tokenEquals

	tokenDQuote
	tokenDQStringStart
	// tokenDQStringEnd

	tokenSQuote
	tokenSQStringStart
	// tokenSQStringEnd

	tokenRunes
	tokenEscapeSequence

	tokenDollar
	tokenLBrace
	tokenRBrace
	tokenVarRefStart
	// tokenVarRefEnd
	tokenLParen
	tokenRParen
	tokenSubCmdStart
	// tokenSubCmdEnd

	tokenScriptLine
	tokenScriptEnd

	// tokenShell
	// tokenShort
	// tokenDesc
	// tokenDescLine
	// tokenDescEnd
	// tokenUsage
	// tokenArg
	// tokenArgName
	// tokenArgShort
	// tokenArgLong
	// tokenArgValue
	// tokenArgDesc
)
