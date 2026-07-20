package backend

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wtsi-hgi/softpack/db"
)

// TODO: Sort helper func for http stuff
func TestEnvironment(t *testing.T) {
	s := New()
	var envs []db.Environment

	code, resp := getResponse(t, s.GetEnvironment, "/get-environment")
	assert.Equal(t, 200, code)
	err := json.NewDecoder(strings.NewReader(resp)).Decode(&envs)
	assert.NoError(t, err)
	assert.Equal(t, []db.Environment{}, envs)

	//

	environment := db.Environment{
		Name:    "test",
		Path:    "path/to/test",
		Version: 1,
	}
	code, resp = getResponse(t, s.CreateEnvironment, "/create-environment", environment)
	assertEmptyResp(t, code, resp)

	//

	code, resp = getResponse(t, s.GetEnvironment, "/get-environment")
	assert.Equal(t, 200, code)
	err = json.NewDecoder(strings.NewReader(resp)).Decode(&envs)
	assert.NoError(t, err)
	assert.Equal(t, []db.Environment{environment}, envs)

	//

	code, resp = getResponse(t, s.DeleteEnvironment, "/delete-environment", environment)
	assertEmptyResp(t, code, resp)

	//

	code, resp = getResponse(t, s.GetEnvironment, "/get-environment")
	assert.Equal(t, 200, code)
	err = json.NewDecoder(strings.NewReader(resp)).Decode(&envs)
	assert.NoError(t, err)
	assert.Equal(t, []db.Environment{}, envs)

	//

	code, resp = getResponse(t, s.CreateEnvironment, "/create-environment")

}
