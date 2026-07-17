package backend

import (
	"sync"

	"github.com/wtsi-hgi/softpack/db"
)

type Server struct {
	envMu sync.RWMutex

	db *db.DB
}

func New() *Server {
	database, _ := db.Connect("sqlite3", ":memory:")

	return &Server{
		db: database,
	}
}
