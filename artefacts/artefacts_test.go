package artefacts

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/johannesboyne/gofakes3"
	"github.com/johannesboyne/gofakes3/backend/s3afero"
	"github.com/stretchr/testify/assert"
)

func TestStoreFile(t *testing.T) {
	tmp := t.TempDir()

	testStore(t, tmp, tmp)
}

func testStore(t *testing.T, tmp, url string) {
	file := filepath.Join(t.TempDir(), "aFile.txt")

	assert.NoError(t, os.WriteFile(file, []byte("A file of data"), 0644))
	assert.NoError(t, Store(url, "myEnv", map[string]Opener{
		"myFile":         File(file),
		"logs/build.log": Data("Some Data"),
	}))

	data, err := os.ReadFile(filepath.Join(tmp, "myFile"))
	assert.NoError(t, err)
	assert.Equal(t, string(data), "A file of data")

	data, err = os.ReadFile(filepath.Join(tmp, "logs", "build.log"))
	assert.NoError(t, err)
	assert.Equal(t, string(data), "Some Data")
}

func TestStoreHTTP(t *testing.T) {
	tmp := t.TempDir()
	mux := http.NewServeMux()

	mux.Handle("PUT /files/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := filepath.Join(tmp, strings.TrimPrefix(r.URL.Path, "/files/"))

		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			w.WriteHeader(http.StatusInternalServerError)

			fmt.Fprintln(w, err)

			return
		}

		f, err := os.Create(p)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)

			fmt.Fprintln(w, err)

			return
		}

		if _, err := io.Copy(f, r.Body); err != nil {
			w.WriteHeader(http.StatusInternalServerError)

			fmt.Fprintln(w, err)

			return
		}
	}))

	srv := httptest.NewServer(mux)

	t.Cleanup(srv.Close)
	testStore(t, tmp, srv.URL+"/files/")
}

func TestStoreS3(t *testing.T) {
	tmp := t.TempDir()

	files, err := s3afero.FsPath(tmp, 0)
	assert.NoError(t, err)

	bucket, err := s3afero.SingleBucket("artefacts", files, nil)
	assert.NoError(t, err)

	srv := httptest.NewServer(gofakes3.New(bucket).Server())

	t.Cleanup(srv.Close)
	t.Setenv("AWS_ENDPOINT_URL", srv.URL)
	t.Setenv("AWS_DEFAULT_REGION", "eu-west-1")
	t.Setenv("AWS_REGION", "eu-west-1")
	t.Setenv("AWS_DEFAULT_OUTPUT", "json")
	t.Setenv("AWS_ACCESS_KEY_ID", "ACCESS_KEY")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "SECRET")

	testStore(t, tmp, "s3://artefacts/")
}
