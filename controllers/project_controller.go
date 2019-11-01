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

	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	projectv1 "github.com/pivotal/projects-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type RoleConfiguration struct {
	APIGroups []string
	Resources []string
	Verbs     []string
}

type RoleConfigurations []RoleConfiguration

// ProjectReconciler reconciles a Project object
type ProjectReconciler struct {
	client.Client
	Log         logr.Logger
	Scheme      *runtime.Scheme
	RoleConfigs RoleConfigurations
}

// +kubebuilder:rbac:groups=marketplace.pivotal.io,resources=projects,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=marketplace.pivotal.io,resources=projects/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=watch;list;create;get;update;patch
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles,verbs=watch;list;create;get;update;patch
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=watch;list;create;get;update;patch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get
// +kubebuilder:rbac:groups=servicecatalog.k8s.io,resources=servicebindings,verbs=*
// +kubebuilder:rbac:groups=servicecatalog.k8s.io,resources=serviceinstances,verbs=*
// +kubebuilder:rbac:groups=servicecatalog.k8s.io,resources=clusterservicebrokers,verbs=list
// +kubebuilder:rbac:groups=servicecatalog.k8s.io,resources=clusterserviceclasses,verbs=list
// +kubebuilder:rbac:groups=servicecatalog.k8s.io,resources=clusterserviceplans,verbs=list

func (r *ProjectReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("project", req.NamespacedName)

	project := &projectv1.Project{}

	if err := r.Client.Get(context.TODO(), req.NamespacedName, project); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if err := r.createNamespace(project); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.createRole(project); err != nil {
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

	status, err := controllerutil.CreateOrUpdate(context.TODO(), r, namespace, func() error { return nil })
	if err != nil {
		return err
	}
	r.Log.Info("creating/updating resource", "type", "namespace", "status", status)

	return nil
}

func (r *ProjectReconciler) createRole(project *projectv1.Project) error {
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      project.Name + "-role",
			Namespace: project.Name,
		},
	}

	if err := controllerutil.SetControllerReference(project, role, r.Scheme); err != nil {
		return err
	}
	status, err := controllerutil.CreateOrUpdate(context.TODO(), r, role, func() error {
		for _, roleConfig := range r.RoleConfigs {
			role.Rules = append(role.Rules, rbacv1.PolicyRule{
				APIGroups: roleConfig.APIGroups,
				Resources: roleConfig.Resources,
				Verbs:     roleConfig.Verbs,
			})
		}

		return nil
	})
	if err != nil {
		return err
	}
	r.Log.Info("creating/updating resource", "type", "role", "status", status)

	return nil
}

func (r *ProjectReconciler) createRoleBinding(project *projectv1.Project) error {
	subjects := []rbacv1.Subject{}
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

	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      project.Name + "-rolebinding",
			Namespace: project.Name,
		},
	}
	if err := controllerutil.SetControllerReference(project, roleBinding, r.Scheme); err != nil {
		return err
	}

	status, err := controllerutil.CreateOrUpdate(context.TODO(), r, roleBinding, func() error {
		roleBinding.Subjects = subjects
		roleBinding.RoleRef = rbacv1.RoleRef{
			Kind:     "Role",
			Name:     project.Name + "-role",
			APIGroup: "rbac.authorization.k8s.io",
		}
		return nil
	})
	if err != nil {
		return err
	}
	r.Log.Info("creating/updating resource", "type", "rolebinding", "status", status)

	return nil
}
