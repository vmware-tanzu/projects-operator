// Copyright 2019-2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

package webhook_test

import (
	projects "github.com/pivotal/projects-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/pivotal/projects-operator/pkg/webhook"
)

var _ = Describe("ProjectFetcher", func() {
	var (
		fetcher    ProjectFetcher
		fakeClient client.Client
	)

	BeforeEach(func() {
		projectA := &projects.Project{
			ObjectMeta: metav1.ObjectMeta{
				Name: "project-a",
			},
		}

		projectB := &projects.Project{
			ObjectMeta: metav1.ObjectMeta{
				Name: "project-b",
			},
		}

		scheme, err := projects.SchemeBuilder.Build()
		Expect(err).NotTo(HaveOccurred())
		fakeClient = fake.NewFakeClientWithScheme(scheme, projectA, projectB)
		fetcher = NewProjectFetcher(fakeClient)
	})

	Describe("GetProjects", func() {
		It("returns a list of projects", func() {
			projects, err := fetcher.GetProjects()
			Expect(err).NotTo(HaveOccurred())
			Expect(projects).To(HaveLen(2))
			Expect([]string{projects[0].ObjectMeta.Name, projects[1].ObjectMeta.Name}).To(ConsistOf("project-a", "project-b"))
		})
	})
})
