/*
Copyright 2021.

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
	"context"
	"flag"
	"fmt"
	"time"

	apisixv2 "github.com/goodrain/rainbond-operator/api/v2"
	"os"
	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "github.com/go-sql-driver/mysql"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	kubeaggregatorv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	rainbondiov1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
	"github.com/goodrain/rainbond-operator/controllers"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(apisixv2.AddToScheme(scheme))

	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(rainbondiov1alpha1.AddToScheme(scheme))

	utilruntime.Must(kubeaggregatorv1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "c3e7a49c.rainbond.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}
	if err = (&controllers.RainbondClusterReconciler{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("controllers").WithName("RainbondCluster"),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("RainbondCluster"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "RainbondCluster")
		os.Exit(1)
	}
	if err = (&controllers.RbdComponentReconciler{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("controllers").WithName("RbdComponent"),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("RbdComponent"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "RbdComponent")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("health", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("check", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	// Trigger reconcile for all RainbondClusters on operator startup
	// This ensures monitoring resources are created after operator upgrade
	if err := triggerRainbondClusterReconcile(mgr); err != nil {
		setupLog.Error(err, "failed to trigger RainbondCluster reconcile on startup")
		// Don't exit, just log the error and continue
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

// triggerRainbondClusterReconcile updates all RainbondCluster resources to trigger reconciliation
// This is called on operator startup to ensure monitoring resources are created after upgrade
func triggerRainbondClusterReconcile(mgr manager.Manager) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get the client from manager
	c := mgr.GetClient()

	// List all RainbondCluster resources
	clusterList := &rainbondiov1alpha1.RainbondClusterList{}
	if err := c.List(ctx, clusterList); err != nil {
		return fmt.Errorf("failed to list RainbondClusters: %v", err)
	}

	if len(clusterList.Items) == 0 {
		setupLog.Info("no RainbondCluster found, skipping reconcile trigger")
		return nil
	}

	// Update each cluster to trigger reconcile
	operatorVersion := os.Getenv("OPERATOR_VERSION")
	if operatorVersion == "" {
		operatorVersion = time.Now().Format("20060102-150405")
	}

	for i := range clusterList.Items {
		cluster := &clusterList.Items[i]

		// Check if already updated by this operator version
		if cluster.Annotations != nil {
			if lastVersion, ok := cluster.Annotations["rainbond.io/operator-version"]; ok && lastVersion == operatorVersion {
				setupLog.Info("RainbondCluster already updated by this operator version, skipping",
					"cluster", cluster.Name,
					"namespace", cluster.Namespace,
					"version", operatorVersion)
				continue
			}
		}

		// Update annotation to trigger reconcile
		if cluster.Annotations == nil {
			cluster.Annotations = make(map[string]string)
		}
		cluster.Annotations["rainbond.io/operator-version"] = operatorVersion
		cluster.Annotations["rainbond.io/operator-update-time"] = time.Now().Format(time.RFC3339)

		if err := c.Update(ctx, cluster); err != nil {
			setupLog.Error(err, "failed to update RainbondCluster",
				"cluster", cluster.Name,
				"namespace", cluster.Namespace)
			// Continue to update other clusters
			continue
		}

		setupLog.Info("triggered reconcile for RainbondCluster",
			"cluster", cluster.Name,
			"namespace", cluster.Namespace,
			"operator-version", operatorVersion)
	}

	return nil
}
