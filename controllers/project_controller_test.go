package controllers_test

import (
	"context"

	marketplacev1 "github.com/pivotal-cf/marketplace-project/api/v1"
	"github.com/pivotal-cf/marketplace-project/controllers"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ProjectController", func() {

	Describe("Reconcile", func() {
		var (
			reconciler controllers.ProjectReconciler
			fakeClient client.Client
			project    *marketplacev1.Project
			scheme     *runtime.Scheme
		)

		BeforeEach(func() {
			scheme = runtime.NewScheme()
			marketplacev1.AddToScheme(scheme)
			corev1.AddToScheme(scheme)

			project = Project("my-project")
		})

		JustBeforeEach(func() {
			fakeClient = fake.NewFakeClientWithScheme(scheme, project)

			reconciler = controllers.ProjectReconciler{
				Log:    ctrl.Log.WithName("controllers").WithName("Project"),
				Client: fakeClient,
				Scheme: scheme,
			}
		})

		When("there is a new project resource", func() {
			It("creates a namespace with given project name", func() {
				_, err := reconciler.Reconcile(Request(project.Namespace, project.Name))
				Expect(err).NotTo(HaveOccurred())

				namespace := &corev1.Namespace{}
				err = fakeClient.Get(context.TODO(), client.ObjectKey{
					Name: project.Name,
				}, namespace)
				Expect(err).NotTo(HaveOccurred())

				Expect(namespace.Name).To(Equal("my-project"))
			})

			It("sets the owner reference to the Project resource", func() {
				_, err := reconciler.Reconcile(Request(project.Namespace, project.Name))
				Expect(err).NotTo(HaveOccurred())

				namespace := &corev1.Namespace{}
				err = fakeClient.Get(context.TODO(), client.ObjectKey{
					Name: project.Name,
				}, namespace)
				Expect(err).NotTo(HaveOccurred())

				Expect(namespace.ObjectMeta.OwnerReferences).To(HaveLen(1))
				ownerReference := namespace.ObjectMeta.OwnerReferences[0]
				Expect(ownerReference.Name).To(Equal(project.Name))
				Expect(ownerReference.Kind).To(Equal("Project"))
			})
		})

		When("project resource already exists", func() {
			It("updates the project", func() {
				_, err := reconciler.Reconcile(Request(project.Namespace, project.Name))
				Expect(err).NotTo(HaveOccurred())

				_, err = reconciler.Reconcile(Request(project.Namespace, project.Name))
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
})

func Request(namespace, name string) ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name: name,
		},
	}
}

func Project(name string) *marketplacev1.Project {
	return &marketplacev1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}
