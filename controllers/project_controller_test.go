/*
Unauthorized use, copying or distribution of any source code in this
repository via any medium is strictly prohibited without the author's
express written consent.

ANY AUTHORIZED USE OF OR ACCESS TO THE SOFTWARE IS "AS IS", WITHOUT
WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO
THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE
LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION
OF CONTRACT,TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION
WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/

package controllers_test

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	projectv1alpha1 "github.com/pivotal/projects-operator/api/v1alpha1"
	"github.com/pivotal/projects-operator/controllers"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ProjectController", func() {
	Describe("Reconcile", func() {
		var (
			reconciler     controllers.ProjectReconciler
			fakeClient     client.Client
			project        *projectv1alpha1.Project
			user1          string
			user2          string
			scheme         *runtime.Scheme
			clusterRoleRef rbacv1.RoleRef
		)

		BeforeEach(func() {
			scheme = runtime.NewScheme()

			projectv1alpha1.AddToScheme(scheme)
			corev1.AddToScheme(scheme)
			rbacv1.AddToScheme(scheme)

			user1 = "some-user1"
			user2 = "some-user2"
			project = Project("my-project", user1, user2)

			fakeClient = fake.NewFakeClientWithScheme(scheme, project)

			clusterRoleRef = rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io/v1",
				Kind:     "ClusterRole",
				Name:     "some-cluster-role",
			}

			reconciler = controllers.ProjectReconciler{
				Log:            ctrl.Log.WithName("controllers").WithName("Project"),
				Client:         fakeClient,
				Scheme:         scheme,
				ClusterRoleRef: clusterRoleRef,
			}
		})

		Describe("deletion", func() {
			It("deletes a project without errors", func() {
				_, err := reconciler.Reconcile(Request(project.Namespace, project.Name))
				Expect(err).NotTo(HaveOccurred())

				err = fakeClient.Delete(context.TODO(), project)
				Expect(err).NotTo(HaveOccurred())

				_, err = reconciler.Reconcile(Request(project.Namespace, project.Name))
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Describe("creation", func() {
			Describe("updates the project", func() {
				It("adds a finalizer for waiting for namespace deletion", func() {
					_, err := reconciler.Reconcile(Request(project.Namespace, project.Name))
					Expect(err).NotTo(HaveOccurred())

					updatedProject := &projectv1alpha1.Project{}
					err = fakeClient.Get(context.TODO(), client.ObjectKey{
						Name: project.Name,
					}, updatedProject)
					Expect(err).NotTo(HaveOccurred())
					Expect(updatedProject.Finalizers).To(ConsistOf("wait-for-namespace-to-be-deleted"))
				})
			})

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

			Describe("creates a cluster role", func() {
				It("with given project name", func() {
					_, err := reconciler.Reconcile(Request(project.Namespace, project.Name))
					Expect(err).NotTo(HaveOccurred())

					clusterRole := &rbacv1.ClusterRole{}
					err = fakeClient.Get(context.TODO(), client.ObjectKey{
						Name: project.Name + "-clusterrole",
					}, clusterRole)
					Expect(err).NotTo(HaveOccurred())

					Expect(clusterRole.Name).To(Equal("my-project-clusterrole"))
				})

				It("owned by the project", func() {
					_, err := reconciler.Reconcile(Request(project.Namespace, project.Name))
					Expect(err).NotTo(HaveOccurred())

					clusterRole := &rbacv1.ClusterRole{}
					err = fakeClient.Get(context.TODO(), client.ObjectKey{
						Name: project.Name + "-clusterrole",
					}, clusterRole)
					Expect(err).NotTo(HaveOccurred())

					Expect(clusterRole.ObjectMeta.OwnerReferences).To(HaveLen(1))
					ownerReference := clusterRole.ObjectMeta.OwnerReferences[0]
					Expect(ownerReference.Name).To(Equal(project.Name))
					Expect(ownerReference.Kind).To(Equal("Project"))
				})

				It("has rules to access the project", func() {
					_, err := reconciler.Reconcile(Request(project.Namespace, project.Name))
					Expect(err).NotTo(HaveOccurred())

					clusterRole := &rbacv1.ClusterRole{}
					err = fakeClient.Get(context.TODO(), client.ObjectKey{
						Name: project.Name + "-clusterrole",
					}, clusterRole)
					Expect(err).NotTo(HaveOccurred())

					Expect(clusterRole.Rules).To(HaveLen(1))
					Expect(clusterRole.Rules[0].APIGroups[0]).To(Equal("developerconsole.pivotal.io"))
					Expect(clusterRole.Rules[0].Resources[0]).To(Equal("projects"))
					Expect(clusterRole.Rules[0].ResourceNames[0]).To(Equal(project.Name))
					Expect(clusterRole.Rules[0].Verbs).To(Equal([]string{"get", "update", "delete", "patch", "watch"}))
				})
			})

			Describe("creates a role binding", func() {
				When("the subject is a ServiceAccount", func() {
					var serviceAccountName = "service-account"

					BeforeEach(func() {
						project = &projectv1alpha1.Project{
							ObjectMeta: metav1.ObjectMeta{
								Name: "my-project",
							},
							Spec: projectv1alpha1.ProjectSpec{
								Access: []projectv1alpha1.SubjectRef{
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

					It("allows the user specified in the project access to the namespace", func() {
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

						Expect(role.RoleRef).To(Equal(clusterRoleRef))
					})

					It("allows the user specified in the project access to the project", func() {
						_, err := reconciler.Reconcile(Request(project.Namespace, project.Name))
						Expect(err).NotTo(HaveOccurred())

						clusterRole := &rbacv1.ClusterRoleBinding{}
						err = fakeClient.Get(context.TODO(), client.ObjectKey{
							Name: project.Name + "-clusterrolebinding",
						}, clusterRole)
						Expect(err).NotTo(HaveOccurred())

						Expect(clusterRole.Name).To(Equal("my-project-clusterrolebinding"))

						subject1 := clusterRole.Subjects[0]
						Expect(subject1.Kind).To(Equal("ServiceAccount"))
						Expect(subject1.Name).To(Equal(serviceAccountName))
						Expect(subject1.APIGroup).To(Equal(""))

						Expect(clusterRole.RoleRef).To(Equal(rbacv1.RoleRef{
							APIGroup: "rbac.authorization.k8s.io",
							Kind:     "ClusterRole",
							Name:     project.Name + "-clusterrole",
						}))
					})
				})

				When("the subject is a User", func() {
					It("allows the user specified in the project access to the namespace", func() {
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

						Expect(role.RoleRef).To(Equal(clusterRoleRef))
					})

					It("allows the user specified in the project access to the project", func() {
						_, err := reconciler.Reconcile(Request(project.Namespace, project.Name))
						Expect(err).NotTo(HaveOccurred())

						clusterRole := &rbacv1.ClusterRoleBinding{}
						err = fakeClient.Get(context.TODO(), client.ObjectKey{
							Name: project.Name + "-clusterrolebinding",
						}, clusterRole)
						Expect(err).NotTo(HaveOccurred())

						Expect(clusterRole.Name).To(Equal("my-project-clusterrolebinding"))

						subject1 := clusterRole.Subjects[0]
						Expect(subject1.Kind).To(Equal("User"))
						Expect(subject1.Name).To(Equal(user1))
						Expect(subject1.APIGroup).To(Equal("rbac.authorization.k8s.io"))

						Expect(clusterRole.RoleRef).To(Equal(rbacv1.RoleRef{
							APIGroup: "rbac.authorization.k8s.io",
							Kind:     "ClusterRole",
							Name:     project.Name + "-clusterrole",
						}))
					})
				})

				When("the subject is a Group", func() {
					var groupName string

					BeforeEach(func() {
						groupName = "my-group"

						project = &projectv1alpha1.Project{
							ObjectMeta: metav1.ObjectMeta{
								Name: "my-project",
							},
							Spec: projectv1alpha1.ProjectSpec{
								Access: []projectv1alpha1.SubjectRef{
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

					It("allows the group specified in the project access to the namespace", func() {
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

						Expect(role.RoleRef).To(Equal(clusterRoleRef))
					})

					It("allows the group specified in the project access to the project", func() {
						_, err := reconciler.Reconcile(Request(project.Namespace, project.Name))
						Expect(err).NotTo(HaveOccurred())

						clusterRole := &rbacv1.ClusterRoleBinding{}
						err = fakeClient.Get(context.TODO(), client.ObjectKey{
							Name: project.Name + "-clusterrolebinding",
						}, clusterRole)
						Expect(err).NotTo(HaveOccurred())

						Expect(clusterRole.Name).To(Equal("my-project-clusterrolebinding"))

						subject1 := clusterRole.Subjects[0]
						Expect(subject1.Kind).To(Equal("Group"))
						Expect(subject1.Name).To(Equal(groupName))
						Expect(subject1.APIGroup).To(Equal("rbac.authorization.k8s.io"))

						Expect(clusterRole.RoleRef).To(Equal(rbacv1.RoleRef{
							APIGroup: "rbac.authorization.k8s.io",
							Kind:     "ClusterRole",
							Name:     project.Name + "-clusterrole",
						}))
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

					clusterRole := &rbacv1.ClusterRoleBinding{}
					err = fakeClient.Get(context.TODO(), client.ObjectKey{
						Name: project.Name + "-clusterrolebinding",
					}, clusterRole)
					Expect(err).NotTo(HaveOccurred())

					Expect(clusterRole.ObjectMeta.OwnerReferences).To(HaveLen(1))
					ownerReference = clusterRole.ObjectMeta.OwnerReferences[0]
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
					project.Spec.Access = []projectv1alpha1.SubjectRef{first}

					err = fakeClient.Update(context.TODO(), project)
					Expect(err).NotTo(HaveOccurred())

					_, err = reconciler.Reconcile(Request(project.Namespace, project.Name))
					Expect(err).NotTo(HaveOccurred())

					updatedRole := &rbacv1.RoleBinding{}
					err = fakeClient.Get(context.TODO(), client.ObjectKey{
						Name:      project.Name + "-rolebinding",
						Namespace: project.Name,
					}, updatedRole)
					Expect(err).NotTo(HaveOccurred())

					Expect(updatedRole.Subjects).To(HaveLen(1))
					subject1 := updatedRole.Subjects[0]
					Expect(subject1.Kind).To(Equal("User"))
					Expect(subject1.Name).To(Equal(user1))
					Expect(subject1.APIGroup).To(Equal("rbac.authorization.k8s.io"))

					updatedClusterRole := &rbacv1.ClusterRoleBinding{}
					err = fakeClient.Get(context.TODO(), client.ObjectKey{
						Name: project.Name + "-clusterrolebinding",
					}, updatedClusterRole)
					Expect(err).NotTo(HaveOccurred())

					Expect(updatedClusterRole.Subjects).To(HaveLen(1))
					subject1 = updatedClusterRole.Subjects[0]
					Expect(subject1.Kind).To(Equal("User"))
					Expect(subject1.Name).To(Equal(user1))
					Expect(subject1.APIGroup).To(Equal("rbac.authorization.k8s.io"))
				})
			})

			Describe("finalizer removal", func() {
				It("deletes the namespace and removes the finalizer when a deletion timestamp is present", func() {
					_, err := reconciler.Reconcile(Request(project.Namespace, project.Name))
					Expect(err).NotTo(HaveOccurred())

					project.DeletionTimestamp = &metav1.Time{Time: time.Now()}

					err = fakeClient.Update(context.TODO(), project)
					Expect(err).NotTo(HaveOccurred())

					_, err = reconciler.Reconcile(Request(project.Namespace, project.Name))
					Expect(err).NotTo(HaveOccurred())

					namespace := &corev1.Namespace{}
					err = fakeClient.Get(context.TODO(), client.ObjectKey{
						Name: project.Name,
					}, namespace)
					Expect(errors.IsNotFound(err)).To(BeTrue())

					updatedProject := &projectv1alpha1.Project{}
					err = fakeClient.Get(context.TODO(), client.ObjectKey{
						Name: project.Name,
					}, updatedProject)
					Expect(err).NotTo(HaveOccurred())
					Expect(updatedProject.Finalizers).To(BeEmpty())
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

func Project(projectName string, users ...string) *projectv1alpha1.Project {
	subjectRefs := []projectv1alpha1.SubjectRef{}

	for _, user := range users {
		subjectRefs = append(subjectRefs, projectv1alpha1.SubjectRef{
			Kind: "User",
			Name: user,
		})
	}

	return &projectv1alpha1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: projectName,
		},
		Spec: projectv1alpha1.ProjectSpec{
			Access: subjectRefs,
		},
	}
}
