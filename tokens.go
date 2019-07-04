package main

import (
	"github.com/tekwizely/go-parsing/lexer"
	"github.com/tekwizely/go-parsing/lexer/token"
)

// We define our lexer tokens starting from the pre-defined START token
//
const (
	TokenID token.Type = lexer.TStart + iota

	tokenColon
	tokenLParen
	tokenRParen
	tokenLBrace
	tokenRBrace

	tokenScriptLine
	tokenEndScript
)
