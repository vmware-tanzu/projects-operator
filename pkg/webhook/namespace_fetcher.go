// Copyright 2019-2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package webhook

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//go:generate counterfeiter . NamespaceFetcher

type NamespaceFetcher interface {
	GetNamespaces() ([]corev1.Namespace, error)
}

type namespaceFetcher struct {
	client client.Client
}

func NewNamespaceFetcher(client client.Client) *namespaceFetcher {
	return &namespaceFetcher{
		client: client,
	}
}

func (f *namespaceFetcher) GetNamespaces() ([]corev1.Namespace, error) {
	namespaceList := &corev1.NamespaceList{}
	err := f.client.List(context.TODO(), namespaceList)

	return namespaceList.Items, err
}
