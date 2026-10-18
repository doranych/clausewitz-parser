package clausewitz

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseSaveFeatures(t *testing.T) {
	tests := []struct {
		name string
		in   string
		keys []string
		want []*Node
	}{
		{
			name: "several pairs on one line separated by tabs",
			in:   "modules={\n0=arkship_laser_battery\t\t\t\t1=arkship_grand_archive\n}",
			want: []*Node{obj("modules", kv("0", "arkship_laser_battery"), kv("1", "arkship_grand_archive"))},
		},
		{
			name: "indentation drops to zero mid-section",
			in:   "colony={\n\t\tplanet=1\nplanet=2\n\t}",
			want: []*Node{obj("colony", kv("planet", "1"), kv("planet", "2"))},
		},
		{
			name: "repeated keys keep file order",
			in:   "planet=654 planet=655 owner=1 planet=656",
			want: []*Node{kv("planet", "654"), kv("planet", "655"), kv("owner", "1"), kv("planet", "656")},
		},
		{
			name: "anonymous objects in a list",
			in:   "hyperlane={ {to=677 length=42} {to=229 length=37} }",
			want: []*Node{obj("hyperlane", obj("", kv("to", "677"), kv("length", "42")), obj("", kv("to", "229"), kv("length", "37")))},
		},
		{
			name: "empty anonymous objects in a list are elements",
			in:   "factions={\n\t{\n\t}\n \n\t{\n\t}\n}",
			want: []*Node{obj("factions", obj(""), obj(""))},
		},
		{
			name: "keyless numbers",
			in:   "starbases={ 1526 } ships={ 0 }",
			want: []*Node{obj("starbases", item("1526")), obj("ships", item("0"))},
		},
		{
			name: "deleted objects",
			in:   "184549403=none 184549404={ name=x }",
			want: []*Node{kv("184549403", "none"), obj("184549404", kv("name", "x"))},
		},
		{
			name: "tabs between equals and a quoted date",
			in:   "last_date_at_war=\t\t\"2617.10.01\"",
			want: []*Node{qkv("last_date_at_war", "2617.10.01")},
		},
		{
			name: "flag colours with null",
			in:   `colors={ "yellow" "black" "null" "null" }`,
			want: []*Node{obj("colors", qitem("yellow"), qitem("black"), qitem("null"), qitem("null"))},
		},
		{
			name: "quoted header is a plain value",
			in:   `name="rgb" x={ 1 }`,
			want: []*Node{qkv("name", "rgb"), obj("x", item("1"))},
		},
		{
			name: "selected top-level keys only",
			in:   "version=\"v4.5.2\" ships={ 1={ fleet=2 } } galactic_object={ 0={ name=\"Sol\" } } flag=rgb { 1 2 3 } { 4 } 5 country={ 0=none }",
			keys: []string{"galactic_object", "country"},
			want: []*Node{obj("galactic_object", obj("0", qkv("name", "Sol"))), obj("country", kv("0", "none"))},
		},
		{
			name: "selected key with unclosed skipped section",
			in:   "country={ 0=none } ships={ 1={",
			keys: []string{"country"},
			want: []*Node{obj("country", kv("0", "none"))},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root, err := Parse([]byte(tc.in), tc.keys...)
			require.NoError(t, err)
			require.Equal(t, tc.want, root.Kids)
		})
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		name string
		in   string
		keys []string
	}{
		{name: "equals without key", in: "=1"},
		{name: "key without value at end", in: "a="},
		{name: "unterminated string", in: `a="b`},
		{name: "unterminated string in skipped section", in: `a={ b="c }`, keys: []string{"x"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse([]byte(tc.in), tc.keys...)
			require.Error(t, err)
		})
	}
}

func TestIDs(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    []uint32
		wantErr bool
	}{
		{name: "list", in: "starbases={ 1526 16777221 }", want: []uint32{1526, 16777221}},
		{name: "empty", in: "starbases={ }", want: []uint32{}},
		{name: "no reference", in: "starbases={ 4294967295 }", want: []uint32{NoID}},
		{name: "too large", in: "starbases={ 4294967296 }", wantErr: true},
		{name: "negative", in: "starbases={ -1 }", wantErr: true},
		{name: "nested object", in: "starbases={ { 1 } }", wantErr: true},
		{name: "keyed element", in: "starbases={ a=1 }", wantErr: true},
		{name: "scalar", in: "starbases=1", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root, err := Parse([]byte(tc.in))
			require.NoError(t, err)
			got, err := root.Get("starbases").IDs()
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestDateBounds(t *testing.T) {
	tests := []struct {
		in      string
		wantErr bool
	}{
		{in: "2623.07.01"},
		{in: "2200.12.31"},
		{in: "1444.0.1", wantErr: true},
		{in: "1444.13.1", wantErr: true},
		{in: "1444.1.0", wantErr: true},
		{in: "1444.1.32", wantErr: true},
		{in: "1444..1", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			_, err := qitem(tc.in).Date()
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func FuzzParse(f *testing.F) {
	for _, s := range []string{"a=b", "a={b c {d=e}}", `a="\"b\""`, "a=rgb{1 2 3}", "}}{{", "#c\na=1"} {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = Parse(data)
		_, _ = Parse(data, "a")
	})
}
