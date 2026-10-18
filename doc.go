// Package clausewitz parses the text format of Paradox games built on the Clausewitz and Jomini engines,
// such as Stellaris saves and game script files.
//
// # Format
//
// A file is a sequence of tokens; line breaks and indentation carry no meaning, so several entries may share a line.
//
//   - { and } open and close a container.
//   - = joins a key and its value. Script files also use the operators <, <=, >, >=, !=, ?= and ==.
//   - A quoted string runs to the next " that is not preceded by a backslash and may contain any byte, including line breaks.
//   - A bare value runs until whitespace or one of ; # { } [ ] = < > ! ". A ? ends it only as part of ?=.
//   - A bare value that starts with @[ or @\[ runs to the next ], so inline maths such as @[ 1 - x ] is one value.
//   - Spaces, tabs, line breaks and ; separate tokens. # starts a comment that runs to the end of the line.
//   - A UTF-8 byte order mark at the start of the file is skipped.
//
// Inside a container, and at the top level of the file, the parser recognises these entries:
//
//   - key=value or key<value with any operator: a keyed scalar.
//   - key={ ... } or key{ ... }: a keyed container. An operator may stand in place of =, as in key>{ ... }.
//   - key=rgb { ... }: a keyed container with a header. The headers are rgb, hsv, hsv360, cylindrical, hex and LIST,
//     written as bare values; anything else, or a quoted "rgb", is an ordinary scalar.
//   - value: a list element, for a value followed by neither an operator nor {.
//   - { ... }: an anonymous container, as in hyperlane={ {to=677} {to=229} }.
//   - { ... }=value: an object template; the template and its value become two list elements, as in jomini.
//   - [[name] ... ] and [[!name] ... ]: a parameter block, whose content the game uses only when a scripted effect
//     or trigger is called with, or without, that parameter. A block holding a single value, as in [[name] $name$ ],
//     becomes a scalar; any other block becomes a container.
//
// Keys and elements can be mixed in one container, and keys may repeat. Variables such as @scale and
// script values such as value:x|A|b| are ordinary bare values.
//
// # Malformed input
//
// The games load files that are not well formed, and the parser accepts the same:
//
//   - A } at the top level is ignored.
//   - Containers still open at the end of the file are closed there.
//   - An empty {} is dropped from a container that has keyed entries, where it carries no data.
//     In a container of elements only, such as factions={ {} {} }, it is kept, because its position can matter.
//   - An operator where a key is expected gives its first character as a key, so =="bar" is the key = with the value bar,
//     and a lone = is a list element.
//
// Parse returns an error for:
//
//   - a key followed by an operator and then by }, ], another operator or the end of the file;
//   - a quoted string with no closing quote, or @[ with no closing ];
//   - a ! not followed by =, or a single [;
//   - a ] outside a parameter block, a } that would close a parameter block, or a parameter block left open at the end of the file;
//   - containers nested deeper than 256 levels.
//
// # Values
//
// The parser keeps values as the bytes of the file: quotes are removed, escapes are not resolved
// and numbers are not converted. Node methods such as Str, Int and Date convert a value when it is read.
//
// Behaviour on game files follows jomini (https://github.com/rakaly/jomini), the parser behind PDX Tools,
// except that list elements are accepted at the top level, as in Stellaris tag files.
package clausewitz
