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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	marketplacev1 "github.com/pivotal-cf/marketplace-project/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ProjectReconciler reconciles a Project object
type ProjectReconciler struct {
	client.Client
	Log logr.Logger
}

// +kubebuilder:rbac:groups=marketplace.pivotal.io,resources=projects,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=marketplace.pivotal.io,resources=projects/status,verbs=get;update;patch

func (r *ProjectReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("project", req.NamespacedName)

	project := &marketplacev1.Project{}
	r.Client.Get(context.TODO(), req.NamespacedName, project)

	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: project.Name,
		},
	}

	_, err := controllerutil.CreateOrUpdate(context.TODO(), r, namespace, func() error { return nil })
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *ProjectReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&marketplacev1.Project{}).
		Complete(r)
}
