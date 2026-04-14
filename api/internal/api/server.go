package api

import (
	"github.com/locksmithhq/telm/internal/storage/postgres"
)

type Server struct {
	store     *postgres.Client
	jwtSecret []byte
}

func NewServer(store *postgres.Client, jwtSecret []byte) *Server {
	return &Server{store: store, jwtSecret: jwtSecret}
}
