package webhook

import (
	"fmt"
	"net/http"
)

type ProjectsHandler struct{}

func NewProjectsHandler() *ProjectsHandler {
	return &ProjectsHandler{}
}

func (h *ProjectsHandler) HandleProjects(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "not implemented")
}
