package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/sebdah/goldie/v2"
	"github.com/stretchr/testify/require"
)

func init() {
	spew.Config.DisablePointerAddresses = true
}

func TestParseFile(t *testing.T) {
	g := goldie.New(t)

	f, err := os.Open("testdata")
	require.NoError(t, err)
	defer f.Close()

	fis, err := f.Readdir(-1)
	require.NoError(t, err)
	for _, fi := range fis {
		if fi.IsDir() {
			continue
		}

		if filepath.Ext(fi.Name()) == ".golden" {
			continue
		}

		t.Run(fi.Name(), func(t *testing.T) {
			cfg, err := ParseFile(filepath.Join("testdata", fi.Name()))
			require.NoError(t, err)
			g.Assert(t, fi.Name(), []byte(spew.Sdump(cfg)))
		})
	}
}
