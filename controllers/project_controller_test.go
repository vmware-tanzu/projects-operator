// Copyright 2019-2020 VMware, Inc.
// SPDX-License-Identifier: BSD-2-Clause

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

	projects "github.com/pivotal/projects-operator/api/v1alpha1"
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
	. "github.com/pivotal/projects-operator/controllers"
)

var _ = Describe("ProjectController", func() {
	Describe("Reconcile", func() {
		var (
			reconciler     ProjectReconciler
			fakeClient     client.Client
			project        *projects.Project
			labels         map[string]string
			user1          string
			user2          string
			scheme         *runtime.Scheme
			clusterRoleRef rbacv1.RoleRef
			ctx            context.Context
		)

		BeforeEach(func() {
			scheme = runtime.NewScheme()

			projects.AddToScheme(scheme)
			corev1.AddToScheme(scheme)
			rbacv1.AddToScheme(scheme)

			user1 = "some-user1"
			user2 = "some-user2"
			labels = map[string]string{"some.org/some.key": "some-value", "other.org/other.key": "other-value"}
			project = Project("my-project", labels, user1, user2)

			fakeClient = fake.NewFakeClientWithScheme(scheme, project)

			clusterRoleRef = rbacv1.RoleRef{
				APIGroup: "rbac.authorization.k8s.io/v1",
				Kind:     "ClusterRole",
				Name:     "some-cluster-role",
			}

			reconciler = ProjectReconciler{
				Log:            ctrl.Log.WithName("controllers").WithName("Project"),
				Client:         fakeClient,
				Scheme:         scheme,
				ClusterRoleRef: clusterRoleRef,
			}
			ctx = context.Background()
		})

		Describe("deletion", func() {
			It("deletes a project without errors", func() {
				_, err := reconciler.Reconcile(ctx, Request(project.Namespace, project.Name))
				Expect(err).NotTo(HaveOccurred())

				err = fakeClient.Delete(ctx, project)
				Expect(err).NotTo(HaveOccurred())

				_, err = reconciler.Reconcile(ctx, Request(project.Namespace, project.Name))
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Describe("creation", func() {
			Describe("updates the project", func() {
				It("adds a finalizer for waiting for namespace deletion", func() {
					_, err := reconciler.Reconcile(ctx, Request(project.Namespace, project.Name))
					Expect(err).NotTo(HaveOccurred())

					updatedProject := &projects.Project{}
					err = fakeClient.Get(ctx, client.ObjectKey{
						Name: project.Name,
					}, updatedProject)
					Expect(err).NotTo(HaveOccurred())
					Expect(updatedProject.Finalizers).To(ConsistOf("project.finalizer.projects.vmware.com"))
				})
			})

			Describe("getting the project", func() {
				var (
					result ctrl.Result
					err    error
				)

				BeforeEach(func() {
					result, err = reconciler.Reconcile(ctx, Request(project.Namespace, project.Name))
				})

				It("doesn't error", func() {
					Expect(err).NotTo(HaveOccurred())
				})

				It("doesn't requeue", func() {
					Expect(result.Requeue).To(BeFalse())
				})

				It("gets the correct project", func() {
					err = fakeClient.Get(ctx, client.ObjectKey{
						Name: project.Name,
					}, project)
					Expect(err).NotTo(HaveOccurred())

					Expect(project.Name).To(Equal("my-project"))
				})

				When("the project doesn't yet exist", func() {
					BeforeEach(func() {
						project = &projects.Project{
							ObjectMeta: metav1.ObjectMeta{
								Name: "new-project",
							},
						}
						result, err = reconciler.Reconcile(ctx, Request(project.Namespace, project.Name))
					})

					It("doesn't error", func() {
						Expect(err).NotTo(HaveOccurred())
					})

					It("does requeue", func() {
						Expect(result.Requeue).To(BeTrue())
					})
				})
			})

			Describe("creates a namespace", func() {
				It("with given project name", func() {
					_, err := reconciler.Reconcile(ctx, Request(project.Namespace, project.Name))
					Expect(err).NotTo(HaveOccurred())

					namespace := &corev1.Namespace{}
					err = fakeClient.Get(ctx, client.ObjectKey{
						Name: project.Name,
					}, namespace)
					Expect(err).NotTo(HaveOccurred())

					Expect(namespace.Name).To(Equal("my-project"))
				})

				It("owned by the project", func() {
					_, err := reconciler.Reconcile(ctx, Request(project.Namespace, project.Name))
					Expect(err).NotTo(HaveOccurred())

					namespace := &corev1.Namespace{}
					err = fakeClient.Get(ctx, client.ObjectKey{
						Name: project.Name,
					}, namespace)
					Expect(err).NotTo(HaveOccurred())

					Expect(namespace.ObjectMeta.OwnerReferences).To(HaveLen(1))
					ownerReference := namespace.ObjectMeta.OwnerReferences[0]
					Expect(ownerReference.Name).To(Equal(project.Name))
					Expect(ownerReference.Kind).To(Equal("Project"))
				})

				It("copies project labels to the namespace", func() {
					_, err := reconciler.Reconcile(ctx, Request(project.Namespace, project.Name))
					Expect(err).NotTo(HaveOccurred())

					namespace := &corev1.Namespace{}
					err = fakeClient.Get(ctx, client.ObjectKey{
						Name: project.Name,
					}, namespace)
					Expect(err).NotTo(HaveOccurred())

					Expect(namespace.Labels).To(Equal(labels))
				})
			})

			Describe("creates a cluster role", func() {
				It("with given project name", func() {
					_, err := reconciler.Reconcile(ctx, Request(project.Namespace, project.Name))
					Expect(err).NotTo(HaveOccurred())

					clusterRole := &rbacv1.ClusterRole{}
					err = fakeClient.Get(ctx, client.ObjectKey{
						Name: project.Name + "-clusterrole",
					}, clusterRole)
					Expect(err).NotTo(HaveOccurred())

					Expect(clusterRole.Name).To(Equal("my-project-clusterrole"))
				})

				It("owned by the project", func() {
					_, err := reconciler.Reconcile(ctx, Request(project.Namespace, project.Name))
					Expect(err).NotTo(HaveOccurred())

					clusterRole := &rbacv1.ClusterRole{}
					err = fakeClient.Get(ctx, client.ObjectKey{
						Name: project.Name + "-clusterrole",
					}, clusterRole)
					Expect(err).NotTo(HaveOccurred())

					Expect(clusterRole.ObjectMeta.OwnerReferences).To(HaveLen(1))
					ownerReference := clusterRole.ObjectMeta.OwnerReferences[0]
					Expect(ownerReference.Name).To(Equal(project.Name))
					Expect(ownerReference.Kind).To(Equal("Project"))
				})

				It("has rules to access the project", func() {
					_, err := reconciler.Reconcile(ctx, Request(project.Namespace, project.Name))
					Expect(err).NotTo(HaveOccurred())

					clusterRole := &rbacv1.ClusterRole{}
					err = fakeClient.Get(ctx, client.ObjectKey{
						Name: project.Name + "-clusterrole",
					}, clusterRole)
					Expect(err).NotTo(HaveOccurred())

					Expect(clusterRole.Rules).To(HaveLen(1))
					Expect(clusterRole.Rules[0].APIGroups[0]).To(Equal("projects.vmware.com"))
					Expect(clusterRole.Rules[0].Resources[0]).To(Equal("projects"))
					Expect(clusterRole.Rules[0].ResourceNames[0]).To(Equal(project.Name))
					Expect(clusterRole.Rules[0].Verbs).To(Equal([]string{"get", "update", "delete", "patch", "watch"}))
				})
			})

			Describe("creates a role binding", func() {
				When("the subject is a ServiceAccount", func() {
					var serviceAccountName = "service-account"

					BeforeEach(func() {
						project = &projects.Project{
							ObjectMeta: metav1.ObjectMeta{
								Name: "my-project",
							},
							Spec: projects.ProjectSpec{
								Access: []projects.SubjectRef{
									{
										Kind:      "ServiceAccount",
										Name:      serviceAccountName,
										Namespace: "some-namespace",
									},
								},
							},
						}

						err := fakeClient.Update(ctx, project)
						Expect(err).NotTo(HaveOccurred())
					})

					It("allows the user specified in the project access to the namespace", func() {
						_, err := reconciler.Reconcile(ctx, Request(project.Namespace, project.Name))
						Expect(err).NotTo(HaveOccurred())

						role := &rbacv1.RoleBinding{}
						err = fakeClient.Get(ctx, client.ObjectKey{
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
						_, err := reconciler.Reconcile(ctx, Request(project.Namespace, project.Name))
						Expect(err).NotTo(HaveOccurred())

						clusterRole := &rbacv1.ClusterRoleBinding{}
						err = fakeClient.Get(ctx, client.ObjectKey{
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
						_, err := reconciler.Reconcile(ctx, Request(project.Namespace, project.Name))
						Expect(err).NotTo(HaveOccurred())

						role := &rbacv1.RoleBinding{}
						err = fakeClient.Get(ctx, client.ObjectKey{
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
						_, err := reconciler.Reconcile(ctx, Request(project.Namespace, project.Name))
						Expect(err).NotTo(HaveOccurred())

						clusterRole := &rbacv1.ClusterRoleBinding{}
						err = fakeClient.Get(ctx, client.ObjectKey{
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

						project = &projects.Project{
							ObjectMeta: metav1.ObjectMeta{
								Name: "my-project",
							},
							Spec: projects.ProjectSpec{
								Access: []projects.SubjectRef{
									{
										Kind:      "Group",
										Name:      groupName,
										Namespace: "some-namespace",
									},
								},
							},
						}

						err := fakeClient.Update(ctx, project)
						Expect(err).NotTo(HaveOccurred())
					})

					It("allows the group specified in the project access to the namespace", func() {
						_, err := reconciler.Reconcile(ctx, Request(project.Namespace, project.Name))
						Expect(err).NotTo(HaveOccurred())

						role := &rbacv1.RoleBinding{}
						err = fakeClient.Get(ctx, client.ObjectKey{
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
						_, err := reconciler.Reconcile(ctx, Request(project.Namespace, project.Name))
						Expect(err).NotTo(HaveOccurred())

						clusterRole := &rbacv1.ClusterRoleBinding{}
						err = fakeClient.Get(ctx, client.ObjectKey{
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
					_, err := reconciler.Reconcile(ctx, Request(project.Namespace, project.Name))
					Expect(err).NotTo(HaveOccurred())

					role := &rbacv1.RoleBinding{}
					err = fakeClient.Get(ctx, client.ObjectKey{
						Name:      project.Name + "-rolebinding",
						Namespace: project.Name,
					}, role)
					Expect(err).NotTo(HaveOccurred())

					Expect(role.ObjectMeta.OwnerReferences).To(HaveLen(1))
					ownerReference := role.ObjectMeta.OwnerReferences[0]
					Expect(ownerReference.Name).To(Equal(project.Name))
					Expect(ownerReference.Kind).To(Equal("Project"))

					clusterRole := &rbacv1.ClusterRoleBinding{}
					err = fakeClient.Get(ctx, client.ObjectKey{
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
					_, err := reconciler.Reconcile(ctx, Request(project.Namespace, project.Name))
					Expect(err).NotTo(HaveOccurred())

					// need to retrieve the reconciled project so that it has the correct ResourceVersion
					reconciledProject := &projects.Project{}
					err = fakeClient.Get(ctx, client.ObjectKey{
						Namespace: project.Namespace,
						Name:      project.Name,
					}, reconciledProject)
					Expect(err).NotTo(HaveOccurred())

					first := reconciledProject.Spec.Access[0]
					reconciledProject.Spec.Access = []projects.SubjectRef{first}

					err = fakeClient.Update(ctx, reconciledProject)
					Expect(err).NotTo(HaveOccurred())

					_, err = reconciler.Reconcile(ctx, Request(project.Namespace, project.Name))
					Expect(err).NotTo(HaveOccurred())

					updatedRole := &rbacv1.RoleBinding{}
					err = fakeClient.Get(ctx, client.ObjectKey{
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
					err = fakeClient.Get(ctx, client.ObjectKey{
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
					_, err := reconciler.Reconcile(ctx, Request(project.Namespace, project.Name))
					Expect(err).NotTo(HaveOccurred())

					// need to retrieve the reconciled project so that it has the correct ResourceVersion
					reconciledProject := &projects.Project{}
					err = fakeClient.Get(ctx, client.ObjectKey{
						Namespace: project.Namespace,
						Name:      project.Name,
					}, reconciledProject)
					Expect(err).NotTo(HaveOccurred())

					reconciledProject.DeletionTimestamp = &metav1.Time{Time: time.Now()}

					err = fakeClient.Update(ctx, reconciledProject)
					Expect(err).NotTo(HaveOccurred())

					_, err = reconciler.Reconcile(ctx, Request(project.Namespace, project.Name))
					Expect(err).NotTo(HaveOccurred())

					namespace := &corev1.Namespace{}
					err = fakeClient.Get(ctx, client.ObjectKey{
						Name: project.Name,
					}, namespace)
					Expect(errors.IsNotFound(err)).To(BeTrue())

					updatedProject := &projects.Project{}
					err = fakeClient.Get(ctx, client.ObjectKey{
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

func Project(projectName string, labels map[string]string, users ...string) *projects.Project {
	subjectRefs := []projects.SubjectRef{}

	for _, user := range users {
		subjectRefs = append(subjectRefs, projects.SubjectRef{
			Kind: "User",
			Name: user,
		})
	}

	return &projects.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name:   projectName,
			Labels: labels,
		},
		Spec: projects.ProjectSpec{
			Access: subjectRefs,
		},
	}
}
