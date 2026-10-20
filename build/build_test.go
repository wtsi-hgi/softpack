package build

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	_ "embed"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wtsi-hgi/softpack/db"
	"github.com/wtsi-hgi/softpack/internal/apt"
)

//go:embed aptcudf.deb
var aptcudf []byte

func TestBuild(t *testing.T) {
	apt.SkipIfBadEnvironment(t)

	root := apt.CreateTestAptRepo(t, apt.ExamplePackages())

	addAptCudf(t, root)

	install := t.TempDir()

	t.Log("Build using FS source")

	arts, err := Build(apt.BuildBase(t), t.TempDir(), install, "some-wrapper", root, []db.Package{
		{Name: "abc"}, //nolint:goconst
	})
	assert.NoError(t, err)

	assert.Equal(t, arts.Exes, []string{"abc", "def"})
	assert.Equal(t, arts.Packages, []db.Package{{Name: "abc", Version: "2"}})
	checkSymlinks(t, install, arts.Exes)

	t.Log("Build using HTTP source")

	srv := httptest.NewServer(http.FileServer(http.Dir(root)))
	defer srv.Close()

	install = t.TempDir()

	arts, err = Build(apt.BuildBase(t), t.TempDir(), install, "some-wrapper", srv.URL, []db.Package{
		{Name: "r-lib"},
		{Name: "abc", Version: "1"},
	})
	assert.NoError(t, err)

	assert.Equal(t, arts.Exes, []string{"R", "Rscript", "abc"})
	assert.Equal(t, arts.Packages, []db.Package{
		{Name: "r-lib", Version: "1.1"},
		{Name: "abc", Version: "1"},
		{Name: "r", Version: "4.4.0", Interpreter: true},
	})
	checkSymlinks(t, install, arts.Exes)

	install = t.TempDir()

	t.Log("Build using S3 source")

	s := apt.MockS3ServerWithRoot(t, root)
	defer s.Close()

	arts, err = Build(apt.BuildBase(t), t.TempDir(), install, "some-wrapper", "s3://apt", []db.Package{
		{Name: "py-xyz"},
	})
	assert.NoError(t, err)

	assert.Equal(t, arts.Exes, []string{"python", "python3.13"}) //nolint:goconst
	assert.Equal(t, arts.Packages, []db.Package{
		{Name: "py-xyz", Version: "2.1"},
		{Name: "python", Version: "3.13", Interpreter: true},
	})
	checkSymlinks(t, install, arts.Exes)
}

func checkSymlinks(t *testing.T, install string, exes []string) {
	t.Helper()

	for _, exe := range exes {
		link, err := os.Readlink(filepath.Join(install, exe))
		assert.NoError(t, err)
		assert.Equal(t, link, "some-wrapper")
	}
}

func addAptCudf(t *testing.T, root string) {
	debPath := filepath.Join("pool", "main", "binary-"+runtime.GOARCH, "aptcudf.deb")
	assert.NoError(t, os.WriteFile(filepath.Join(root, debPath), aptcudf, 0600))

	m := md5.New()   //nolint:gosec
	s1 := sha1.New() //nolint:gosec
	s2 := sha256.New()
	s5 := sha512.New()

	io.MultiWriter(m, s1, s2, s5).Write(aptcudf)

	f, err := os.OpenFile(filepath.Join(root, "dists", "resolute", "main", "binary-"+runtime.GOARCH, "Packages"), os.O_WRONLY|os.O_APPEND, 0600)
	assert.NoError(t, err)

	fmt.Fprintf(
		f,
		"Package: apt-cudf\nArchitecture: all\nVersion: 1,9,9\nFilename: %s\nSize: %d\nMD5sum: %x\nSHA1: %x\nSHA256: %x\nSHA512: %x\n\n",
		debPath,
		len(aptcudf),
		m.Sum(nil),
		s1.Sum(nil),
		s2.Sum(nil),
		s5.Sum(nil),
	)

	assert.NoError(t, f.Close())
}
