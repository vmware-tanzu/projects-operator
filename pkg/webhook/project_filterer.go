package webhook

import (
	"fmt"

	"github.com/pivotal/projects-operator/api/v1alpha1"
	v1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
)

//go:generate counterfeiter . ProjectFilterer

type ProjectFilterer interface {
	FilterProjects([]v1alpha1.Project, v1.UserInfo) []string
}

type projectFilterer struct{}

func NewProjectFilterer() projectFilterer {
	return projectFilterer{}
}

//TODO: return []v1alpha1.Project instead of []string ?
func (f projectFilterer) FilterProjects(projects []v1alpha1.Project, user v1.UserInfo) []string {
	groupProjectMap := make(map[string][]string)
	usernameProjectMap := make(map[string][]string)
	serviceAccountProjectMap := make(map[string][]string)

	namespace := corev1.NamespaceDefault

	for _, project := range projects {
		for _, access := range project.Spec.Access {
			switch kind := access.Kind; kind {
			case "Group":
				groupProjectMap[access.Name] = append(groupProjectMap[access.Name], project.Name)
			case "User":
				usernameProjectMap[access.Name] = append(usernameProjectMap[access.Name], project.Name)
			case "ServiceAccount":
				if access.Namespace != "" {
					namespace = access.Namespace
				}
				fullServiceAccountName := fmt.Sprintf("system:serviceaccount:%s:%s", namespace, access.Name)
				serviceAccountProjectMap[fullServiceAccountName] = append(serviceAccountProjectMap[fullServiceAccountName], project.Name)
			}
		}
	}

	var filteredProjects []string
	for _, group := range user.Groups {
		filteredProjects = append(filteredProjects, groupProjectMap[group]...)
	}
	filteredProjects = append(filteredProjects, usernameProjectMap[user.Username]...)
	filteredProjects = append(filteredProjects, serviceAccountProjectMap[user.Username]...)

	return deduplicateStringSlice(filteredProjects)
}

func deduplicateStringSlice(strs []string) []string {
	set := make(map[string]struct{}, len(strs))
	for _, str := range strs {
		set[str] = struct{}{}
	}
	var dedupedStrs []string
	for key := range set {
		dedupedStrs = append(dedupedStrs, key)
	}

	return dedupedStrs
}
