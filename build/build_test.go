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
	root := initRepo(t)

	install := t.TempDir()

	t.Log("Build using FS source")

	arts, err := Build(apt.BuildBase, t.TempDir(), install, "some-wrapper", root, []Package{
		{Name: "abc"},
	})
	assert.ErrorIs(t, err, nil)

	assert.Equal(t, arts.Exes, []string{"abc", "def"})
	assert.Equal(t, arts.Packages, []Package{{Name: "abc", Version: "2"}})
	checkSymlinks(t, install, arts.Exes)

	t.Log("Build using HTTP source")

	srv := httptest.NewServer(http.FileServer(http.Dir(root)))
	defer srv.Close()

	install = t.TempDir()

	arts, err = Build(apt.BuildBase, t.TempDir(), install, "some-wrapper", srv.URL, []Package{
		{Name: "r-lib"},
		{Name: "abc", Version: "1"},
	})
	assert.ErrorIs(t, err, nil)

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
	assert.ErrorIs(t, err, nil)

	bucket, err := s3afero.SingleBucket("apt", files, nil)
	assert.ErrorIs(t, err, nil)

	srv = httptest.NewServer(gofakes3.New(bucket).Server())
	defer srv.Close()

	t.Setenv("AWS_ENDPOINT_URL", srv.URL)
	t.Setenv("AWS_DEFAULT_REGION", "eu-west-1")
	t.Setenv("AWS_REGION", "eu-west-1")
	t.Setenv("AWS_DEFAULT_OUTPUT", "json")
	t.Setenv("AWS_ACCESS_KEY_ID", "ACCESS_KEY")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "SECRET")

	t.Cleanup(func() { clear(s3options) })

	arts, err = Build(apt.BuildBase, t.TempDir(), install, "some-wrapper", "s3://apt", []Package{
		{Name: "py-xyz"},
	})
	assert.ErrorIs(t, err, nil)

	assert.Equal(t, arts.Exes, []string{"python", "python3.13"})
	assert.Equal(t, arts.Packages, []Package{
		{Name: "py-xyz", Version: "2.1"},
		{Name: "python", Version: "3.13", Interpreter: true},
	})
	checkSymlinks(t, install, arts.Exes)
}

func checkSymlinks(t *testing.T, install string, exes []string) {
	t.Helper()

	for _, exe := range exes {
		link, err := os.Readlink(filepath.Join(install, exe))
		assert.ErrorIs(t, err, nil)
		assert.Equal(t, link, "some-wrapper")
	}
}

func initRepo(t *testing.T) string {
	t.Helper()
	apt.SkipIfBadEnvironment(t)

	return apt.CreateTestAptRepo(t, []apt.Deb{
		{
			Name:    "abc",
			Version: "1",
			Metadata: map[string]string{
				"XB-Executables": "abc",
			},
		},
		{
			Name:    "abc",
			Version: "2",
			Metadata: map[string]string{
				"XB-Executables": "abc, def",
			},
		},
		{
			Name:    "python",
			Version: "3.13",
			Metadata: map[string]string{
				"XB-Executables": "python, python3.13",
			},
		},
		{
			Name:    "py-xyz",
			Version: "2.1",
			Metadata: map[string]string{
				"Depends": "python",
			},
		},
		{
			Name:    "r",
			Version: "4.4.0",
			Metadata: map[string]string{
				"XB-Executables": "R, Rscript",
			},
		},
		{
			Name:    "r-lib",
			Version: "1.1",
			Metadata: map[string]string{
				"Depends": "r",
			},
		},
	})
}
