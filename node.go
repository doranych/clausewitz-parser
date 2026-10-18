package clausewitz

import (
	"bytes"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// NoID is the value the games write for an empty reference, 4294967295.
// Compare the result of ID and the elements of IDs with it before following a reference.
const NoID = math.MaxUint32

// Node is one entry of a parsed file: a scalar or a container.
//
//   - A scalar has Val set and Kids nil: foo=bar gives Key "foo", Val "bar".
//   - A container has Kids non-nil, possibly empty: foo={} gives Key "foo" and Kids of length 0.
//   - A container with a header also has Val set: color=rgb { 1 2 3 } gives Key "color", Val "rgb" and three elements.
//   - A list element has Key nil: in ships={ 0 1 } both children are list elements.
//
// Kids keep file order. Keys may repeat, as in planet=654 planet=655, so a container is a list rather than a map:
// use Get for the first match and All for every match.
type Node struct {
	// Key is the entry's key without quotes, or nil for a list element. A quoted empty key "" is empty but not nil.
	Key []byte
	// Val is the scalar without quotes, as written in the file: escapes are not resolved.
	// For a container it is the header, or nil if there is none.
	Val []byte
	// Quoted reports whether Val was a quoted string, which tells "123" from 123 and "rgb" from the rgb header.
	Quoted bool
	// Kids are the entries of a container in file order; nil for a scalar.
	Kids []*Node
}

// Get returns the first child whose key equals key, or nil if there is none.
// Keys compare byte for byte and are case-sensitive; quoted and bare keys compare equal.
// Get on a nil node returns nil, so lookups can be chained: root.Get("country").Get("0").Get("name").
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

// All returns every child whose key equals key, in file order, or nil if there is none.
// Keys compare as in Get. All on a nil node returns nil.
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

// Str returns Val as a string ready for display:
//
//   - trailing spaces, tabs and line breaks are trimmed;
//   - a backslash is dropped and the byte after it kept, so \" becomes " and \\ becomes \;
//   - bytes that are not valid UTF-8 are replaced with U+FFFD.
//
// Text in Windows-1252, used by some older games, is not decoded. For a container Str returns its header or "".
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

// Bool returns true for yes and false for no. Any other value, including an empty one, is an error.
func (n *Node) Bool() (bool, error) {
	switch string(n.Val) {
	case "yes":
		return true, nil
	case "no":
		return false, nil
	}
	return false, fmt.Errorf("parse %q as bool: want yes or no", n.Val)
}

// Int parses Val as a base 10 integer with an optional + or - sign.
// A value outside the int64 range or with other characters returns a *strconv.NumError.
func (n *Node) Int() (int64, error) {
	return strconv.ParseInt(string(n.Val), 10, 64)
}

// Float parses Val as strconv.ParseFloat does, and also accepts a trailing f as in 0.5f or 10.f.
// Forms such as .5, +0.5 and 1.000 are accepted; an empty value, f alone or 1.8.3 return a *strconv.NumError.
// Like strconv.ParseFloat it also accepts exponents, inf and nan, which the games do not write.
func (n *Node) Float() (float64, error) {
	v := n.Val
	if len(v) > 1 && v[len(v)-1] == 'f' {
		v = v[:len(v)-1]
	}
	return strconv.ParseFloat(string(v), 64)
}

// ID parses Val as a reference to another object: an unsigned base 10 integer that fits in uint32, with no sign.
// The games encode an index and a generation in one number, as in 16777221 (0x01000005); ID returns it as is.
// An empty reference parses as NoID. Other values return a *strconv.NumError.
func (n *Node) ID() (uint32, error) {
	id, err := strconv.ParseUint(string(n.Val), 10, 32)
	return uint32(id), err
}

// IDs parses a container of references such as starbases={ 1526 1527 }, keeping their order.
// Every child must be a list element scalar; a scalar node, a keyed child or a nested container is an error,
// as is any element ID rejects. An empty container gives an empty, non-nil slice.
func (n *Node) IDs() ([]uint32, error) {
	if n.Kids == nil {
		return nil, fmt.Errorf("parse %q as id list: not a container", n.Key)
	}
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

// Date is a game date as written in files. Year can be negative or zero.
type Date struct {
	Year  int16
	Month uint8
	Day   uint8
}

// Date parses Val in the form year.month.day, as in 2623.07.01 or 1444.11.11:
//
//   - year is a base 10 integer with an optional sign that fits in int16;
//   - month is 1 to 12 and day is 1 to 31, with or without leading zeros;
//   - the day is not checked against the month, because Stellaris uses 30-day months and other games the real calendar;
//   - dates with an hour, as in 1936.1.1.12, and dates stored as a single number are errors.
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
