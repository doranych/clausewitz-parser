# clausewitz-parser

A Go parser for the text format of Paradox games (Clausewitz and Jomini engines), such as Stellaris saves and game script files. The library uses only the standard library; tests use testify.

```sh
go get github.com/doranych/clausewitz-parser
```

This is a personal project powered by coffee and passion. There's no long-term support commitment, but any PRs are welcome.

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
- `Op` holds an operator other than `=`, as in `count < 2`; `Param` marks a `[[name] ... ]` block.

With keys given, other top-level entries are skipped without building nodes. Nodes share memory with `data`, so do not modify it while the tree is in use.

Values are read with `Get`, `All`, `Str`, `Bool`, `Int`, `Float`, `ID`, `IDs` and `Date`. An empty reference is written as `4294967295`, available as `NoID`.

## Supported syntax

- Pairs, lists, mixed containers, repeated keys, several pairs on one line; indentation is ignored.
- Quoted strings with `\"` escapes, comments, `;` as whitespace, UTF-8 BOM.
- `rgb`, `hsv`, `hsv360`, `cylindrical`, `hex` and `LIST` headers.
- Script syntax: operators `<`, `<=`, `>`, `>=`, `!=`, `?=`, `==`, `@variables`, inline maths `@[ ... ]`, `[[parameter]]` blocks, object templates.
- Malformed files the games accept: stray `}`, containers left open at the end of the file.

Not supported: Windows-1252 text, used by some older games.

## Tests

```sh
go test ./...
```

The golden test parses a real Stellaris 4.5.2 save from `testdata/saves`.

Many cases are ported from [jomini](https://github.com/rakaly/jomini) (MIT, see `testdata/jomini/LICENSE`).

## Acknowledgements

Thanks to [PDX Tools](https://pdx.tools) and its parser [jomini](https://github.com/rakaly/jomini). Its tests document years of quirks in Paradox files, and this parser's behaviour on malformed input follows it.

## AI disclosure

The code, tests and documentation in this repository are written with Claude, Anthropic's AI model, through Claude Code. Every change goes through a pull request that the maintainer reviews and merges.

## Licence

MIT, see `LICENSE`.
