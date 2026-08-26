package backend

import (
	"net/http"
	"os/user"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetGroups(t *testing.T) {
	s := newTestServer(t)
	u, err := user.Current()
	assert.NoError(t, err)

	code, resp := getResponse(t, s, "/groups", u.Username)
	grp, err := user.LookupGroupId(u.Gid)
	assert.NoError(t, err)

	assert.Equal(t, http.StatusOK, code)
	assert.Contains(t, resp, grp.Name)
}
