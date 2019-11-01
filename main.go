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

package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	projectv1 "github.com/pivotal/projects-operator/api/v1"

	"github.com/pivotal/projects-operator/controllers"

	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

var requiredEnvVars = []string{
	"ROLE_1_APIGROUPS",
	"ROLE_1_RESOURCES",
	"ROLE_1_VERBS",
	"ROLE_2_APIGROUPS",
	"ROLE_2_RESOURCES",
	"ROLE_2_VERBS",
}

func init() {
	_ = clientgoscheme.AddToScheme(scheme)

	_ = projectv1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
	flag.Parse()

	ctrl.SetLogger(zap.Logger(true))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		LeaderElection:     enableLeaderElection,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if requiredEnvVarsMissing() {
		err = errors.New("ROLE_{1,2,3}_APIGROUPS, ROLE_{1,2,3}_RESOURCES and ROLE_{1,2,3}_VERBS envs must be set")
		setupLog.Error(err, "unable to create controller", "controller", "Project")
		os.Exit(1)
	}

	roleConfigs := controllers.RoleConfigurations{}
	for i := 0; i < 2; i++ {
		apiGroupsEnv := os.Getenv(fmt.Sprintf("ROLE_%d_APIGROUPS", i+1))
		resourcesEnv := os.Getenv(fmt.Sprintf("ROLE_%d_RESOURCES", i+1))
		verbsEnv := os.Getenv(fmt.Sprintf("ROLE_%d_VERBS", i+1))

		roleConfigs = append(roleConfigs, controllers.RoleConfiguration{
			APIGroups: strings.Split(strings.Trim(apiGroupsEnv, ""), ","),
			Resources: strings.Split(resourcesEnv, ","),
			Verbs:     strings.Split(verbsEnv, ","),
		})
	}

	if err = (&controllers.ProjectReconciler{
		Client:      mgr.GetClient(),
		Log:         ctrl.Log.WithName("controllers").WithName("Project"),
		Scheme:      scheme,
		RoleConfigs: roleConfigs,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Project")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func requiredEnvVarsMissing() bool {
	for _, requiredEnvVar := range requiredEnvVars {
		if _, present := os.LookupEnv(requiredEnvVar); !present {
			return true
		}
	}

	return false
}
