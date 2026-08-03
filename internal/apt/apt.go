package apt

import (
	"archive/tar"
	"cmp"
	"compress/gzip"
	"crypto/md5"
	"crypto/sha1"
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
	t.Helper()

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
func CreateTestAptRepo(t *testing.T, debs []Deb) string {
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

	assert.ErrorIs(t, cmp.Or(
		os.MkdirAll(dists, 0700),                     //nolint:mnd
		os.MkdirAll(filepath.Join(root, bins), 0700), //nolint:mnd
	), nil)

	distFile, err := os.Create(filepath.Join(dists, "Packages"))
	assert.ErrorIs(t, err, nil)

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
		assert.ErrorIs(t, err, nil)

		m := md5.New()
		s1 := sha1.New()
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
		assert.ErrorIs(t, err, nil)
	}

	assert.ErrorIs(t, distFile.Close(), nil)

	return root
}

const emptyTarGz = "\x1f\x8b\x08\x00\x00\x00\x00\x00" +
	"\x02\x03\x63\x60\x18\x05\xa3\x60" +
	"\x14\x8c\x54\x00\x00\x2e\xaf\xb5" +
	"\xef\x00\x04\x00\x00"

func createDebFile(t *testing.T, w io.Writer, control string) {
	t.Helper()

	arw := ar.NewWriter(w)

	assert.ErrorIs(t, arw.WriteHeader(&ar.Header{
		Name:    "debian-binary",
		ModTime: time.Now(),
		Mode:    0644, //nolint:mnd
		Size:    4,    //nolint:mnd
	}), nil)
	assert.ErrorIs(t, writeString(arw, "2.0\n"), nil)
	assert.ErrorIs(t, arw.WriteHeader(&ar.Header{
		Name:    "control.tar.gz",
		ModTime: time.Now(),
		Mode:    0644, //nolint:mnd
		Size:    ar.UnknownSize,
	}), nil)

	g := gzip.NewWriter(arw)
	tr := tar.NewWriter(g)

	assert.ErrorIs(t, tr.WriteHeader(&tar.Header{
		Name:    "control",
		Mode:    0644, //nolint:mnd
		Size:    int64(len(control)),
		ModTime: time.Now(),
	}), nil)

	assert.ErrorIs(t, writeString(tr, control), nil)
	assert.ErrorIs(t, tr.Close(), nil)
	assert.ErrorIs(t, g.Close(), nil)
	assert.ErrorIs(t, arw.WriteHeader(&ar.Header{
		Name:    "data.tar.gz",
		ModTime: time.Now(),
		Mode:    0644, //nolint:mnd
		Size:    int64(len(emptyTarGz)),
	}), nil)
	assert.ErrorIs(t, writeString(arw, emptyTarGz), nil)
	assert.ErrorIs(t, arw.Close(), nil)
}

func writeString(w io.Writer, str string) error {
	_, err := io.WriteString(w, str)

	return err
}
