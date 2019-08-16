package main

import "github.com/tekwizely/go-parsing/lexer"

type runeFn func(rune) bool

// matchRune
//
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

// matchRuneOrNone
//
func matchRuneOrNone(l *lexer.Lexer, runes ...rune) bool {
	matchRune(l, runes...)
	return true
}

// matchRuneOrEOF
//
func matchRuneOrEOF(l *lexer.Lexer, runes ...rune) bool {
	return !l.CanPeek(1) || matchRune(l, runes...)
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

// ignoreEmptyLines
//
func ignoreEmptyLines(l *lexer.Lexer) {
	for {
		m := l.Marker()
		matchZeroOrMore(l, isSpaceOrTab)
		if matchRuneOrEOF(l, runeNewline) {
			if len(l.PeekToken()) > 0 {
				l.Clear()
			} else {
				return
			}
		} else {
			m.Apply()
			return
		}
	}
}

// matchLeadingSpace
//
func matchLeadingSpace(l *lexer.Lexer) bool {
	return matchOneOrMore(l, isSpaceOrTab)
}

// ignoreLeadingSpace
//
func ignoreLeadingSpace(l *lexer.Lexer) {
	matchZeroOrMore(l, isSpaceOrTab)
	l.Clear()
}

// ignoreSpace
//
func ignoreSpace(l *lexer.Lexer) {
	matchZeroOrMore(l, isSpaceOrTab)
	l.Clear()
}
