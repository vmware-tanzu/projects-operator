package controllers_test

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-cf/marketplace-project/api/v1"
	marketplacev1 "github.com/pivotal-cf/marketplace-project/api/v1"
	"github.com/pivotal-cf/marketplace-project/controllers"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func Request(namespace, name string) ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func Project(namespace, name string) *v1.Project {
	return &v1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1.ProjectSpec{
			Name: name,
		},
	}
}

var _ = Describe("ProjectController", func() {

	var (
		reconciler controllers.ProjectReconciler
		fakeClient client.Client
		project    *v1.Project
	)

	BeforeEach(func() {
		scheme := runtime.NewScheme()
		marketplacev1.AddToScheme(scheme)
		corev1.AddToScheme(scheme)

		project = Project("default", "my-project")

		fakeClient = fake.NewFakeClientWithScheme(scheme, project)

		reconciler = controllers.ProjectReconciler{
			Log:    ctrl.Log.WithName("controllers").WithName("Project"),
			Client: fakeClient,
		}
	})

	Describe("Reconcile", func() {
		It("creates a namespace under required namespace", func() {
			_, err := reconciler.Reconcile(Request(project.Namespace, project.Name))
			Expect(err).NotTo(HaveOccurred())

			namespace := &corev1.Namespace{}
			err = fakeClient.Get(context.TODO(), client.ObjectKey{
				Name: "my-project",
			}, namespace)
			Expect(err).NotTo(HaveOccurred())

			Expect(namespace.Name).To(Equal("my-project"))
			Expect(namespace.Namespace).To(Equal("default"))
		})

		It("updates an existing namespace under required namespace", func() {
			_, err := reconciler.Reconcile(Request(project.Namespace, project.Name))
			Expect(err).NotTo(HaveOccurred())

			_, err = reconciler.Reconcile(Request(project.Namespace, project.Name))
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
