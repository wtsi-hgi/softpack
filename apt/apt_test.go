package apt

import (
	"cmp"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

const testPackages = `
Package: py-torch
Architecture: amd64
Version: 2.0.0
Filename: pool/main/p/py-torch-2.0.0.deb
XB-Softpack: true
Description: big lib

Package: r-ggplot2
Architecture: amd64
Version: 1.2.3
Filename: pool/main/r/r-ggplot2-1.2.3.deb
XB-Softpack: true
Description: ggplot library

Package: r-ggplot2
Architecture: amd64
Version: 1.2.4
Filename: pool/main/r/r-ggplot2-1.2.4.deb
XB-Softpack: true
Description: ggplot library

Package: system-lib
Architecture: amd64
Version: 1.2.4
Filename: pool/main/s/system-lib.deb
Description: system package

Package: py-torch
Architecture: amd64
Version: 2.0.1
Filename: pool/main/p/py-torch-2.0.1.deb
XB-Softpack: true
Description: big lib
`

func TestReadIndex(t *testing.T) {
	expectations := []Package{
		{
			Name:        "py-torch",
			Description: "big lib",
			Versions:    []string{"2.0.0", "2.0.1"},
		},
		{
			Name:        "r-ggplot2",
			Description: "ggplot library",
			Versions:    []string{"1.2.3", "1.2.4"},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, testPackages)
	}))

	filePath := filepath.Join(t.TempDir(), "Packages.gz")

	f, err := os.Create(filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	g := gzip.NewWriter(f)

	_, err = io.WriteString(g, testPackages)

	if e := cmp.Or(err, g.Close(), f.Close()); e != nil {
		t.Fatalf("unexpected error: %v", e)
	}

	httpResult, err := readIndex(srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	fileResult, err := readIndex(filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !reflect.DeepEqual(httpResult, expectations) {
		t.Errorf("http result did not match: expecting %#v, got %#v", expectations, httpResult)
	}

	if !reflect.DeepEqual(fileResult, expectations) {
		t.Errorf("file result did not match: expecting %#v, got %#v", expectations, fileResult)
	}
}

func TestReadS3(t *testing.T) {
	url := os.Getenv("S3URL")
	if url == "" {
		t.Skip("set S3URL to enable S3 test")
	}

	pkgs, err := readS3Index(url)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	} else if len(pkgs) == 0 {
		t.Error("got not packages")
	}
}
