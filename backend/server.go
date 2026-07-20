package backend

import (
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

func New() *Server {
	database, _ := db.Connect("sqlite3", ":memory:")

	return &Server{
		db: database,
	}
}
