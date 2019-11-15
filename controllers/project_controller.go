/*

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	projectv1 "github.com/pivotal/projects-operator/api/v1"
)

type RoleConfiguration struct {
	APIGroups []string
	Resources []string
	Verbs     []string
}

// ProjectReconciler reconciles a Project object
type ProjectReconciler struct {
	client.Client
	Log            logr.Logger
	Scheme         *runtime.Scheme
	ClusterRoleRef rbacv1.RoleRef
}

// +kubebuilder:rbac:groups=developerconsole.pivotal.io,resources=projects,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=developerconsole.pivotal.io,resources=projects/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=watch;list;create;get;update;patch
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles,verbs=watch;list;create;get;update;patch
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=watch;list;create;get;update;patch

func (r *ProjectReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("project", req.NamespacedName)

	project := &projectv1.Project{}

	if err := r.Client.Get(context.Background(), req.NamespacedName, project); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if err := r.createNamespace(project); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.createClusterRole(project); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.createClusterRoleBinding(project); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.createRoleBinding(project); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ProjectReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&projectv1.Project{}).
		Complete(r)
}

func (r *ProjectReconciler) createNamespace(project *projectv1.Project) error {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: project.Name,
		},
	}

	if err := controllerutil.SetControllerReference(project, namespace, r.Scheme); err != nil {
		return err
	}

	status, err := controllerutil.CreateOrUpdate(context.Background(), r, namespace, func() error { return nil })
	if err != nil {
		return err
	}
	r.Log.Info("creating/updating resource", "type", "namespace", "status", status)

	return nil
}

func (r *ProjectReconciler) createClusterRole(project *projectv1.Project) error {
	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterRoleName(project),
		},
	}
	if err := controllerutil.SetControllerReference(project, clusterRole, r.Scheme); err != nil {
		return err
	}

	status, err := controllerutil.CreateOrUpdate(context.Background(), r, clusterRole, func() error {
		clusterRole.Rules = []rbacv1.PolicyRule{
			{
				APIGroups: []string{
					"developerconsole.pivotal.io",
				},
				Resources: []string{
					"projects",
				},
				ResourceNames: []string{
					project.Name,
				},
				Verbs: []string{
					"get",
					"update",
					"delete",
					"patch",
					"watch",
				},
			},
		}
		return nil
	})
	if err != nil {
		return err
	}

	r.Log.Info("creating/updating resource", "type", "clusterrole", "status", status)

	return nil
}

func (r *ProjectReconciler) createClusterRoleBinding(project *projectv1.Project) error {
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: project.Name + "-clusterrolebinding",
		},
	}
	if err := controllerutil.SetControllerReference(project, clusterRoleBinding, r.Scheme); err != nil {
		return err
	}

	status, err := controllerutil.CreateOrUpdate(context.Background(), r, clusterRoleBinding, func() error {
		clusterRoleBinding.Subjects = subjects(project)
		clusterRoleBinding.RoleRef = rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     clusterRoleName(project),
		}
		return nil
	})
	if err != nil {
		return err
	}

	r.Log.Info("creating/updating resource", "type", "clusterrolebinding", "status", status)

	return nil
}

func (r *ProjectReconciler) createRoleBinding(project *projectv1.Project) error {
	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      project.Name + "-rolebinding",
			Namespace: project.Name,
		},
	}
	if err := controllerutil.SetControllerReference(project, roleBinding, r.Scheme); err != nil {
		return err
	}

	status, err := controllerutil.CreateOrUpdate(context.Background(), r, roleBinding, func() error {
		roleBinding.Subjects = subjects(project)
		roleBinding.RoleRef = r.ClusterRoleRef
		return nil
	})
	if err != nil {
		return err
	}

	r.Log.Info("creating/updating resource", "type", "rolebinding", "status", status)

	return nil
}

func clusterRoleName(project *projectv1.Project) string {
	return project.Name + "-clusterrole"
}

func subjects(project *projectv1.Project) []rbacv1.Subject {
	var subjects []rbacv1.Subject
	for _, userRef := range project.Spec.Access {

		apiGroup := ""
		if userRef.Kind == "User" || userRef.Kind == "Group" {
			apiGroup = "rbac.authorization.k8s.io"
		}
		subjects = append(subjects, rbacv1.Subject{
			Kind:      string(userRef.Kind),
			Name:      userRef.Name,
			Namespace: userRef.Namespace,
			APIGroup:  apiGroup,
		})
	}
	return subjects
}
