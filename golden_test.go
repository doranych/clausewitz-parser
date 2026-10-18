package clausewitz

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// Saves are not committed; STELLARIS_SAVES names the folder that holds them.
func TestGoldenSaves(t *testing.T) {
	tests := []struct {
		file     string
		version  string
		date     string
		sections map[string]int // children per section
		deleted  map[string]int // children written as id=none
	}{
		{
			file:     "ironman.sav",
			version:  "Cygnus v4.5.2",
			date:     "2623.07.01",
			sections: map[string]int{"galactic_object": 1011, "country": 101, "ships": 10321, "starbase_mgr": 1},
			deleted:  map[string]int{"galactic_object": 0, "country": 16, "ships": 1205, "starbase_mgr": 0},
		},
	}
	for _, tc := range tests {
		t.Run(tc.file, func(t *testing.T) {
			path := filepath.Join(os.Getenv("STELLARIS_SAVES"), tc.file)
			if _, err := os.Stat(path); os.Getenv("STELLARIS_SAVES") == "" || err != nil {
				t.Skipf("save %s not found, set STELLARIS_SAVES", path)
			}
			zr, err := zip.OpenReader(path)
			require.NoError(t, err)
			defer zr.Close()
			f, err := zr.Open("gamestate")
			require.NoError(t, err)
			data, err := io.ReadAll(f)
			require.NoError(t, err)

			full, err := Parse(data)
			require.NoError(t, err)
			require.Equal(t, tc.version, full.Get("version").Str())
			_, err = full.Get("date").Date()
			require.NoError(t, err)
			require.Equal(t, tc.date, full.Get("date").Str())

			keys := make([]string, 0, len(tc.sections))
			for name, want := range tc.sections {
				keys = append(keys, name)
				sec := full.Get(name)
				require.NotNil(t, sec, name)
				require.Len(t, sec.Kids, want, name)
				deleted := 0
				for _, k := range sec.Kids {
					if string(k.Val) == "none" && k.Kids == nil {
						deleted++
					}
				}
				require.Equal(t, tc.deleted[name], deleted, name)
			}

			partial, err := Parse(data, keys...)
			require.NoError(t, err)
			require.Len(t, partial.Kids, len(keys))
			for _, name := range keys {
				require.Equal(t, full.Get(name), partial.Get(name), name)
			}
		})
	}
}
