package build

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wtsi-hgi/softpack/db"
	"github.com/wtsi-hgi/softpack/internal/apt"
)

func TestBuild(t *testing.T) {
	apt.SkipIfBadEnvironment(t)

	root := apt.CreateTestAptRepo(t, apt.ExamplePackages())

	install := t.TempDir()

	t.Log("Build using FS source")

	arts, err := Build(apt.BuildBase, t.TempDir(), install, "some-wrapper", root, []db.Package{
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

	arts, err = Build(apt.BuildBase, t.TempDir(), install, "some-wrapper", srv.URL, []db.Package{
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

	s := apt.MockS3Server(t, apt.ExamplePackages())
	defer s.Close()

	arts, err = Build(apt.BuildBase, t.TempDir(), install, "some-wrapper", "s3://apt", []db.Package{
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
