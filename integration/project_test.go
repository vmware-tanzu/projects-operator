package integration_test

import (
	"context"

	"github.com/pivotal-cf/marketplace-project/api/v1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Project", func() {

	It("creates a namespace with the project name", func() {

		project := &v1.Project{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-project",
				Namespace: "default",
			},
			Spec: v1.ProjectSpec{},
		}

		Expect(k8sClient.Create(context.TODO(), project)).To(Succeed())

		namespace := &corev1.Namespace{}
		getNamespace := func() error {
			err := k8sClient.Get(context.TODO(), client.ObjectKey{
				Name: "my-project",
			}, namespace)

			return err
		}

		Eventually(getNamespace, 10).Should(Succeed())
		Expect(namespace.ObjectMeta.Name).To(Equal("my-project"))
	})

	It("aa", func() {

		namespace := &corev1.Namespace{}
		getNamespace := func() error {
			err := k8sClient.Get(context.TODO(), client.ObjectKey{
				Name: "my-project",
			}, namespace)

			return err
		}

		Eventually(getNamespace, 10).Should(Succeed())
		Expect(namespace.ObjectMeta.Name).To(Equal("my-project"))
	})
})
