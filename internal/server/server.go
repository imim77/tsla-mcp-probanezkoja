package server

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Server struct {
	router chi.Router
}

func NewServer() *Server {
	srv := &Server{
		router: chi.NewRouter(),
	}

	srv.Routes()
	return srv
}

func (s *Server) Routes() {
	s.router.Get("/", s.handleHome)
}

func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	_, err := w.Write([]byte("Hello, world!"))
	if err != nil {
		fmt.Println(err)
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}
