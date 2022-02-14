package github

import (
	"encoding/base64"
	"fmt"
	"github.com/go-enry/go-license-detector/v4/licensedb"
	"github.com/go-enry/go-license-detector/v4/licensedb/api"

	"github.com/go-enry/go-license-detector/v4/licensedb/filer"
	"github.com/google/go-github/v18/github"
	"github.com/hpapaxen/golicense/license"
	"github.com/mitchellh/go-spdx"
)

// detect uses go-license-detector as a fallback.
func detect(rl *github.RepositoryLicense) (*license.License, error) {
	ms, err := licensedb.Detect(&filerImpl{License: rl})
	if err != nil {
		return nil, err
	}

	// Find the highest matching license
	var highest api.Match
	current := ""
	for id, v := range ms {
		if v.Confidence > 0.90 && v.Confidence > highest.Confidence {
			highest = v
			current = id
		}
	}

	if current == "" {
		return nil, nil
	}

	// License detection only returns SPDX IDs but we want the complete name.
	lic, err := spdx.License(current)
	if err != nil {
		return nil, fmt.Errorf("error looking up license %q: %s", current, err)
	}

	return &license.License{
		Name: lic.Name,
		SPDX: lic.ID,
	}, nil
}

// filerImpl implements filer.Filer to return the license text directly
// from the github.RepositoryLicense structure.
type filerImpl struct {
	License *github.RepositoryLicense
}

func (f *filerImpl) PathsAreAlwaysSlash() bool {
	return true
}

func (f *filerImpl) ReadFile(name string) ([]byte, error) {
	if name != "LICENSE" {
		return nil, fmt.Errorf("unknown file: %s", name)
	}

	return base64.StdEncoding.DecodeString(f.License.GetContent())
}

func (f *filerImpl) ReadDir(dir string) ([]filer.File, error) {
	// We only support root
	if dir != "" {
		return nil, nil
	}

	return []filer.File{filer.File{Name: "LICENSE"}}, nil
}

func (f *filerImpl) Close() {}
