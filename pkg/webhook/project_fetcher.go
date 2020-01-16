package webhook

import (
	"context"

	"github.com/pivotal/projects-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//go:generate counterfeiter . ProjectFetcher

type ProjectFetcher interface {
	GetProjects() ([]v1alpha1.Project, error)
}

type projectFetcher struct {
	client client.Client
}

func NewProjectFetcher(client client.Client) *projectFetcher {
	return &projectFetcher{
		client: client,
	}
}

func (f *projectFetcher) GetProjects() ([]v1alpha1.Project, error) {
	projectList := &v1alpha1.ProjectList{}
	err := f.client.List(context.TODO(), projectList)

	return projectList.Items, err
}
