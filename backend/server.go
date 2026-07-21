package backend

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/wtsi-hgi/softpack/db"
)

type Server struct {
	envMu sync.RWMutex

	db *db.DB
}

type Error struct {
	err  error
	code int
}

func HttpError(w http.ResponseWriter, err Error) {
	http.Error(w, err.err.Error(), err.code)
}

func GetItemFromRequest[T any](r *http.Request) (*T, error) {
	var item T

	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		return nil, err
	}

	return &item, nil
}

func New() *Server {
	database, _ := db.Connect("sqlite3", ":memory:")

	return &Server{
		db: database,
	}
}
