package backend

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServer(t *testing.T) {}

// var routes map[string]http.HandlerFunc = {
// 		"/create-environment": b.CreateEnvironment,
// 		"/get-environment":    b.GetEnvironment,
// 		"/delete-environment": b.DeleteEnvironment,
// 	}

// type HandlerFunction struct {
// 	fn http.HandlerFunc
// 	endpoint, method string
// }

func getResponse(t *testing.T, fn http.HandlerFunc, endpoint string, body ...any) (int, string) {
	t.Helper()

	var reader io.Reader
	var method string

	w := httptest.NewRecorder()

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

	r := httptest.NewRequest(method, endpoint, reader)
	fn(w, r)

	return w.Code, w.Body.String()
}

func assertEmptyResp(t *testing.T, code int, resp string) {
	assert.Equal(t, 200, code)
	assert.Empty(t, resp)
}
