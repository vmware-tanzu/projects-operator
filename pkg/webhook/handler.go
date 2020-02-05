package webhook

import (
	"net/http"
)

func NewHandler(namespaceFetcher NamespaceFetcher, projectFetcher ProjectFetcher, projectFilterer ProjectFilterer) http.Handler {
	mux := http.NewServeMux()

	projectHandler := NewProjectHandler(namespaceFetcher)
	projectAccessHandler := NewProjectAccessHandler(projectFetcher, projectFilterer)

	mux.HandleFunc("/project", projectHandler.HandleProject)
	mux.HandleFunc("/projectaccess", projectAccessHandler.HandleProjectAccess)

	return mux
}
