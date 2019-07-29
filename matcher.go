package main

import "github.com/tekwizely/go-parsing/lexer"

type runeFn func(rune) bool

func matchRune(l *lexer.Lexer, runes ...rune) bool {
	if p, ok := tryPeekRune(l); ok {
		for _, r := range runes {
			if r == p {
				l.Next()
				return true
			}
		}
	}
	return false
}
func matchRuneOrNone(l *lexer.Lexer, runes ...rune) bool {
	matchRune(l, runes...)
	return true
}

// func matchZeroOrOne(l *lexer.Lexer, fn runeFn) bool {
// 	if l.CanPeek(1) && fn(l.Peek(1)) {
// 		l.Next()
// 	}
// 	return true
// }
func matchZeroOrMore(l *lexer.Lexer, fn runeFn) bool {
	for l.CanPeek(1) && fn(l.Peek(1)) {
		l.Next()
	}
	return true
}
func matchOne(l *lexer.Lexer, fn runeFn) bool {
	if l.CanPeek(1) && fn(l.Peek(1)) {
		l.Next()
		return true
	}
	return false
}
func matchOneOrMore(l *lexer.Lexer, fn runeFn) bool {
	b := false
	for l.CanPeek(1) && fn(l.Peek(1)) {
		l.Next()
		b = true
	}
	return b
}
