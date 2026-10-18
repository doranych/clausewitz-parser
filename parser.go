// Package clausewitz parses the text format of Paradox games (Clausewitz and Jomini engines).
package clausewitz

import (
	"bytes"
	"fmt"
	"slices"
)

// Bounds recursion so hostile input cannot exhaust the stack.
const maxDepth = 256

var bom = []byte("\xef\xbb\xbf")

// Container headers such as rgb { 1 2 3 }; anything else before { is a key.
var headers = map[string]bool{"rgb": true, "hsv": true, "hsv360": true, "cylindrical": true, "hex": true, "LIST": true}

type parser struct {
	lex  lexer
	keep map[string]bool
}

// Parse builds the tree of data, skipping top-level entries not in keys when any are given; nodes share memory with data.
func Parse(data []byte, keys ...string) (*Node, error) {
	p := parser{lex: lexer{data: bytes.TrimPrefix(data, bom)}}
	if len(keys) > 0 {
		p.keep = make(map[string]bool, len(keys))
		for _, k := range keys {
			p.keep[k] = true
		}
	}
	root := &Node{Kids: []*Node{}}
	if err := p.parseKids(root, 0); err != nil {
		return nil, err
	}
	return root, nil
}

func (p *parser) parseKids(parent *Node, depth int) error {
	if depth > maxDepth {
		return fmt.Errorf("nesting deeper than %d levels at offset %d", maxDepth, p.lex.pos)
	}
	filter := depth == 0 && p.keep != nil
	for {
		tok, err := p.lex.next()
		if err != nil {
			return err
		}
		switch tok.kind {
		case tokEOF:
			// Unclosed containers end with the file, as the games read them.
			dropEmptyAnon(parent)
			return nil
		case tokClose:
			if depth == 0 {
				continue
			}
			dropEmptyAnon(parent)
			return nil
		case tokEq:
			return fmt.Errorf("unexpected = at offset %d", p.lex.pos-1)
		case tokOpen:
			if filter {
				if err := p.skipBlock(); err != nil {
					return err
				}
				continue
			}
			n := &Node{Kids: []*Node{}}
			if err := p.parseKids(n, depth+1); err != nil {
				return err
			}
			parent.Kids = append(parent.Kids, n)
		case tokBare, tokQuoted:
			nxt, err := p.lex.peek()
			if err != nil {
				return err
			}
			if nxt.kind != tokEq && nxt.kind != tokOpen {
				if !filter {
					parent.Kids = append(parent.Kids, &Node{Val: tok.val, Quoted: tok.kind == tokQuoted})
				}
				continue
			}
			if nxt.kind == tokEq {
				p.lex.next()
			}
			n, err := p.parseValue(tok.val, depth, filter && !p.keep[string(tok.val)])
			if err != nil {
				return err
			}
			if n != nil {
				parent.Kids = append(parent.Kids, n)
			}
		}
	}
}

// parseValue reads the value after key; with skip set it consumes the value and returns nil.
func (p *parser) parseValue(key []byte, depth int, skip bool) (*Node, error) {
	tok, err := p.lex.next()
	if err != nil {
		return nil, err
	}
	n := &Node{Key: key}
	switch tok.kind {
	case tokOpen:
	case tokBare, tokQuoted:
		n.Val, n.Quoted = tok.val, tok.kind == tokQuoted
		if tok.kind == tokQuoted || !headers[string(tok.val)] {
			break
		}
		nxt, err := p.lex.peek()
		if err != nil {
			return nil, err
		}
		if nxt.kind == tokOpen {
			p.lex.next()
			tok = nxt
		}
	default:
		return nil, fmt.Errorf("missing value for %q at offset %d", key, p.lex.pos)
	}
	if tok.kind != tokOpen {
		if skip {
			return nil, nil
		}
		return n, nil
	}
	if skip {
		return nil, p.skipBlock()
	}
	n.Kids = []*Node{}
	return n, p.parseKids(n, depth+1)
}

// skipBlock consumes tokens up to the } that closes an already opened {.
func (p *parser) skipBlock() error {
	for depth := 1; depth > 0; {
		tok, err := p.lex.next()
		if err != nil {
			return err
		}
		switch tok.kind {
		case tokOpen:
			depth++
		case tokClose:
			depth--
		case tokEOF:
			return nil
		}
	}
	return nil
}

// dropEmptyAnon removes {} from containers with keys: there it is noise, while in a list it is a real element.
func dropEmptyAnon(n *Node) {
	if !slices.ContainsFunc(n.Kids, func(k *Node) bool { return k.Key != nil }) {
		return
	}
	n.Kids = slices.DeleteFunc(n.Kids, func(k *Node) bool {
		return k.Key == nil && k.Val == nil && k.Kids != nil && len(k.Kids) == 0
	})
}
