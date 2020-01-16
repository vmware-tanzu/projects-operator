package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/pivotal/projects-operator/api/v1alpha1"
	"github.com/pivotal/projects-operator/pkg/webhook"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const port = 8080

var (
	scheme = runtime.NewScheme()
)

func init() {
	_ = v1alpha1.AddToScheme(scheme)
}

func main() {
	kubeClient, err := client.New(ctrl.GetConfigOrDie(), client.Options{Scheme: scheme})
	if err != nil {
		log.Printf("Failed to build a Kubernetes client: %s", err.Error())
		os.Exit(1)
	}

	projectFetcher := webhook.NewProjectFetcher(kubeClient)
	projectFilterer := webhook.NewProjectFilterer()

	handler := webhook.NewHandler(projectFetcher, projectFilterer)

	keyPath := os.Getenv("TLS_KEY_FILEPATH")
	crtPath := os.Getenv("TLS_CERT_FILEPATH")

	_, err = tls.LoadX509KeyPair(crtPath, keyPath)
	if err != nil {
		log.Fatalf("Failed to load key pair: %+v", err)
	}

	log.Println("starting webhook server")
	log.Fatal(http.ListenAndServeTLS(
		fmt.Sprintf(":%d", port),
		crtPath,
		keyPath,
		handler,
	))
}
