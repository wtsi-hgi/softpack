package backend

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetGroups(t *testing.T) {
	s := newTestServer(t)

	code, resp := getResponse(t, s, "/groups", "mercury")
	assert.Equal(t, http.StatusOK, code)
	assert.Contains(t, resp, "hgi")
}
