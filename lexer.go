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
	tokEq
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
		switch l.data[l.pos] {
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
		case '=':
			l.pos++
			return token{kind: tokEq}, nil
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
		default:
			start := l.pos
			for l.pos < len(l.data) && !isDelim(l.data[l.pos]) {
				l.pos++
			}
			return token{kind: tokBare, val: l.data[start:l.pos]}, nil
		}
	}
	return token{kind: tokEOF}, nil
}

func isDelim(c byte) bool {
	switch c {
	case ' ', '\t', '\r', '\n', '\v', '\f', ';', '#', '{', '}', '=', '"':
		return true
	}
	return false
}
