package webhook_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	. "github.com/pivotal/projects-operator/pkg/webhook"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("NamespaceFetcher", func() {
	var (
		fetcher    NamespaceFetcher
		fakeClient client.Client
	)

	BeforeEach(func() {
		namespaceA := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "namespace-a",
			},
		}

		namespaceB := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "namespace-b",
			},
		}

		fakeClient = fake.NewFakeClient(namespaceA, namespaceB)
		fetcher = NewNamespaceFetcher(fakeClient)
	})

	Describe("GetNamespaces", func() {
		It("returns a list of namespaces", func() {
			namespaces, err := fetcher.GetNamespaces()
			Expect(err).NotTo(HaveOccurred())
			Expect(namespaces).To(HaveLen(2))
			Expect([]string{namespaces[0].ObjectMeta.Name, namespaces[1].ObjectMeta.Name}).To(ConsistOf("namespace-a", "namespace-b"))
		})
	})
})
