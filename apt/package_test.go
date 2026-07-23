package apt

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
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
			io.WriteString(w, secondPackages)
		} else {
			doneFirst = true

			io.WriteString(w, firstPackages)
		}
	}))

	t.Cleanup(srv.Close)

	p, err := New(srv.URL, time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectation := []Package{
		{
			Name:        "py-torch",
			Description: "big lib",
			Versions:    []string{"2.0.0"},
		},
	}

	if pkgs := p.GetAllPackages(); !reflect.DeepEqual(pkgs, expectation) {
		t.Errorf("expecting to get packages %#v, got %#v", expectation, pkgs)
	}

	time.Sleep(time.Second * 2)

	expectation[0].Versions = append(expectation[0].Versions, "2.0.1")

	if pkgs := p.GetAllPackages(); !reflect.DeepEqual(pkgs, expectation) {
		t.Errorf("expecting to get packages %#v, got %#v", expectation, pkgs)
	}
}

func TestGetRecipeDescription(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, testPackages)
	}))

	s, err := New(srv.URL, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if desc, err := s.GetRecipeDescription("py-torch"); err != nil {
		t.Errorf("unexpected error: %v", err)
	} else if desc != "big lib" {
		t.Errorf("expecting description %q, got %q", "big lib", desc)
	}

	if desc, err := s.GetRecipeDescription("r-ggplot2"); err != nil {
		t.Errorf("unexpected error: %v", err)
	} else if desc != "ggplot library" {
		t.Errorf("expecting description %q, got %q", "ggplot library", desc)
	}

	if desc, err := s.GetRecipeDescription("system-lib"); !errors.Is(err, ErrPackageNotFound) {
		t.Errorf("expecting error %v, got %v", ErrPackageNotFound, err)
	} else if desc != "" {
		t.Errorf("expecting blank description, got %q", desc)
	}

}
