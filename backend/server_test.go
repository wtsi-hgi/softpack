package backend

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wtsi-hgi/softpack/db"
)

func TestServer(t *testing.T) {}

const testPackages = `
Package: pkg1
Version: 1
XB-Softpack: true
Description: desc1

Package: pkg1
Version: 2
XB-Softpack: true
Description: desc1

Package: pkg2
Version: 2
XB-Softpack: true
Description: desc2

Package: pkg2
Version: 5
XB-Softpack: true
Description: desc2

Package: pkg2
Version: 8
XB-Softpack: true
Description: desc2

Package: pkg3
Version: 3
XB-Softpack: true
Description: desc3

Package: pkg4
Version: 4
XB-Softpack: true
`

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	ps := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, testPackages)
	}))

	t.Cleanup(ps.Close)

	backend := New(ps.URL)
	s := httptest.NewServer(backend.Serve())

	t.Cleanup(s.Close)

	return s
}

func getResponse(t *testing.T, s *httptest.Server, endpoint string, body ...any) (int, string) {
	t.Helper()

	var reader io.Reader
	var method string

	if len(body) == 0 {
		method = http.MethodGet
	} else {
		method = http.MethodPost

		switch v := body[0].(type) {
		case io.Reader:
			reader = v
		default:
			jsonBody, err := json.Marshal(body[0])
			assert.NoError(t, err)
			reader = bytes.NewReader(jsonBody)
		}
	}

	r, err := http.NewRequest(method, s.URL+endpoint, reader)
	if err != nil {
		t.Fatalf("getResponse error: %s", err)
	}

	resp, err := s.Client().Do(r)
	if err != nil {
		t.Fatalf("getResponse error: %s", err)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("getResponse error: %s", err)
	}

	return resp.StatusCode, string(respBody)
}

// TODO: it would probably be nice to use these helpers in the db pkg but without duplication,
// is it too far to abstract these and definitions into their own errors pkg?
func assertEmptyResp(t *testing.T, code int, resp string) {
	assert.Equal(t, http.StatusNoContent, code)
	assert.Empty(t, resp)
}

func assertBadRequest(t *testing.T, code int, resp string, err error) {
	assert.Equal(t, http.StatusBadRequest, code)
	assert.Contains(t, resp, err.Error())
}

func checkAllEqual[T db.Environment | db.RecipeRequest](t *testing.T, s *httptest.Server, expected []T) {
	t.Helper()

	var endpoint string
	var v T

	switch any(v).(type) {
	case db.Environment:
		endpoint = "/get-environments"
	case db.RecipeRequest:
		endpoint = "/requested-recipes"
	}

	code, resp := getResponse(t, s, endpoint)
	assert.Equal(t, http.StatusOK, code)

	var actual []T
	err := json.NewDecoder(strings.NewReader(resp)).Decode(&actual)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}
