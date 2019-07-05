package main

import (
	"strings"
)

// normalizeCmdText
// Removes leading and trailing lines that are empty or whitespace only.
// Removes all leading whitepace that matches leading whitespace on first non-empty line
//
func normalizeCmdText(txt []string) []string {
	// Remove empty leading lines
	//
	for isLineWhitespaceOnly(txt[0]) {
		txt = txt[1:]
	}
	// Remove empty trailing lines
	//
	for isLineWhitespaceOnly(txt[len(txt)-1]) {
		txt = txt[:len(txt)-1]
	}
	// Still have anything?
	//
	if len(txt) > 0 {
		// Leading whitespace on first line is considered as indention-only
		//
		runes := []rune(txt[0])
		i := 0
		for isWhitespace(runes[i]) {
			i++
		}
		// Any leading ws?
		//
		if i > 0 {
			leadingWS := string(runes[:i])
			for i, line := range txt {
				if strings.HasPrefix(line, leadingWS) {
					txt[i] = line[len(leadingWS):]
				}
			}
		}

	}
	return txt
}

// isLineWhitespaceOnly return true if the input contains ONLY (' ' | '\t' | '\n' | '\r')
//
func isLineWhitespaceOnly(line string) bool {

	for _, r := range line {
		// TODO Consider using a more liberal whitespace check ( i.e unicode.IsSpace() )
		if !isWhitespace(r) {
			return false
		}
	}
	return true
}

// isWhitespace return true if the input is one of (' ' | '\t' | '\n' | '\r')
//
func isWhitespace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\r'
}
