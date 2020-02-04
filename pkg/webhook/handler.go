package webhook

import (
	"net/http"
)

func NewHandler(projectFetcher ProjectFetcher, projectFilterer ProjectFilterer) http.Handler {
	mux := http.NewServeMux()

	projectAccessHandler := NewProjectAccessHandler(projectFetcher, projectFilterer)

	mux.HandleFunc("/projectaccess", projectAccessHandler.HandleProjectAccess)

	return mux
}
