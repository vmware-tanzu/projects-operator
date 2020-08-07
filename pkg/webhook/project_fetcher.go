// Copyright 2020-Present VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package webhook

import (
	"context"

	projects "github.com/pivotal/projects-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//go:generate counterfeiter . ProjectFetcher

type ProjectFetcher interface {
	GetProjects() ([]projects.Project, error)
}

type projectFetcher struct {
	client client.Client
}

func NewProjectFetcher(client client.Client) *projectFetcher {
	return &projectFetcher{
		client: client,
	}
}

func (f *projectFetcher) GetProjects() ([]projects.Project, error) {
	projectList := &projects.ProjectList{}
	err := f.client.List(context.TODO(), projectList)

	return projectList.Items, err
}
