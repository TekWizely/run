package lexer

import "github.com/tekwizely/go-parsing/lexer"

type runeFn func(rune) bool

// isRune accepts a rune and returns a predicate suitable for match* functions.
//
func isRune(r rune) runeFn {
	return func(r_ rune) bool { return r_ == r }
}

// matchRune attempts to match the next rune to one specified, returning success or failure.
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

// matchRuneOrNone attempts to match the next rune to one specified, returning success regardless.
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

func matchZeroOrOne(l *lexer.Lexer, fn runeFn) bool {
	if l.CanPeek(1) && fn(l.Peek(1)) {
		l.Next()
	}
	return true
}

// matchZeroOrMore attempts to match zero or more of the specified predicate, ruturning succcess regardless.
//
func matchZeroOrMore(l *lexer.Lexer, fn runeFn) bool {
	for l.CanPeek(1) && fn(l.Peek(1)) {
		l.Next()
	}
	return true
}

// matchOne attempts to match one or more of the specified predicate, returning success or failure.
//
func matchOne(l *lexer.Lexer, fn runeFn) bool {
	if l.CanPeek(1) && fn(l.Peek(1)) {
		l.Next()
		return true
	}
	return false
}

// matchOneOrMore attempts to match one or more of the specified predicate, returning success or failure.
//
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

		if matchNewlineOrEOF(l) {
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

// ignoreSpace matches one or more isSpaceOrTab and discards any matches.
//
func ignoreSpace(l *lexer.Lexer) {
	if matchOneOrMore(l, isSpaceOrTab) {
		l.Clear()
	}
}

// ignoreEOL attempts to match newline or EOF, discarding any matches.
//
func ignoreEOL(l *lexer.Lexer) {
	if matchNewlineOrEOF(l) {
		l.Clear()
	}
}
