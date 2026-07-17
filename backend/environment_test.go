package backend

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wtsi-hgi/softpack/db"
)

// TODO: Sort helper func for http stuff
func TestEnvironment(t *testing.T) {
	s := New()

	w := httptest.NewRecorder()
	r := httptest.NewRequest(
		http.MethodGet,
		"/get-environment",
		nil,
	)

	s.GetEnvironment(w, r)

	assert.Equal(t, 200, w.Code)

	var envs []db.Environment

	err := json.NewDecoder(w.Body).Decode(&envs)
	assert.NoError(t, err)

	assert.Equal(t, envs, []db.Environment{})

	//

	environment := db.Environment{
		Name:    "test",
		Path:    "path/to/test",
		Version: 1,
	}
	jsonBody, _ := json.Marshal(environment)

	w = httptest.NewRecorder()
	r = httptest.NewRequest(
		http.MethodPost,
		"/create-environment",
		strings.NewReader(string(jsonBody)),
	)

	s.CreateEnvironment(w, r)
	assert.Equal(t, 200, w.Code)

	//

	w = httptest.NewRecorder()
	r = httptest.NewRequest(
		http.MethodGet,
		"/get-environment",
		nil,
	)

	s.GetEnvironment(w, r)

	assert.Equal(t, 200, w.Code)

	err = json.NewDecoder(w.Body).Decode(&envs)
	assert.NoError(t, err)
	assert.Equal(t, envs, []db.Environment{environment})
}
