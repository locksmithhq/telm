package api

import (
	"github.com/locksmithhq/telm/internal/storage/postgres"
)

type Server struct {
	store *postgres.Client
}

func NewServer(store *postgres.Client) *Server {
	return &Server{store: store}
}
