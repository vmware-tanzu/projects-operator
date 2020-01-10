package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/pivotal/projects-operator/pkg/webhook"
)

const port = 8080

func main() {
	handler := webhook.NewHandler()
	keyPath := os.Getenv("TLS_KEY_FILEPATH")
	crtPath := os.Getenv("TLS_CERT_FILEPATH")

	_, err := tls.LoadX509KeyPair(crtPath, keyPath)
	if err != nil {
		log.Fatalf("Failed to load key pair: %+v", err)
	}

	log.Fatal(http.ListenAndServeTLS(
		fmt.Sprintf(":%d", port),
		crtPath,
		keyPath,
		handler,
	))
}
