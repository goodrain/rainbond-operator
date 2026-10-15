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
	"os"
	"time"

	"github.com/go-logr/logr"
	apisixv2 "github.com/goodrain/rainbond-operator/api/v2"
	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "github.com/go-sql-driver/mysql"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/util/retry"
	kubeaggregatorv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	rainbondiov1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
	"github.com/goodrain/rainbond-operator/controllers"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

const (
	operatorVersionAnnotation    = "rainbond.io/operator-version"
	operatorUpdateTimeAnnotation = "rainbond.io/operator-update-time"
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
	if err = (&controllers.NodeReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("Node"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Node")
		os.Exit(1)
	}
	setupLog.Info("successfully registered NodeReconciler controller")
	// +kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("health", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("check", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	// Add a runnable to trigger reconcile after manager starts
	// This ensures monitoring resources are created after operator upgrade
	if err := mgr.Add(&rainbondClusterReconcileTrigger{
		Client: mgr.GetClient(),
		Log:    setupLog,
	}); err != nil {
		setupLog.Error(err, "unable to add reconcile trigger runnable")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

// rainbondClusterReconcileTrigger triggers all cluster and component
// reconciliations after an operator upgrade.
type rainbondClusterReconcileTrigger struct {
	Client client.Client
	Log    logr.Logger
}

// Start implements manager.Runnable
func (r *rainbondClusterReconcileTrigger) Start(ctx context.Context) error {
	// Wait a bit for the cache to be fully synced
	time.Sleep(2 * time.Second)

	r.Log.Info("triggering reconcile for all RainbondClusters and RbdComponents after operator startup")

	// Create a timeout context
	triggerCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	operatorVersion := os.Getenv("OPERATOR_VERSION")
	if operatorVersion == "" {
		operatorVersion = time.Now().Format("20060102-150405")
	}

	clusterList := &rainbondiov1alpha1.RainbondClusterList{}
	if err := r.Client.List(triggerCtx, clusterList); err != nil {
		r.Log.Error(err, "failed to list RainbondClusters")
	} else {
		for i := range clusterList.Items {
			r.triggerObjectReconcile(triggerCtx, &clusterList.Items[i], operatorVersion, "RainbondCluster")
		}
	}

	componentList := &rainbondiov1alpha1.RbdComponentList{}
	if err := r.Client.List(triggerCtx, componentList); err != nil {
		r.Log.Error(err, "failed to list RbdComponents")
	} else {
		for i := range componentList.Items {
			r.triggerObjectReconcile(triggerCtx, &componentList.Items[i], operatorVersion, "RbdComponent")
		}
	}

	// This runnable completes after triggering, it doesn't need to keep running
	return nil
}

func (r *rainbondClusterReconcileTrigger) triggerObjectReconcile(ctx context.Context, object client.Object, operatorVersion, kind string) {
	key := client.ObjectKeyFromObject(object)
	updated := false
	err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		current := object.DeepCopyObject().(client.Object)
		if err := r.Client.Get(ctx, key, current); err != nil {
			return err
		}
		annotations := current.GetAnnotations()
		if annotations[operatorVersionAnnotation] == operatorVersion {
			return nil
		}
		if annotations == nil {
			annotations = make(map[string]string)
		}
		annotations[operatorVersionAnnotation] = operatorVersion
		annotations[operatorUpdateTimeAnnotation] = time.Now().Format(time.RFC3339)
		current.SetAnnotations(annotations)
		if err := r.Client.Update(ctx, current); err != nil {
			return err
		}
		updated = true
		return nil
	})
	if err != nil {
		r.Log.Error(err, "failed to trigger reconcile", "kind", kind, "name", key.Name, "namespace", key.Namespace)
		return
	}
	if updated {
		r.Log.Info("triggered reconcile", "kind", kind, "name", key.Name, "namespace", key.Namespace, "operator-version", operatorVersion)
	}
}
