package controllers_test

import (
	"context"

	projectv1 "github.com/pivotal/projects-operator/api/v1"
	"github.com/pivotal/projects-operator/controllers"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
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
			project    *projectv1.Project
			user1      string
			user2      string
			scheme     *runtime.Scheme
			roleConfig controllers.RoleConfiguration
		)

		BeforeEach(func() {
			scheme = runtime.NewScheme()

			projectv1.AddToScheme(scheme)
			corev1.AddToScheme(scheme)
			rbacv1.AddToScheme(scheme)

			user1 = "some-user1"
			user2 = "some-user2"
			project = Project("my-project", user1, user2)

			fakeClient = fake.NewFakeClientWithScheme(scheme, project)

			roleConfig = controllers.RoleConfiguration{
				APIGroups: []string{"*"},
				Resources: []string{"configmap"},
				Verbs:     []string{"*"},
			}

			reconciler = controllers.ProjectReconciler{
				Log:        ctrl.Log.WithName("controllers").WithName("Project"),
				Client:     fakeClient,
				Scheme:     scheme,
				RoleConfig: roleConfig,
			}
		})

		Describe("deletion", func() {
			It("deletes a project without errors", func() {
				_, err := reconciler.Reconcile(Request(project.Namespace, project.Name))
				Expect(err).NotTo(HaveOccurred())

				err = fakeClient.Delete(context.TODO(), project)

				_, err = reconciler.Reconcile(Request(project.Namespace, project.Name))
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Describe("creation", func() {
			Describe("creates a namespace", func() {

				It("with given project name", func() {
					_, err := reconciler.Reconcile(Request(project.Namespace, project.Name))
					Expect(err).NotTo(HaveOccurred())

					namespace := &corev1.Namespace{}
					err = fakeClient.Get(context.TODO(), client.ObjectKey{
						Name: project.Name,
					}, namespace)
					Expect(err).NotTo(HaveOccurred())

					Expect(namespace.Name).To(Equal("my-project"))
				})

				It("owned by the project", func() {
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

			Describe("creates a role", func() {
				It("to access the project", func() {
					_, err := reconciler.Reconcile(Request(project.Namespace, project.Name))
					Expect(err).NotTo(HaveOccurred())

					role := &rbacv1.Role{}
					err = fakeClient.Get(context.TODO(), client.ObjectKey{
						Name:      project.Name + "-role",
						Namespace: project.Name,
					}, role)
					Expect(err).NotTo(HaveOccurred())

					Expect(role.Name).To(Equal("my-project-role"))
					Expect(role.ObjectMeta.Namespace).To(Equal("my-project"))
					rule := role.Rules[0]
					Expect(rule.APIGroups).To(ConsistOf(roleConfig.APIGroups))
					Expect(rule.Resources).To(ConsistOf(roleConfig.Resources))
					Expect(rule.Verbs).To(ConsistOf(roleConfig.Verbs))
				})

				It("owned by the project", func() {
					_, err := reconciler.Reconcile(Request(project.Namespace, project.Name))
					Expect(err).NotTo(HaveOccurred())

					role := &rbacv1.Role{}
					err = fakeClient.Get(context.TODO(), client.ObjectKey{
						Name:      project.Name + "-role",
						Namespace: project.Name,
					}, role)
					Expect(err).NotTo(HaveOccurred())

					Expect(role.ObjectMeta.OwnerReferences).To(HaveLen(1))
					ownerReference := role.ObjectMeta.OwnerReferences[0]
					Expect(ownerReference.Name).To(Equal(project.Name))
					Expect(ownerReference.Kind).To(Equal("Project"))
				})
			})

			Describe("creates a role binding", func() {
				When("the subject is a ServiceAccount", func() {
					var serviceAccountName = "service-account"

					BeforeEach(func() {
						project = &projectv1.Project{
							ObjectMeta: metav1.ObjectMeta{
								Name: "my-project",
							},
							Spec: projectv1.ProjectSpec{
								Access: []projectv1.SubjectRef{
									{
										Kind:      "ServiceAccount",
										Name:      serviceAccountName,
										Namespace: "some-namespace",
									},
								},
							},
						}

						err := fakeClient.Update(context.TODO(), project)
						Expect(err).NotTo(HaveOccurred())
					})

					It("allows the user specified in the project access to the project", func() {
						_, err := reconciler.Reconcile(Request(project.Namespace, project.Name))
						Expect(err).NotTo(HaveOccurred())

						role := &rbacv1.RoleBinding{}
						err = fakeClient.Get(context.TODO(), client.ObjectKey{
							Name:      project.Name + "-rolebinding",
							Namespace: project.Name,
						}, role)
						Expect(err).NotTo(HaveOccurred())

						Expect(role.Name).To(Equal("my-project-rolebinding"))
						Expect(role.ObjectMeta.Namespace).To(Equal("my-project"))

						subject1 := role.Subjects[0]
						Expect(subject1.Kind).To(Equal("ServiceAccount"))
						Expect(subject1.Name).To(Equal(serviceAccountName))
						Expect(subject1.Namespace).To(Equal("some-namespace"))
						Expect(subject1.APIGroup).To(Equal(""))

						roleRef := role.RoleRef
						Expect(roleRef.Kind).To(Equal("Role"))
						Expect(roleRef.Name).To(Equal("my-project-role"))
						Expect(roleRef.APIGroup).To(Equal("rbac.authorization.k8s.io"))
					})
				})

				When("the subject is a User", func() {
					It("allows the user specified in the project access to the project", func() {
						_, err := reconciler.Reconcile(Request(project.Namespace, project.Name))
						Expect(err).NotTo(HaveOccurred())

						role := &rbacv1.RoleBinding{}
						err = fakeClient.Get(context.TODO(), client.ObjectKey{
							Name:      project.Name + "-rolebinding",
							Namespace: project.Name,
						}, role)
						Expect(err).NotTo(HaveOccurred())

						Expect(role.Name).To(Equal("my-project-rolebinding"))
						Expect(role.ObjectMeta.Namespace).To(Equal("my-project"))

						subject1 := role.Subjects[0]
						Expect(subject1.Kind).To(Equal("User"))
						Expect(subject1.Name).To(Equal(user1))
						Expect(subject1.APIGroup).To(Equal("rbac.authorization.k8s.io"))

						subject2 := role.Subjects[1]
						Expect(subject2.Kind).To(Equal("User"))
						Expect(subject2.Name).To(Equal(user2))
						Expect(subject2.APIGroup).To(Equal("rbac.authorization.k8s.io"))

						roleRef := role.RoleRef
						Expect(roleRef.Kind).To(Equal("Role"))
						Expect(roleRef.Name).To(Equal("my-project-role"))
						Expect(roleRef.APIGroup).To(Equal("rbac.authorization.k8s.io"))
					})
				})

				When("the subject is a Group", func() {
					var groupName string

					BeforeEach(func() {
						groupName = "my-group"

						project = &projectv1.Project{
							ObjectMeta: metav1.ObjectMeta{
								Name: "my-project",
							},
							Spec: projectv1.ProjectSpec{
								Access: []projectv1.SubjectRef{
									{
										Kind:      "Group",
										Name:      groupName,
										Namespace: "some-namespace",
									},
								},
							},
						}

						err := fakeClient.Update(context.TODO(), project)
						Expect(err).NotTo(HaveOccurred())
					})

					It("allows the group specified in the project access to the project", func() {
						_, err := reconciler.Reconcile(Request(project.Namespace, project.Name))
						Expect(err).NotTo(HaveOccurred())

						role := &rbacv1.RoleBinding{}
						err = fakeClient.Get(context.TODO(), client.ObjectKey{
							Name:      project.Name + "-rolebinding",
							Namespace: project.Name,
						}, role)
						Expect(err).NotTo(HaveOccurred())

						Expect(role.Name).To(Equal("my-project-rolebinding"))
						Expect(role.ObjectMeta.Namespace).To(Equal("my-project"))

						Expect(role.Subjects).To(HaveLen(1))
						subject1 := role.Subjects[0]
						Expect(subject1.Kind).To(Equal("Group"))
						Expect(subject1.Name).To(Equal(groupName))
						Expect(subject1.APIGroup).To(Equal("rbac.authorization.k8s.io"))

						roleRef := role.RoleRef
						Expect(roleRef.Kind).To(Equal("Role"))
						Expect(roleRef.Name).To(Equal("my-project-role"))
						Expect(roleRef.APIGroup).To(Equal("rbac.authorization.k8s.io"))
					})
				})

				It("owned by the project", func() {
					_, err := reconciler.Reconcile(Request(project.Namespace, project.Name))
					Expect(err).NotTo(HaveOccurred())

					role := &rbacv1.RoleBinding{}
					err = fakeClient.Get(context.TODO(), client.ObjectKey{
						Name:      project.Name + "-rolebinding",
						Namespace: project.Name,
					}, role)
					Expect(err).NotTo(HaveOccurred())

					Expect(role.ObjectMeta.OwnerReferences).To(HaveLen(1))
					ownerReference := role.ObjectMeta.OwnerReferences[0]
					Expect(ownerReference.Name).To(Equal(project.Name))
					Expect(ownerReference.Kind).To(Equal("Project"))
				})
			})
		})

		Describe("update", func() {
			Describe("update a role binding", func() {
				It("that can be updated", func() {
					_, err := reconciler.Reconcile(Request(project.Namespace, project.Name))
					Expect(err).NotTo(HaveOccurred())

					first := project.Spec.Access[0]
					project.Spec.Access = []projectv1.SubjectRef{first}

					err = fakeClient.Update(context.TODO(), project)
					Expect(err).NotTo(HaveOccurred())

					_, err = reconciler.Reconcile(Request(project.Namespace, project.Name))
					Expect(err).NotTo(HaveOccurred())

					updatedRole := &rbacv1.RoleBinding{}
					_ = fakeClient.Get(context.TODO(), client.ObjectKey{
						Name:      project.Name + "-rolebinding",
						Namespace: project.Name,
					}, updatedRole)

					Expect(updatedRole.Subjects).To(HaveLen(1))
					subject1 := updatedRole.Subjects[0]
					Expect(subject1.Kind).To(Equal("User"))
					Expect(subject1.Name).To(Equal(user1))
					Expect(subject1.APIGroup).To(Equal("rbac.authorization.k8s.io"))
				})
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

func Project(projectName string, users ...string) *projectv1.Project {
	subjectRefs := []projectv1.SubjectRef{}

	for _, user := range users {
		subjectRefs = append(subjectRefs, projectv1.SubjectRef{
			Kind: "User",
			Name: user,
		})
	}

	return &projectv1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: projectName,
		},
		Spec: projectv1.ProjectSpec{
			Access: subjectRefs,
		},
	}
}
