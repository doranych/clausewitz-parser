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

// Parse reads data and returns the root container, whose Kids are the top-level entries in file order.
//
// With no keys, every entry is built. With keys, only top-level entries whose key is in keys are built;
// the rest, including top-level list elements and anonymous containers, are skipped. Skipped entries are
// still read token by token, so an unterminated string or a missing value inside them is still an error.
// Keys select top-level entries only; nested entries with the same key are not affected.
// A parameter block [[name] ... ] is selected by its name.
//
// Key and Val of the returned nodes point into data rather than copying it,
// so data must not be modified while the tree is in use.
//
// See the package documentation for the accepted syntax and the cases that return an error.
func Parse(data []byte, keys ...string) (*Node, error) {
	p := parser{lex: lexer{data: bytes.TrimPrefix(data, bom)}}
	if len(keys) > 0 {
		p.keep = make(map[string]bool, len(keys))
		for _, k := range keys {
			p.keep[k] = true
		}
	}
	root := &Node{Kids: []*Node{}}
	if err := p.parseKids(root, 0, tokEOF); err != nil {
		return nil, err
	}
	return root, nil
}

// parseKids reads entries into parent up to closer: tokEOF for the root, tokClose for {, tokCloseParam for [[name].
func (p *parser) parseKids(parent *Node, depth int, closer tokKind) error {
	if depth > maxDepth {
		return fmt.Errorf("nesting deeper than %d levels at offset %d", maxDepth, p.lex.pos)
	}
	top := closer == tokEOF
	filter := top && p.keep != nil
	for {
		tok, err := p.lex.next()
		if err != nil {
			return err
		}
		switch tok.kind {
		case tokEOF:
			if closer == tokCloseParam {
				return fmt.Errorf("unterminated parameter block at offset %d", p.lex.pos)
			}
			// Unclosed containers end with the file, as the games read them.
			dropEmptyAnon(parent)
			return nil
		case tokClose, tokCloseParam:
			if tok.kind == closer {
				dropEmptyAnon(parent)
				return nil
			}
			if top && tok.kind == tokClose {
				continue
			}
			return fmt.Errorf("unexpected closing bracket at offset %d", p.lex.pos-1)
		case tokOpen:
			if filter {
				if err := p.skipBlock(); err != nil {
					return err
				}
				continue
			}
			n := &Node{Kids: []*Node{}}
			if err := p.parseKids(n, depth+1, tokClose); err != nil {
				return err
			}
			parent.Kids = append(parent.Kids, n)
			if err := p.parseTemplateValue(parent, depth); err != nil {
				return err
			}
		case tokParam:
			n := &Node{Key: tok.val, Param: ParamSet, Kids: []*Node{}}
			if tok.val[0] == '!' {
				n.Key, n.Param = tok.val[1:], ParamUnset
			}
			if err := p.parseKids(n, depth+1, tokCloseParam); err != nil {
				return err
			}
			if len(n.Kids) == 1 && n.Kids[0].Key == nil && n.Kids[0].Kids == nil {
				n.Val, n.Quoted, n.Kids = n.Kids[0].Val, n.Kids[0].Quoted, nil
			}
			if !filter || p.keep[string(n.Key)] {
				parent.Kids = append(parent.Kids, n)
			}
		case tokOp:
			// In key position the first character of an operator is a key, as in =="bar"; the rest is read again.
			p.lex.pos -= len(tok.val) - 1
			tok = token{kind: tokBare, val: tok.val[:1]}
			fallthrough
		case tokBare, tokQuoted:
			nxt, err := p.lex.peek()
			if err != nil {
				return err
			}
			if nxt.kind != tokOp && nxt.kind != tokOpen {
				if !filter {
					parent.Kids = append(parent.Kids, &Node{Val: tok.val, Quoted: tok.kind == tokQuoted})
				}
				continue
			}
			var op string
			if nxt.kind == tokOp {
				p.lex.next()
				if string(nxt.val) != "=" {
					op = string(nxt.val)
				}
			}
			n, err := p.parseValue(tok.val, op, depth, filter && !p.keep[string(tok.val)])
			if err != nil {
				return err
			}
			if n != nil {
				parent.Kids = append(parent.Kids, n)
			}
		}
	}
}

// parseTemplateValue reads the value of an object template such as { a=b }={ 1 2 } and adds it after the template.
func (p *parser) parseTemplateValue(parent *Node, depth int) error {
	nxt, err := p.lex.peek()
	if err != nil || nxt.kind != tokOp || string(nxt.val) != "=" {
		return err
	}
	p.lex.next()
	tok, err := p.lex.next()
	if err != nil {
		return err
	}
	n := &Node{Val: tok.val, Quoted: tok.kind == tokQuoted}
	switch tok.kind {
	case tokBare, tokQuoted:
	case tokOpen:
		n.Kids = []*Node{}
		if err := p.parseKids(n, depth+1, tokClose); err != nil {
			return err
		}
	default:
		return fmt.Errorf("missing value for template at offset %d", p.lex.pos)
	}
	parent.Kids = append(parent.Kids, n)
	return nil
}

// parseValue reads the value after key and op; with skip set it consumes the value and returns nil.
func (p *parser) parseValue(key []byte, op string, depth int, skip bool) (*Node, error) {
	tok, err := p.lex.next()
	if err != nil {
		return nil, err
	}
	n := &Node{Key: key, Op: op}
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
	return n, p.parseKids(n, depth+1, tokClose)
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
