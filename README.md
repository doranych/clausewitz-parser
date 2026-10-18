# clausewitz-parser

A Go parser for the text format of Paradox games (Clausewitz and Jomini engines), such as Stellaris saves. The library uses only the standard library; tests use testify.

```sh
go get github.com/doranych/clausewitz-parser
```

## Usage

```go
root, err := clausewitz.Parse(data, "galactic_object", "country")
if err != nil {
	return fmt.Errorf("parse gamestate: %w", err)
}
for _, sys := range root.Get("galactic_object").Kids {
	if string(sys.Val) == "none" {
		continue
	}
	starbases, err := sys.Get("starbases").IDs()
	...
}
```

`Parse` returns a tree of `Node`s:

- `Key` is nil for list elements; keys may repeat, so children keep file order.
- `Val` holds a scalar, or a header such as `rgb` in `color = rgb { 1 2 3 }`.
- `Kids` is nil for scalars and non-nil for containers, including empty ones.

With keys given, other top-level entries are skipped without building nodes. Nodes share memory with `data`, so do not modify it while the tree is in use.

Values are read with `Get`, `All`, `Str`, `Bool`, `Int`, `Float`, `ID`, `IDs` and `Date`. An empty reference is written as `4294967295`, available as `NoID`.

## Supported syntax

- Pairs, lists, mixed containers, repeated keys, several pairs on one line; indentation is ignored.
- Quoted strings with `\"` escapes, comments, `;` as whitespace, UTF-8 BOM.
- `rgb`, `hsv`, `hsv360`, `cylindrical`, `hex` and `LIST` headers.
- Malformed files the games accept: stray `}`, containers left open at the end of the file.

Not supported yet: comparison operators (`<`, `>=`, `?=`), `[[parameters]]`, object templates, Windows-1252 text.

## Tests

```sh
go test ./...
STELLARIS_SAVES=~/saves go test -run Golden ./...   # golden test on the saves listed in golden_test.go
```

Many cases are ported from [jomini](https://github.com/rakaly/jomini) (MIT, see `testdata/jomini/LICENSE`).

## AI disclosure

The code, tests and documentation in this repository are written with Claude, Anthropic's AI model, through Claude Code. Every change goes through a pull request that the maintainer reviews and merges.

## Licence

MIT, see `LICENSE`.
