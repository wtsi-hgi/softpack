package apt

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/wtsi-hgi/softpack/db"
)

const firstPackages = `
Package: py-torch
Architecture: amd64
Version: 2.0.0
Filename: pool/main/p/py-torch-2.0.0.deb
XB-Softpack: true
Description: big lib

`

const secondPackages = firstPackages + `
Package: py-torch
Architecture: amd64
Version: 2.0.1
Filename: pool/main/p/py-torch-2.0.1.deb
XB-Softpack: true
Description: big lib
`

func TestNew(t *testing.T) {
	var doneFirst bool

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if doneFirst {
			io.WriteString(w, secondPackages) //nolint:errcheck
		} else {
			doneFirst = true

			io.WriteString(w, firstPackages) //nolint:errcheck
		}
	}))

	t.Cleanup(srv.Close)

	p, err := New(srv.URL, time.Second)
	assert.NoError(t, err)

	expectation := []Package{
		{
			Name:        "py-torch",        //nolint:goconst
			Description: "big lib",         //nolint:goconst
			Versions:    []string{"2.0.0"}, //nolint:goconst
		},
	}

	assert.Equal(t, p.GetAllPackages(), expectation)

	time.Sleep(time.Second * 2)

	expectation[0].Versions = append(expectation[0].Versions, "2.0.1")

	assert.Equal(t, p.GetAllPackages(), expectation)
}

func TestGetRecipeDescription(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, testPackages) //nolint:errcheck
	}))

	t.Cleanup(srv.Close)

	s, err := New(srv.URL, 0)
	assert.NoError(t, err)

	desc, err := s.GetRecipeDescription("py-torch")
	assert.NoError(t, err)
	assert.Equal(t, desc, "big lib")

	desc, err = s.GetRecipeDescription("r-ggplot2")
	assert.NoError(t, err)
	assert.Equal(t, desc, "ggplot library")

	desc, err = s.GetRecipeDescription("system-lib")
	assert.Equal(t, err, ErrInvalidPackage)
	assert.Equal(t, desc, "")
}

func TestCheckPkgsExist(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, testPackages) //nolint:errcheck
	}))

	t.Cleanup(srv.Close)

	s, err := New(srv.URL, time.Second)
	assert.NoError(t, err)

	expected := db.Package{
		Name:        "py-torch",
		Description: "big lib",
		Version:     "2.0.0",
	}

	// TODO: This will fail if the expected pkg only has one of the versions, should it?

	assert.True(t, s.CheckPackageExists(expected))
	assert.True(t, s.CheckPackagesExist([]db.Package{expected}))
}
