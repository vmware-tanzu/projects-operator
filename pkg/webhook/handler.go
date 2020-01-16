package webhook

import (
	"net/http"
)

func NewHandler(projectFetcher ProjectFetcher, projectFilterer ProjectFilterer) http.Handler {
	mux := http.NewServeMux()

	projectsHandler := NewProjectsHandler(projectFetcher, projectFilterer)

	mux.HandleFunc("/projects", projectsHandler.HandleProjects)

	return mux
}
