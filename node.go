package clausewitz

import (
	"bytes"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// NoID is how the games write an empty reference.
const NoID = math.MaxUint32

// Node is a scalar (Val set, Kids nil) or a container (Kids non-nil); keys may repeat, so children stay in file order.
type Node struct {
	Key    []byte // nil for list elements
	Val    []byte // scalar value, or the header of a container such as rgb { 1 2 3 }
	Quoted bool
	Kids   []*Node
}

// Get returns the first child with key, or nil; it is safe on a nil node so lookups can be chained.
func (n *Node) Get(key string) *Node {
	if n == nil {
		return nil
	}
	for _, k := range n.Kids {
		if string(k.Key) == key {
			return k
		}
	}
	return nil
}

// All returns every child with key.
func (n *Node) All(key string) []*Node {
	if n == nil {
		return nil
	}
	var out []*Node
	for _, k := range n.Kids {
		if string(k.Key) == key {
			out = append(out, k)
		}
	}
	return out
}

// Str decodes Val as UTF-8 with escapes resolved and trailing whitespace trimmed.
func (n *Node) Str() string {
	v := bytes.TrimRight(n.Val, " \t\r\n")
	if bytes.IndexByte(v, '\\') >= 0 {
		buf := make([]byte, 0, len(v))
		for i := 0; i < len(v); i++ {
			if v[i] == '\\' && i+1 < len(v) {
				i++
			}
			buf = append(buf, v[i])
		}
		v = buf
	}
	return strings.ToValidUTF8(string(v), "�")
}

func (n *Node) Bool() (bool, error) {
	switch string(n.Val) {
	case "yes":
		return true, nil
	case "no":
		return false, nil
	}
	return false, fmt.Errorf("parse %q as bool: want yes or no", n.Val)
}

func (n *Node) Int() (int64, error) {
	return strconv.ParseInt(string(n.Val), 10, 64)
}

// Float also accepts the f suffix used in game files, as in 0.5f.
func (n *Node) Float() (float64, error) {
	v := n.Val
	if len(v) > 1 && v[len(v)-1] == 'f' {
		v = v[:len(v)-1]
	}
	return strconv.ParseFloat(string(v), 64)
}

// ID parses a reference; compare the result with NoID.
func (n *Node) ID() (uint32, error) {
	id, err := strconv.ParseUint(string(n.Val), 10, 32)
	return uint32(id), err
}

// IDs parses a list of references such as starbases={ 1526 1527 }.
func (n *Node) IDs() ([]uint32, error) {
	out := make([]uint32, 0, len(n.Kids))
	for _, k := range n.Kids {
		if k.Key != nil || k.Kids != nil {
			return nil, fmt.Errorf("parse %q as id list: element is not a scalar", n.Key)
		}
		id, err := k.ID()
		if err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, nil
}

type Date struct {
	Year  int16
	Month uint8
	Day   uint8
}

// Date parses y.m.d; Stellaris months have 30 days, other games up to 31.
func (n *Node) Date() (Date, error) {
	parts := strings.Split(string(n.Val), ".")
	if len(parts) != 3 {
		return Date{}, fmt.Errorf("parse %q as date: want y.m.d", n.Val)
	}
	y, err := strconv.ParseInt(parts[0], 10, 16)
	if err != nil {
		return Date{}, fmt.Errorf("parse %q as date: %w", n.Val, err)
	}
	m, err := strconv.ParseUint(parts[1], 10, 8)
	if err != nil || m < 1 || m > 12 {
		return Date{}, fmt.Errorf("parse %q as date: bad month", n.Val)
	}
	d, err := strconv.ParseUint(parts[2], 10, 8)
	if err != nil || d < 1 || d > 31 {
		return Date{}, fmt.Errorf("parse %q as date: bad day", n.Val)
	}
	return Date{Year: int16(y), Month: uint8(m), Day: uint8(d)}, nil
}
