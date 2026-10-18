package clausewitz

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Cases ported from jomini at e26ee3f (https://github.com/rakaly/jomini, MIT, see testdata/jomini/LICENSE); names point to the source test.

func kv(k, v string) *Node  { return &Node{Key: []byte(k), Val: []byte(v)} }
func qkv(k, v string) *Node { return &Node{Key: []byte(k), Val: []byte(v), Quoted: true} }
func item(v string) *Node   { return &Node{Val: []byte(v)} }
func qitem(v string) *Node  { return &Node{Val: []byte(v), Quoted: true} }

func obj(k string, kids ...*Node) *Node { return hdr(k, "", kids...) }

func hdr(k, h string, kids ...*Node) *Node {
	n := &Node{Kids: []*Node{}}
	if k != "" {
		n.Key = []byte(k)
	}
	if h != "" {
		n.Val = []byte(h)
	}
	n.Kids = append(n.Kids, kids...)
	return n
}

func TestJominiTape(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    []*Node
		wantErr bool
	}{
		{name: "tape.rs/test_simple_event", in: "foo=bar", want: []*Node{kv("foo", "bar")}},
		{name: "tape.rs/test_simple_event_with_spaces", in: "  \t\t foo =bar \r\ndef=\tqux", want: []*Node{kv("foo", "bar"), kv("def", "qux")}},
		{name: "tape.rs/test_scalars_with_quotes", in: `"foo"="bar" "3"="1444.11.11"`, want: []*Node{qkv("foo", "bar"), qkv("3", "1444.11.11")}},
		{name: "tape.rs/test_scalars_with_quoteas_delimiters", in: `"foo"="bar"3="1444.11.11"`, want: []*Node{qkv("foo", "bar"), qkv("3", "1444.11.11")}},
		{name: "tape.rs/test_escaped_quotes", in: `name = "Joe \"Captain\" Rogers"`, want: []*Node{qkv("name", `Joe \"Captain\" Rogers`)}},
		{name: "tape.rs/test_escaped_quotes_short", in: `name = "J Rogers \"a"`, want: []*Node{qkv("name", `J Rogers \"a`)}},
		{name: "tape.rs/test_escaped_quotes_crazy", in: `custom_name="THE !@#$%^&*( '\"LEGION\"')"`, want: []*Node{qkv("custom_name", `THE !@#$%^&*( '\"LEGION\"')`)}},
		{name: "tape.rs/test_quotes_with_escape_sequences", in: "custom_name=\"ab \x15D ( ID: 691 )\x15!\"", want: []*Node{qkv("custom_name", "ab \x15D ( ID: 691 )\x15!")}},
		{name: "tape.rs/test_numbers_are_scalars", in: "foo=1.000", want: []*Node{kv("foo", "1.000")}},
		{name: "tape.rs/test_object_event", in: "foo={bar=qux}", want: []*Node{obj("foo", kv("bar", "qux"))}},
		{name: "tape.rs/test_object_multi_field_event", in: "foo={bar=1 qux=28}", want: []*Node{obj("foo", kv("bar", "1"), kv("qux", "28"))}},
		{name: "tape.rs/test_array_event", in: "versions={\r\n\t\"1.28.3.0\"\r\n}", want: []*Node{obj("versions", qitem("1.28.3.0"))}},
		{name: "tape.rs/test_array_multievent", in: "versions={\r\n\t\"1.28.3.0\"\r\n foo}", want: []*Node{obj("versions", qitem("1.28.3.0"), item("foo"))}},
		{name: "tape.rs/test_no_equal_object_event", in: "foo{bar=qux}", want: []*Node{obj("foo", kv("bar", "qux"))}},
		{
			name: "tape.rs/test_no_equal_object_field",
			in: `dlc002 = {
				name = "Arachnoid Portrait Pack"
				steam_id = "447680"
				gog_store_id = ""
				paradoxplaza_store_url ""
				category = "content_pack"
				show = no
				recommendations = {}
			}`,
			want: []*Node{obj("dlc002",
				qkv("name", "Arachnoid Portrait Pack"),
				qkv("steam_id", "447680"),
				qkv("gog_store_id", ""),
				item("paradoxplaza_store_url"),
				qitem(""),
				qkv("category", "content_pack"),
				kv("show", "no"),
				obj("recommendations"),
			)},
		},
		{name: "tape.rs/test_empty_array", in: "discovered_by={}", want: []*Node{obj("discovered_by")}},
		{
			name: "tape.rs/test_array_of_objects",
			in:   "stats={{id=0 type=general} {id=1 type=admiral}}",
			want: []*Node{obj("stats", obj("", kv("id", "0"), kv("type", "general")), obj("", kv("id", "1"), kv("type", "admiral")))},
		},
		{name: "tape.rs/test_empty_objects", in: "history={{} 1629.11.10={core=AAA}}", want: []*Node{obj("history", obj("1629.11.10", kv("core", "AAA")))}},
		{name: "tape.rs/test_empty_multi_objects", in: "history={{} {} 1629.11.10={core=AAA}}", want: []*Node{obj("history", obj("1629.11.10", kv("core", "AAA")))}},
		{name: "tape.rs/test_empty_objects2", in: "foo={bar=val {}} me=you", want: []*Node{obj("foo", kv("bar", "val")), kv("me", "you")}},
		{name: "tape.rs/test_empty_objects3", in: "foo={bar=val { } a=b} me=you", want: []*Node{obj("foo", kv("bar", "val"), kv("a", "b")), kv("me", "you")}},
		{name: "tape.rs/test_spanning_objects", in: "army={name=abc} army={name=def}", want: []*Node{obj("army", kv("name", "abc")), obj("army", kv("name", "def"))}},
		{name: "tape.rs/test_regression", in: "\x00 \"\x00", wantErr: true},
		{name: "tape.rs/test_regression2", in: "\x00\x04!\x00", want: []*Node{item("\x00\x04!\x00")}},
		{name: "tape.rs/test_too_heavily_nested", in: "foo=" + strings.Repeat("{", 100000), wantErr: true},
		{name: "tape.rs/test_no_ws_comment", in: "foo=abc#def\nbar=qux", want: []*Node{kv("foo", "abc"), kv("bar", "qux")}},
		{name: "tape.rs/test_bom", in: "\xef\xbb\xbf#hello", want: []*Node{}},
		{name: "tape.rs/test_period_in_identifiers", in: "flavor_tur.8=yes", want: []*Node{kv("flavor_tur.8", "yes")}},
		{name: "tape.rs/test_dashed_identifiers", in: "dashed-identifier=yes", want: []*Node{kv("dashed-identifier", "yes")}},
		{name: "tape.rs/test_colon_values", in: "province_id = event_target:agenda_province", want: []*Node{kv("province_id", "event_target:agenda_province")}},
		{
			name: "tape.rs/test_many_line_comment",
			in:   "foo=1.000\n" + strings.Repeat("# this is a comment\n", 100000) + "foo=2.000\n",
			want: []*Node{kv("foo", "1.000"), kv("foo", "2.000")},
		},
		{name: "tape.rs/test_terminating_comment", in: "# boo\r\n# baa\r\nfoo=a\r\n# bee", want: []*Node{kv("foo", "a")}},
		{name: "tape.rs/test_rgb_trick", in: "name = rgb ", want: []*Node{kv("name", "rgb")}},
		{name: "tape.rs/test_rgb_trick2", in: "name = rgb type = 4713", want: []*Node{kv("name", "rgb"), kv("type", "4713")}},
		{name: "tape.rs/test_rgb_trick3", in: "name = rgbeffect", want: []*Node{kv("name", "rgbeffect")}},
		{name: "tape.rs/test_rgb", in: "color = rgb { 100 200 150 } ", want: []*Node{hdr("color", "rgb", item("100"), item("200"), item("150"))}},
		{name: "tape.rs/test_hsv_trick", in: "name = hsv ", want: []*Node{kv("name", "hsv")}},
		{name: "tape.rs/test_hsv_trick2", in: "name = hsv type = 4713", want: []*Node{kv("name", "hsv"), kv("type", "4713")}},
		{name: "tape.rs/test_hsv_trick3", in: "name = hsveffect", want: []*Node{kv("name", "hsveffect")}},
		{name: "tape.rs/test_hsv", in: "color = hsv { 0.43 0.86 0.61 } ", want: []*Node{hdr("color", "hsv", item("0.43"), item("0.86"), item("0.61"))}},
		{name: "tape.rs/test_hsv360", in: "color = hsv360{ 25 75 63 }", want: []*Node{hdr("color", "hsv360", item("25"), item("75"), item("63"))}},
		{name: "tape.rs/test_cylindrical", in: "color = cylindrical{ 150 3 0 }", want: []*Node{hdr("color", "cylindrical", item("150"), item("3"), item("0"))}},
		{name: "tape.rs/test_hex", in: "color = hex { aabbccdd }", want: []*Node{hdr("color", "hex", item("aabbccdd"))}},
		{name: "tape.rs/test_list_header", in: "mild_winter = LIST { 3700 3701 }", want: []*Node{hdr("mild_winter", "LIST", item("3700"), item("3701"))}},
		{name: "tape.rs/test_operator_early_eof", in: "a{b=}", wantErr: true},
		{name: "tape.rs/test_extraneous_closing_bracket", in: "foo = { bar } } ", want: []*Node{obj("foo", item("bar"))}},
		{name: "tape.rs/test_extraneous_closing_bracket2", in: "a{} b=r}", want: []*Node{obj("a"), kv("b", "r")}},
		{name: "tape.rs/test_extraneous_closing_bracket3", in: "a=c}}", want: []*Node{kv("a", "c")}},
		{
			name: "tape.rs/test_extraneous_bracket",
			in:   `color = { 121 163 114 } } army_names = {"Armata di $PROVINCE$"}`,
			want: []*Node{obj("color", item("121"), item("163"), item("114")), obj("army_names", qitem("Armata di $PROVINCE$"))},
		},
		{
			name: "tape.rs/test_missing_bracket",
			in:   `BRA_GAR_01 = { ordered = { 1 = { "%da Divisao de Infantaria" } }`,
			want: []*Node{obj("BRA_GAR_01", obj("ordered", obj("1", qitem("%da Divisao de Infantaria"))))},
		},
		{name: "tape.rs/test_unquoted_non_ascii", in: "jean_jaur\xe8s = bar ", want: []*Node{kv("jean_jaur\xe8s", "bar")}},
		{name: "tape.rs/test_text_number_plus", in: "pop_happiness = +0.10", want: []*Node{kv("pop_happiness", "+0.10")}},
		{name: "tape.rs/test_skip_semicolon", in: "value=\"win\"; a=b", want: []*Node{qkv("value", "win"), kv("a", "b")}},
		{name: "tape.rs/test_mixed_container_1", in: "levels={a=b 10}", want: []*Node{obj("levels", kv("a", "b"), item("10"))}},
		{name: "tape.rs/test_mixed_container_2", in: "levels={a=b 10 20}", want: []*Node{obj("levels", kv("a", "b"), item("10"), item("20"))}},
		{name: "tape.rs/test_mixed_container_3", in: "levels={a=b 10 c=d}", want: []*Node{obj("levels", kv("a", "b"), item("10"), kv("c", "d"))}},
		{name: "tape.rs/test_mixed_container_4", in: "levels={a=b 10 c=d 20}", want: []*Node{obj("levels", kv("a", "b"), item("10"), kv("c", "d"), item("20"))}},
		{
			name: "tape.rs/test_mixed_container_5",
			in:   "16778374={ levels={ 10 0=2 1=2 } }",
			want: []*Node{obj("16778374", obj("levels", item("10"), kv("0", "2"), kv("1", "2")))},
		},
		{
			name: "tape.rs/test_mixed_container_6",
			in:   "levels={ 10 0=2 1=2 } foo={bar=qux}",
			want: []*Node{obj("levels", item("10"), kv("0", "2"), kv("1", "2")), obj("foo", kv("bar", "qux"))},
		},
		{
			name: "tape.rs/test_mixed_container_7",
			in: `brittany_area = { #5
				color = { 118  99  151 }
				169 170 171 172 4384
			}`,
			want: []*Node{obj("brittany_area",
				obj("color", item("118"), item("99"), item("151")),
				item("169"), item("170"), item("171"), item("172"), item("4384"),
			)},
		},
		{
			name: "tape.rs/test_mixed_container_8",
			in: `brittany_area = { #5
				color = { 118  99  151 }
				169
			}`,
			want: []*Node{obj("brittany_area", obj("color", item("118"), item("99"), item("151")), item("169"))},
		},
		{name: "tape.rs/test_mixed_container_9", in: "a {{b=c d c}}", want: []*Node{obj("a", obj("", kv("b", "c"), item("d"), item("c")))}},
		{
			name: "tape.rs/test_mixed_container_10",
			in: `on_actions = {
				faith_holy_order_land_acquisition_pulse
				delay = { days = { 5 10 }}
				faith_heresy_events_pulse
				delay = { days = { 15 20 }}
				faith_fervor_events_pulse
			}`,
			want: []*Node{obj("on_actions",
				item("faith_holy_order_land_acquisition_pulse"),
				obj("delay", obj("days", item("5"), item("10"))),
				item("faith_heresy_events_pulse"),
				obj("delay", obj("days", item("15"), item("20"))),
				item("faith_fervor_events_pulse"),
			)},
		},
		{
			name: "tape.rs/test_dont_fail_on_list_keyword",
			in: `  simple_cross_flag = {
				pattern = list christian_emblems_list
				color1 = list normal_colors
			}`,
			want: []*Node{obj("simple_cross_flag",
				kv("pattern", "list"), item("christian_emblems_list"),
				kv("color1", "list"), item("normal_colors"),
			)},
		},
		{name: "tape.rs/test_initial_end_does_not_panic", in: "}", want: []*Node{}},
		{name: "tape.rs/test_semicolon_as_whitespace", in: "foo = 0.3;", want: []*Node{kv("foo", "0.3")}},
		{name: "tape.rs/test_multiple_semicolons_as_whitespace", in: "a = 1; b = 2;; c = 3;", want: []*Node{kv("a", "1"), kv("b", "2"), kv("c", "3")}},
		{
			name: "dom.rs/text_reader_read_fields",
			in:   "name=aaa name=bbb core=123 core=456 name=ccc name=ddd",
			want: []*Node{kv("name", "aaa"), kv("name", "bbb"), kv("core", "123"), kv("core", "456"), kv("name", "ccc"), kv("name", "ddd")},
		},
		{
			name: "dom.rs/text_reader_read_fields_nested",
			in:   "army={name=aaa unit={name=bbb} unit={name=ccc}} army={name=ddd unit={name=eee}}",
			want: []*Node{
				obj("army", kv("name", "aaa"), obj("unit", kv("name", "bbb")), obj("unit", kv("name", "ccc"))),
				obj("army", kv("name", "ddd"), obj("unit", kv("name", "eee"))),
			},
		},
		{
			name: "dom.rs/test_issue_168",
			in: `has_leviathans = {
				optimize_memory
				host_has_dlc = "Leviathans Story Pack"
			}`,
			want: []*Node{obj("has_leviathans", item("optimize_memory"), qkv("host_has_dlc", "Leviathans Story Pack"))},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root, err := Parse([]byte(tc.in))
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, root.Kids)
		})
	}
}

func TestJominiDomLookups(t *testing.T) {
	root, err := Parse([]byte("army={name=aaa unit={name=bbb} unit={name=ccc}} army={name=ddd unit={name=eee}}"))
	require.NoError(t, err)

	tests := []struct {
		name  string
		nodes []*Node
		want  []string
	}{
		{name: "dom.rs/text_reader_read_fields_nested armies", nodes: root.All("army"), want: []string{"aaa", "ddd"}},
		{name: "dom.rs/text_reader_read_fields_nested first army units", nodes: root.Get("army").All("unit"), want: []string{"bbb", "ccc"}},
		{name: "dom.rs/text_reader_read_fields_nested second army units", nodes: root.All("army")[1].All("unit"), want: []string{"eee"}},
		{name: "missing key", nodes: root.Get("navy").All("unit"), want: nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var got []string
			for _, n := range tc.nodes {
				got = append(got, n.Get("name").Str())
			}
			require.Equal(t, tc.want, got)
		})
	}
}

func TestJominiFloat(t *testing.T) {
	tests := []struct {
		in      string
		want    float64
		wantErr bool
	}{
		{in: "-1", want: -1},
		{in: "0", want: 0},
		{in: "1", want: 1},
		{in: "-10000", want: -10000},
		{in: "10000", want: 10000},
		{in: "20405029", want: 20405029},
		{in: "-20405029", want: -20405029},
		{in: "20405029553322", want: 20405029553322},
		{in: "-20405029553322", want: -20405029553322},
		{in: "0.504", want: 0.504},
		{in: "-0.504", want: -0.504},
		{in: ".504", want: 0.504},
		{in: "-.504", want: -0.504},
		{in: "1.00125", want: 1.00125},
		{in: "-1.50000", want: -1.5},
		{in: "-10000.0", want: -10000},
		{in: "10000.000", want: 10000},
		{in: "20405029.125", want: 20405029.125},
		{in: "-20405029.125", want: -20405029.125},
		{in: "20405029553322.015", want: 20405029553322.015},
		{in: "-20405029553322.015", want: -20405029553322.015},
		{in: "10.99999999999999", want: 10.99999999999999},
		{in: "+0.5", want: 0.5},
		{in: "E", wantErr: true},
		{in: "", wantErr: true},
		{in: "1.8.3", wantErr: true},
		{in: "0.0f", want: 0},
		{in: "-5.5f", want: -5.5},
		{in: "10.0f", want: 10},
		{in: "0.40f", want: 0.4},
		{in: "123.456f", want: 123.456},
		{in: "-0.001f", want: -0.001},
		{in: "+42.0f", want: 42},
		{in: ".5f", want: 0.5},
		{in: "1f", want: 1},
		{in: "-1f", want: -1},
		{in: "10.f", want: 10},
		{in: "0f", want: 0},
		{in: "f", wantErr: true},
		{in: "invalidf", wantErr: true},
	}
	for _, tc := range tests {
		t.Run("scalar.rs/"+tc.in, func(t *testing.T) {
			got, err := item(tc.in).Float()
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestJominiInt(t *testing.T) {
	tests := []struct {
		in      string
		want    int64
		wantErr bool
	}{
		{in: "0", want: 0},
		{in: "1", want: 1},
		{in: "-1", want: -1},
		{in: "-10000", want: -10000},
		{in: "10000", want: 10000},
		{in: "20405029", want: 20405029},
		{in: "-20405029", want: -20405029},
		{in: "20405029553322", want: 20405029553322},
		{in: "-20405029553322", want: -20405029553322},
		{in: "+0", want: 0},
		{in: "+1", want: 1},
		{in: "9223372036854775807", want: 9223372036854775807},
		{in: "90071992547409097", want: 90071992547409097},
		{in: "-90071992547409097", want: -90071992547409097},
		{in: "-9223372036854775809", wantErr: true},
		{in: "9223372036854775808", wantErr: true},
		{in: "", wantErr: true},
	}
	for _, tc := range tests {
		t.Run("scalar.rs/"+tc.in, func(t *testing.T) {
			got, err := item(tc.in).Int()
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestJominiBool(t *testing.T) {
	tests := []struct {
		in      string
		want    bool
		wantErr bool
	}{
		{in: "yes", want: true},
		{in: "no", want: false},
		{in: "", wantErr: true},
		{in: "1", wantErr: true},
	}
	for _, tc := range tests {
		t.Run("scalar.rs/"+tc.in, func(t *testing.T) {
			got, err := item(tc.in).Bool()
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestJominiStr(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "", want: ""},
		{in: "new\n", want: "new"},
		{in: "\t", want: ""},
		{in: `new \"Captain\" ` + "\n", want: `new "Captain"`},
		{in: "newø ", want: "newø"},
		{in: `Joe \"Captain\" Rogers`, want: `Joe "Captain" Rogers`},
		{in: `Joe Captain\"s`, want: `Joe Captain"s`},
		{in: "Joe\xffcheeze", want: "Joe�cheeze"},
	}
	for _, tc := range tests {
		t.Run("encoding.rs/"+tc.in, func(t *testing.T) {
			require.Equal(t, tc.want, qitem(tc.in).Str())
		})
	}
}

func TestJominiDate(t *testing.T) {
	tests := []struct {
		in      string
		want    Date
		wantErr bool
	}{
		{in: "1400.1.2", want: Date{Year: 1400, Month: 1, Day: 2}},
		{in: "1.01.01", want: Date{Year: 1, Month: 1, Day: 1}},
		{in: "1444.11.30", want: Date{Year: 1444, Month: 11, Day: 30}},
		{in: "-17.1.1", want: Date{Year: -17, Month: 1, Day: 1}},
		{in: "-2500.1.1", want: Date{Year: -2500, Month: 1, Day: 1}},
		{in: "-10000.1.1", want: Date{Year: -10000, Month: 1, Day: 1}},
		{in: "1936.01.01", want: Date{Year: 1936, Month: 1, Day: 1}},
		{in: "2205.02.30", want: Date{Year: 2205, Month: 2, Day: 30}},
		{in: "2200.01.01", want: Date{Year: 2200, Month: 1, Day: 1}},
		{in: "1444.257.1", wantErr: true},
		{in: "1444.1.257", wantErr: true},
		{in: "60000.1.1", wantErr: true},
		{in: "-60000.1.1", wantErr: true},
		{in: "1936.1.1.0", wantErr: true},
		{in: "05.5.3`.3", wantErr: true},
		{in: "43808760", wantErr: true},
	}
	for _, tc := range tests {
		t.Run("date.rs/"+tc.in, func(t *testing.T) {
			got, err := qitem(tc.in).Date()
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}
