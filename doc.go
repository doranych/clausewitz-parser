// Package clausewitz parses the text format of Paradox games built on the Clausewitz and Jomini engines,
// such as Stellaris saves and game files.
//
// # Format
//
// A file is a sequence of tokens; line breaks and indentation carry no meaning, so several entries may share a line.
//
//   - { and } open and close a container, = joins a key and its value.
//   - A quoted string runs to the next " that is not preceded by a backslash and may contain any byte, including line breaks.
//   - A bare value runs until whitespace or one of ; # { } = ".
//   - Spaces, tabs, line breaks and ; separate tokens. # starts a comment that runs to the end of the line.
//   - A UTF-8 byte order mark at the start of the file is skipped.
//
// Inside a container, and at the top level of the file, the parser recognises these entries:
//
//   - key=value: a keyed scalar.
//   - key={ ... } or key{ ... }: a keyed container.
//   - key=rgb { ... }: a keyed container with a header. The headers are rgb, hsv, hsv360, cylindrical, hex and LIST,
//     written as bare values; anything else, or a quoted "rgb", is an ordinary scalar.
//   - value: a list element, for a value followed by neither = nor {.
//   - { ... }: an anonymous container, as in hyperlane={ {to=677} {to=229} }.
//
// Keys and elements can be mixed in one container, and keys may repeat.
//
// # Malformed input
//
// The games load files that are not well formed, and the parser accepts the same:
//
//   - A } at the top level is ignored.
//   - Containers still open at the end of the file are closed there.
//   - An empty {} is dropped from a container that has keyed entries, where it carries no data.
//     In a container of elements only, such as factions={ {} {} }, it is kept, because its position can matter.
//
// Parse returns an error for:
//
//   - = with no key before it;
//   - a key followed by = and then by }, = or the end of the file;
//   - a quoted string with no closing quote;
//   - containers nested deeper than 256 levels.
//
// # Values
//
// The parser keeps values as the bytes of the file: quotes are removed, escapes are not resolved
// and numbers are not converted. Node methods such as Str, Int and Date convert a value when it is read.
//
// Behaviour on game files follows jomini (https://github.com/rakaly/jomini), the parser behind PDX Tools.
package clausewitz
