package webhook_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal/projects-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	. "github.com/pivotal/projects-operator/pkg/webhook"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("ProjectFetcher", func() {
	var (
		fetcher    ProjectFetcher
		fakeClient client.Client
	)

	BeforeEach(func() {
		projectA := &v1alpha1.Project{
			ObjectMeta: metav1.ObjectMeta{
				Name: "project-a",
			},
		}

		projectB := &v1alpha1.Project{
			ObjectMeta: metav1.ObjectMeta{
				Name: "project-b",
			},
		}

		scheme, err := v1alpha1.SchemeBuilder.Build()
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
