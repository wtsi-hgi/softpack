package build

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/johannesboyne/gofakes3"
	"github.com/johannesboyne/gofakes3/backend/s3afero"
	"github.com/stretchr/testify/assert"
	"github.com/wtsi-hgi/softpack/internal/apt"
)

func TestBuild(t *testing.T) {
	apt.SkipIfBadEnvironment(t)

	root := apt.CreateTestAptRepo(t, apt.ExamplePackages())

	install := t.TempDir()

	t.Log("Build using FS source")

	arts, err := Build(apt.BuildBase, t.TempDir(), install, "some-wrapper", root, []Package{
		{Name: "abc"}, //nolint:goconst
	})
	assert.NoError(t, err)

	assert.Equal(t, arts.Exes, []string{"abc", "def"})                     //nolint:goconst
	assert.Equal(t, arts.Packages, []Package{{Name: "abc", Version: "2"}}) //nolint:goconst
	checkSymlinks(t, install, arts.Exes)

	t.Log("Build using HTTP source")

	srv := httptest.NewServer(http.FileServer(http.Dir(root)))
	defer srv.Close()

	install = t.TempDir()

	arts, err = Build(apt.BuildBase, t.TempDir(), install, "some-wrapper", srv.URL, []Package{
		{Name: "r-lib"}, //nolint:goconst
		{Name: "abc", Version: "1"},
	})
	assert.NoError(t, err)

	assert.Equal(t, arts.Exes, []string{"R", "Rscript", "abc"})
	assert.Equal(t, arts.Packages, []Package{
		{Name: "r-lib", Version: "1.1"},
		{Name: "abc", Version: "1"},
		{Name: "r", Version: "4.4.0", Interpreter: true},
	})
	checkSymlinks(t, install, arts.Exes)

	install = t.TempDir()

	t.Log("Build using S3 source")

	files, err := s3afero.FsPath(root, 0)
	assert.NoError(t, err)

	bucket, err := s3afero.SingleBucket("apt", files, nil)
	assert.NoError(t, err)

	srv = httptest.NewServer(gofakes3.New(bucket).Server())
	defer srv.Close()

	t.Setenv("AWS_ENDPOINT_URL", srv.URL)
	t.Setenv("AWS_DEFAULT_REGION", "eu-west-1")
	t.Setenv("AWS_REGION", "eu-west-1")
	t.Setenv("AWS_DEFAULT_OUTPUT", "json")
	t.Setenv("AWS_ACCESS_KEY_ID", "ACCESS_KEY")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "SECRET")

	arts, err = Build(apt.BuildBase, t.TempDir(), install, "some-wrapper", "s3://apt", []Package{
		{Name: "py-xyz"}, //nolint:goconst
	})
	assert.NoError(t, err)

	assert.Equal(t, arts.Exes, []string{"python", "python3.13"}) //nolint:goconst
	assert.Equal(t, arts.Packages, []Package{
		{Name: "py-xyz", Version: "2.1"},
		{Name: "python", Version: "3.13", Interpreter: true}, //nolint:goconst
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