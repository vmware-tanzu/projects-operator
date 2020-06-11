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

package controllers

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	projects "github.com/pivotal/projects-operator/api/v1alpha1"
	"github.com/pivotal/projects-operator/pkg/finalizer"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const projectFinalizer = "project.finalizer.projects.vmware.com"

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

// +kubebuilder:rbac:groups=projects.vmware.com,resources=projects,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=projects.vmware.com,resources=projects/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=watch;list;create;get;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles;roles,verbs=watch;list;create;get;update;patch
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterrolebindings;rolebindings,verbs=watch;list;create;get;update;patch

func (r *ProjectReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	logger := r.Log.WithValues("project", req.NamespacedName)

	project := &projects.Project{}

	if err := r.Client.Get(context.Background(), req.NamespacedName, project); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("Project resource not found")
			return ctrl.Result{Requeue: true}, client.IgnoreNotFound(err)
		}

		logger.Error(err, "unable to fetch Project")
		return ctrl.Result{}, err
	}

	if !project.ObjectMeta.DeletionTimestamp.IsZero() {
		err := r.deleteNamespace(project)
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

	err := r.addFinalizer(project)

	return ctrl.Result{}, err
}

func (r *ProjectReconciler) SetupWithManager(mgr ctrl.Manager, maxConcurrentReconciles int) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&projects.Project{}).
		WithOptions(controller.Options{MaxConcurrentReconciles: maxConcurrentReconciles}).
		Complete(r)
}

func (r *ProjectReconciler) createNamespace(project *projects.Project) error {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   project.Name,
			Labels: project.Labels,
		},
	}

	if err := controllerutil.SetOwnerReference(project, namespace, r.Scheme); err != nil {
		return err
	}

	status, err := controllerutil.CreateOrUpdate(context.Background(), r, namespace, func() error { return nil })
	if err != nil {
		return err
	}
	r.Log.Info("creating/updating resource", "type", "namespace", "status", status)

	return nil
}

func (r *ProjectReconciler) deleteNamespace(project *projects.Project) error {
	_ = r.Client.Delete(context.Background(), &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: project.Name}})
	for {
		namespace := &corev1.Namespace{}
		if err := r.Client.Get(context.Background(), types.NamespacedName{Name: project.Name}, namespace); err != nil {
			if errors.IsNotFound(err) {
				status, err := controllerutil.CreateOrUpdate(context.Background(), r, project, func() error {
					finalizer.RemoveFinalizer(project, projectFinalizer)
					return nil
				})
				if err != nil {
					return err
				}
				r.Log.Info("creating/updating resource", "type", "project", "status", status)
				return nil
			}
			time.Sleep(time.Second)
		}
	}
}

func (r *ProjectReconciler) createClusterRole(project *projects.Project) error {
	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterRoleName(project),
		},
	}
	if err := controllerutil.SetOwnerReference(project, clusterRole, r.Scheme); err != nil {
		return err
	}

	status, err := controllerutil.CreateOrUpdate(context.Background(), r, clusterRole, func() error {
		clusterRole.Rules = []rbacv1.PolicyRule{
			{
				APIGroups: []string{
					"projects.vmware.com",
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

func (r *ProjectReconciler) createClusterRoleBinding(project *projects.Project) error {
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: project.Name + "-clusterrolebinding",
		},
	}
	if err := controllerutil.SetOwnerReference(project, clusterRoleBinding, r.Scheme); err != nil {
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

func (r *ProjectReconciler) createRoleBinding(project *projects.Project) error {
	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      project.Name + "-rolebinding",
			Namespace: project.Name,
		},
	}
	if err := controllerutil.SetOwnerReference(project, roleBinding, r.Scheme); err != nil {
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

func clusterRoleName(project *projects.Project) string {
	return project.Name + "-clusterrole"
}

func subjects(project *projects.Project) []rbacv1.Subject {
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

func (r *ProjectReconciler) addFinalizer(project *projects.Project) error {
	status, err := controllerutil.CreateOrUpdate(context.Background(), r, project, func() error {
		finalizer.AddFinalizer(project, projectFinalizer)
		return nil
	})
	if err != nil {
		return err
	}
	r.Log.Info("creating/updating resource", "type", "project", "status", status)
	return nil
}
