package apt

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wtsi-hgi/softpack/internal/apt"
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

Package: r-cran-ggplot2
Architecture: amd64
Version: 1.2.4
Filename: pool/main/r/r-ggplot2-1.2.4.deb
XB-Softpack: true
XB-Alias: r-ggplot2
Description: ggplot library

Package: system-lib
Architecture: amd64
Version: 1.2.4
Filename: pool/main/s/system-lib.deb
Description: system package

Package: py-other
Architecture: amd64
Version: 2.0.1
Filename: pool/main/p/py-torch-2.0.1.deb
XB-Alias: py-torch
XB-Softpack: true
Description: big lib
`

func TestReadIndex(t *testing.T) {
	expectations := []Package{
		{
			Name:        "py-torch",                 //nolint:goconst
			Description: "big lib",                  //nolint:goconst
			Versions:    []string{"2.0.0", "2.0.1"}, //nolint:goconst
		},
		{
			Name:        "r-ggplot2",
			Description: "ggplot library",
			Versions:    []string{"1.2.3", "1.2.4"},
		},
	}
	aliases := map[string]string{"py-torch": "py-other", "r-ggplot2": "r-cran-ggplot2"}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, testPackages) //nolint:errcheck
	}))

	filePath := filepath.Join(t.TempDir(), "Packages.gz")

	f, err := os.Create(filePath)
	assert.NoError(t, err)

	g := gzip.NewWriter(f)

	_, err = io.WriteString(g, testPackages)
	assert.NoError(t, err)
	assert.NoError(t, g.Close())
	assert.NoError(t, f.Close())

	httpResult, httpAliases, err := readIndex(srv.URL)
	assert.NoError(t, err)

	fileResult, fileAliases, err := readIndex(filePath)
	assert.NoError(t, err)

	assert.Equal(t, httpResult, expectations)
	assert.Equal(t, fileResult, expectations)
	assert.Equal(t, httpAliases, aliases)
	assert.Equal(t, fileAliases, aliases)
}

func TestReadS3(t *testing.T) {
	packages := []apt.Deb{
		{
			Name:    "package",
			Version: "1",
			Metadata: map[string]string{
				"Description": "desc1",
				"XB-Softpack": "true",
			},
		},
		{
			Name:    "package2",
			Version: "7",
			Metadata: map[string]string{
				"XB-Softpack": "true",
			},
		},
		{
			Name:    "package",
			Version: "6",
			Metadata: map[string]string{
				"Description": "desc1",
				"XB-Softpack": "true",
			},
		},
	}

	s := apt.MockS3Server(t, packages)
	defer s.Close()

	dists := "s3://" + path.Join("apt", "dists", "resolute", "main", "binary-"+runtime.GOARCH, "Packages")

	pkgs, _, err := readIndex(dists)
	assert.NoError(t, err)
	assert.Equal(t, []Package{
		{
			Name:        "package",
			Description: "desc1",
			Versions:    []string{"1", "6"},
		},
		{
			Name:        "package2",
			Description: "",
			Versions:    []string{"7"},
		}}, pkgs)
}
