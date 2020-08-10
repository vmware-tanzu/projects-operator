// Copyright 2019-2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"

	projects "github.com/pivotal/projects-operator/api/v1alpha1"
	"github.com/pivotal/projects-operator/pkg/webhook"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/klog/v2/klogr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const port = 8080

var (
	scheme        = runtime.NewScheme()
	webhookLogger = ctrl.Log.WithName("webhook")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
	_ = projects.AddToScheme(scheme)
}

func main() {
	ctrl.SetLogger(klogr.New())

	kubeClient, err := client.New(ctrl.GetConfigOrDie(), client.Options{Scheme: scheme})
	if err != nil {
		webhookLogger.Error(err, "Failed to build a Kubernetes client")
		os.Exit(1)
	}

	projectFetcher := webhook.NewProjectFetcher(kubeClient)
	namespaceFetcher := webhook.NewNamespaceFetcher(kubeClient)
	projectFilterer := webhook.NewProjectFilterer()

	handler := webhook.NewHandler(webhookLogger.WithName("handler"), namespaceFetcher, projectFetcher, projectFilterer)

	keyPath := os.Getenv("TLS_KEY_FILEPATH")
	crtPath := os.Getenv("TLS_CERT_FILEPATH")

	_, err = tls.LoadX509KeyPair(crtPath, keyPath)
	if err != nil {
		webhookLogger.Error(err, "Failed to load key pair")
		os.Exit(1)
	}

	webhookLogger.Info("starting webhook server")
	if err = http.ListenAndServeTLS(
		fmt.Sprintf(":%d", port),
		crtPath,
		keyPath,
		handler,
	); err != nil {
		webhookLogger.Error(err, "ListenAndServe terminated")
	}
}
