package apt

import (
	"archive/tar"
	"cmp"
	"compress/gzip"
	"crypto/md5"  //nolint:gosec
	"crypto/sha1" //nolint:gosec
	"crypto/sha256"
	"crypto/sha512"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/erikgeiser/ar"
	"github.com/stretchr/testify/assert"
	"vimagination.zapto.org/rwcount"
)

// BuildBase is the ubuntu docker container used for testing.
const BuildBase = "docker://ubuntu:resolute-20260413"

// SkipIfBadEnvironment will skip remaining tests if the environment does not
// contain singularity and mksquashfs.
func SkipIfBadEnvironment(t *testing.T) {
	if _, err := exec.LookPath("singularity"); errors.Is(err, exec.ErrNotFound) {
		t.Skip("skipping due to no singularity executable")
	} else if _, err = exec.LookPath("mksquashfs"); errors.Is(err, exec.ErrNotFound) {
		t.Skip("skipping due to no mksquashfs executable")
	}
}

// Deb represents the basic metadata for a deb file.
type Deb struct {
	Name     string
	Version  string
	Metadata map[string]string
}

// CreateTestAptRepo creates a simple on-disk APT repository, from the supplied
// deb file metadata, that can be used with the build package.
func CreateTestAptRepo(t *testing.T, debs []Deb) string { //nolint:funlen
	t.Helper()
	root := t.TempDir()

	slices.SortFunc(debs, func(a, b Deb) int {
		return cmp.Or(
			strings.Compare(a.Name, b.Name),
			strings.Compare(a.Version, b.Version),
		)
	})

	bins := filepath.Join("pool", "main", "binary-"+runtime.GOARCH)
	dists := filepath.Join(root, "dists", "resolute", "main", "binary-"+runtime.GOARCH)

	assert.NoError(t, cmp.Or(
		os.MkdirAll(dists, 0700),                     //nolint:mnd
		os.MkdirAll(filepath.Join(root, bins), 0700), //nolint:mnd
	))

	distFile, err := os.Create(filepath.Join(dists, "Packages"))
	assert.NoError(t, err)

	for n, deb := range debs {
		debPath := filepath.Join(bins, strconv.Itoa(n)+".deb")

		var control strings.Builder

		fmt.Fprintf(
			&control,
			"Package: %s\nArchitecture: all\nVersion: %s\nFilename: %s\n",
			deb.Name,
			deb.Version,
			debPath,
		)

		for key, value := range deb.Metadata {
			fmt.Fprintln(&control, key, ": ", value)
		}

		f, err := os.Create(filepath.Join(root, debPath))
		assert.NoError(t, err)

		m := md5.New()   //nolint:gosec
		s1 := sha1.New() //nolint:gosec
		s2 := sha256.New()
		s5 := sha512.New()

		w := rwcount.Writer{Writer: f}

		createDebFile(t, io.MultiWriter(&w, m, s1, s2, s5), control.String())

		_, err = fmt.Fprintf(
			distFile,
			"%sFilename: %s\nSize: %d\nMD5sum: %x\nSHA1: %x\nSHA256: %x\nSHA512: %x\n\n",
			control.String(),
			debPath,
			w.Count,
			m.Sum(nil),
			s1.Sum(nil),
			s2.Sum(nil),
			s5.Sum(nil),
		)
		assert.NoError(t, err)
	}

	assert.NoError(t, distFile.Close())

	return root
}

const emptyTarGz = "\x1f\x8b\x08\x00\x00\x00\x00\x00" +
	"\x02\x03\x63\x60\x18\x05\xa3\x60" +
	"\x14\x8c\x54\x00\x00\x2e\xaf\xb5" +
	"\xef\x00\x04\x00\x00"

func createDebFile(t *testing.T, w io.Writer, control string) { //nolint:funlen
	t.Helper()

	arw := ar.NewWriter(w)

	assert.NoError(t, arw.WriteHeader(&ar.Header{
		Name:    "debian-binary",
		ModTime: time.Now(),
		Mode:    0644, //nolint:mnd
		Size:    4,    //nolint:mnd
	}))
	assert.NoError(t, writeString(arw, "2.0\n"), nil)
	assert.NoError(t, arw.WriteHeader(&ar.Header{
		Name:    "control.tar.gz",
		ModTime: time.Now(),
		Mode:    0644, //nolint:mnd
		Size:    ar.UnknownSize,
	}))

	g := gzip.NewWriter(arw)
	tr := tar.NewWriter(g)

	assert.NoError(t, tr.WriteHeader(&tar.Header{
		Name:    "control",
		Mode:    0644, //nolint:mnd
		Size:    int64(len(control)),
		ModTime: time.Now(),
	}))

	assert.NoError(t, writeString(tr, control))
	assert.NoError(t, tr.Close())
	assert.NoError(t, g.Close())
	assert.NoError(t, arw.WriteHeader(&ar.Header{
		Name:    "data.tar.gz",
		ModTime: time.Now(),
		Mode:    0644, //nolint:mnd
		Size:    int64(len(emptyTarGz)),
	}))
	assert.NoError(t, writeString(arw, emptyTarGz))
	assert.NoError(t, arw.Close())
}

func writeString(w io.Writer, str string) error {
	_, err := io.WriteString(w, str)

	return err
}

// ExamplePackages returns a simple selection of possible packages that can be
// used with CreateTestAptRepo to create a simple APT repo that can be used for
// testing.
func ExamplePackages() []Deb { //nolint:funlen
	return []Deb{
		{
			Name:    "abc", //nolint:goconst
			Version: "1",
			Metadata: map[string]string{
				"XB-Executables": "abc", //nolint:goconst
			},
		},
		{
			Name:    "abc", //nolint:goconst
			Version: "2",
			Metadata: map[string]string{
				"XB-Executables": "abc, def", //nolint:goconst
			},
		},
		{
			Name:    "python", //nolint:goconst
			Version: "3.13",
			Metadata: map[string]string{
				"XB-Executables": "python, python3.13", //nolint:goconst
			},
		},
		{
			Name:    "py-xyz",
			Version: "2.1",
			Metadata: map[string]string{
				"Depends": "python", //nolint:goconst
			},
		},
		{
			Name:    "r",
			Version: "4.4.0",
			Metadata: map[string]string{
				"XB-Executables": "R, Rscript", //nolint:goconst
			},
		},
		{
			Name:    "r-lib",
			Version: "1.1",
			Metadata: map[string]string{
				"Depends": "r",
			},
		},
	}
}
