package clausewitz

import (
	"bytes"
	"fmt"
)

type tokKind uint8

const (
	tokEOF tokKind = iota
	tokOpen
	tokClose
	tokCloseParam
	tokOp
	tokParam
	tokBare
	tokQuoted
)

type token struct {
	kind tokKind
	val  []byte
}

type lexer struct {
	data   []byte
	pos    int
	peeked bool
	buf    token
}

func (l *lexer) next() (token, error) {
	if l.peeked {
		l.peeked = false
		return l.buf, nil
	}
	return l.scan()
}

func (l *lexer) peek() (token, error) {
	if !l.peeked {
		t, err := l.scan()
		if err != nil {
			return t, err
		}
		l.buf, l.peeked = t, true
	}
	return l.buf, nil
}

func (l *lexer) scan() (token, error) {
	for l.pos < len(l.data) {
		c := l.data[l.pos]
		switch c {
		case ' ', '\t', '\r', '\n', '\v', '\f', ';':
			l.pos++
		case '#':
			nl := bytes.IndexByte(l.data[l.pos:], '\n')
			if nl < 0 {
				l.pos = len(l.data)
			} else {
				l.pos += nl + 1
			}
		case '{':
			l.pos++
			return token{kind: tokOpen}, nil
		case '}':
			l.pos++
			return token{kind: tokClose}, nil
		case ']':
			l.pos++
			return token{kind: tokCloseParam}, nil
		case '=', '<', '>', '!', '?':
			start := l.pos
			l.pos++
			if l.pos < len(l.data) && l.data[l.pos] == '=' {
				l.pos++
			} else if c == '!' {
				return token{}, fmt.Errorf("unexpected ! at offset %d", start)
			} else if c == '?' {
				// A lone ? is an ordinary character, as in jomini.
				l.pos = start
				return l.bare(), nil
			}
			return token{kind: tokOp, val: l.data[start:l.pos]}, nil
		case '[':
			return l.param()
		case '"':
			start := l.pos + 1
			for i := start; i < len(l.data); i++ {
				switch l.data[i] {
				case '\\':
					i++
				case '"':
					l.pos = i + 1
					return token{kind: tokQuoted, val: l.data[start:i]}, nil
				}
			}
			return token{}, fmt.Errorf("unterminated string at offset %d", l.pos)
		case '@':
			// Inline maths: @[ 1 + x ], or @\[ ... ] inside scripted effects with $parameters$.
			rest := l.data[l.pos+1:]
			if bytes.HasPrefix(rest, []byte("[")) || bytes.HasPrefix(rest, []byte(`\[`)) {
				end := bytes.IndexByte(l.data[l.pos:], ']')
				if end < 0 {
					return token{}, fmt.Errorf("unterminated @[ at offset %d", l.pos)
				}
				start := l.pos
				l.pos += end + 1
				return token{kind: tokBare, val: l.data[start:l.pos]}, nil
			}
			return l.bare(), nil
		default:
			return l.bare(), nil
		}
	}
	return token{kind: tokEOF}, nil
}

func (l *lexer) bare() token {
	start := l.pos
	l.pos++
	for l.pos < len(l.data) && !isDelim(l.data[l.pos]) {
		l.pos++
	}
	return token{kind: tokBare, val: l.data[start:l.pos]}
}

// param reads the opening [[name] or [[!name] of a parameter block; val keeps the ! of the second form.
func (l *lexer) param() (token, error) {
	start := l.pos
	if l.pos+1 >= len(l.data) || l.data[l.pos+1] != '[' {
		return token{}, fmt.Errorf("unexpected [ at offset %d", start)
	}
	l.pos += 2
	name := l.pos
	if l.pos < len(l.data) && l.data[l.pos] == '!' {
		l.pos++
	}
	first := l.pos
	for l.pos < len(l.data) && !isDelim(l.data[l.pos]) {
		l.pos++
	}
	if l.pos >= len(l.data) || l.data[l.pos] != ']' || l.pos == first {
		return token{}, fmt.Errorf("bad parameter name at offset %d", start)
	}
	l.pos++
	return token{kind: tokParam, val: l.data[name : l.pos-1]}, nil
}

func isDelim(c byte) bool {
	switch c {
	case ' ', '\t', '\r', '\n', '\v', '\f', ';', '#', '{', '}', '[', ']', '=', '<', '>', '!', '"':
		return true
	}
	return false
}
