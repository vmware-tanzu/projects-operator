package webhook

import (
	"net/http"
)

func NewHandler() http.Handler {
	mux := http.NewServeMux()

	projectsHandler := NewProjectsHandler()

	mux.HandleFunc("/projects", projectsHandler.HandleProjects)

	return mux
}
