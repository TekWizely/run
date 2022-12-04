package lexer

import "io"

// readerIgnoreCR wraps a RuneReader, filtering out '\r'.
// Useful for input sources that use '\r'+'\n' for end-of-line.
//
type readerIgnoreCR struct {
	r io.RuneReader
}

// newReaderIgnoreCR is a convenience method.
//
func newReaderIgnoreCR(r io.RuneReader) io.RuneReader {
	return &readerIgnoreCR{r: r}
}

// ReadRune implements io.RuneReader
//
func (c *readerIgnoreCR) ReadRune() (r rune, size int, err error) {
	r, size, err = c.r.ReadRune()
	for size == 1 && r == '\r' {
		r, size, err = c.r.ReadRune()
	}
	return
}
