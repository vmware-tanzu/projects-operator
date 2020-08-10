// Copyright 2019-2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package webhook_test

import (
	projects "github.com/pivotal/projects-operator/api/v1alpha1"
	authenticationv1 "k8s.io/api/authentication/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/pivotal/projects-operator/pkg/webhook"
)

var _ = Describe("ProjectFilterer", func() {
	var (
		filterer         ProjectFilterer
		projectsToFilter []projects.Project
		user             authenticationv1.UserInfo

		filteredProjects []string
	)

	BeforeEach(func() {
		project1 := projects.Project{
			ObjectMeta: metav1.ObjectMeta{
				Name: "project-1",
			},
			Spec: projects.ProjectSpec{
				Access: []projects.SubjectRef{
					{
						Kind: "User",
						Name: "developer-1",
					},
					{
						Kind: "Group",
						Name: "group-1",
					},
					{
						Kind:      "ServiceAccount",
						Name:      "service-account-1",
						Namespace: "namespace-1",
					},
				},
			},
		}

		project2 := projects.Project{
			ObjectMeta: metav1.ObjectMeta{
				Name: "project-2",
			},
			Spec: projects.ProjectSpec{
				Access: []projects.SubjectRef{
					{
						Kind: "User",
						Name: "developer-2",
					},
					{
						Kind: "Group",
						Name: "group-2",
					},
					{
						Kind:      "ServiceAccount",
						Name:      "service-account-2",
						Namespace: "namespace-2",
					},
				},
			},
		}

		projectsToFilter = []projects.Project{project1, project2}
		filterer = NewProjectFilterer()
	})

	JustBeforeEach(func() {
		filteredProjects = filterer.FilterProjects(projectsToFilter, user)
	})

	When("the user matches no projects", func() {
		BeforeEach(func() {
			user = authenticationv1.UserInfo{Username: "other-developer", Groups: []string{"other-group"}}
		})

		It("returns no projects", func() {
			Expect(filteredProjects).To(BeEmpty())
		})
	})

	When("the user matches a project by username matching user access", func() {
		BeforeEach(func() {
			user = authenticationv1.UserInfo{Username: "developer-1", Groups: []string{"other-group"}}
		})

		It("returns the project that grants access to the user", func() {
			Expect(filteredProjects).To(ConsistOf("project-1"))
		})
	})

	When("the user matches a project by group", func() {
		BeforeEach(func() {
			user = authenticationv1.UserInfo{Username: "other-developer", Groups: []string{"group-2"}}
		})

		It("returns the project that grants access to the group", func() {
			Expect(filteredProjects).To(ConsistOf("project-2"))
		})
	})

	When("the user matches a project by username matching service-account access", func() {
		BeforeEach(func() {
			user = authenticationv1.UserInfo{
				Username: "system:serviceaccount:namespace-1:service-account-1",
			}
		})

		It("returns the project that grants access to the service account", func() {
			Expect(filteredProjects).To(ConsistOf("project-1"))
		})
	})

	When("the user matches multiple projects", func() {
		BeforeEach(func() {
			user = authenticationv1.UserInfo{Username: "developer-1", Groups: []string{"group-2"}}
		})

		It("returns all the matched projects", func() {
			Expect(filteredProjects).To(ConsistOf("project-1", "project-2"))
		})
	})

	When("the user matches a project by both group and username", func() {
		BeforeEach(func() {
			user = authenticationv1.UserInfo{Username: "developer-2", Groups: []string{"group-2"}}
		})

		It("returns the project once only", func() {
			Expect(filteredProjects).To(ConsistOf("project-2"))
		})
	})

	When("the username matches a permission granted to a group", func() {
		BeforeEach(func() {
			user = authenticationv1.UserInfo{Username: "group-1"}
		})

		It("returns no projects", func() {
			Expect(filteredProjects).To(BeEmpty())
		})
	})

	When("the non-serviceaccount qualified username matches a permission granted to a service account", func() {
		BeforeEach(func() {
			user = authenticationv1.UserInfo{
				Username: "service-account-1",
			}
		})

		It("returns an empty list", func() {
			Expect(filteredProjects).To(BeEmpty())
		})
	})
})
