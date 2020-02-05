package webhook

import (
	"net/http"

	"github.com/go-logr/logr"
)

func NewHandler(logger logr.Logger, namespaceFetcher NamespaceFetcher, projectFetcher ProjectFetcher, projectFilterer ProjectFilterer) http.Handler {
	mux := http.NewServeMux()

	projectHandler := NewProjectHandler(logger.WithName("project"), namespaceFetcher)
	projectAccessHandler := NewProjectAccessHandler(logger.WithName("projectaccess"), projectFetcher, projectFilterer)

	mux.HandleFunc("/project", projectHandler.HandleProject)
	mux.HandleFunc("/projectaccess", projectAccessHandler.HandleProjectAccess)

	return mux
}
