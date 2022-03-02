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

package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	wutongv1alpha1 "github.com/wutong-paas/wutong-operator/api/v1alpha1"
	componentmgr "github.com/wutong-paas/wutong-operator/controllers/component-mgr"
	chandler "github.com/wutong-paas/wutong-operator/controllers/handler"
	"github.com/wutong-paas/wutong-operator/util/constants"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WutongComponentReconciler reconciles a WutongComponent object
type WutongComponentReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=wutong.io,resources=WutongComponents,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=wutong.io,resources=WutongComponents/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=wutong.io,resources=WutongComponents/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the WutongComponent object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.0/pkg/reconcile
func (r *WutongComponentReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("WutongComponent", request.NamespacedName)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Fetch the WutongComponent cpt
	cpt := &wutongv1alpha1.WutongComponent{}
	err := r.Get(ctx, request.NamespacedName, cpt)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{Requeue: true}, err
	}

	mgr := componentmgr.NewWutongComponentMgr(ctx, r.Client, r.Recorder, log, cpt)

	fn, ok := handlerFuncs[cpt.Name]
	if !ok {
		reason := "UnsupportedType"
		msg := fmt.Sprintf("only supports the following types of WutongComponent: %s", supportedComponents())

		condition := wutongv1alpha1.NewWutongComponentCondition(wutongv1alpha1.WutongComponentReady, corev1.ConditionFalse, reason, msg)
		changed := cpt.Status.UpdateCondition(condition)
		if changed {
			r.Recorder.Event(cpt, corev1.EventTypeWarning, reason, msg)
			return reconcile.Result{}, mgr.UpdateStatus()
		}
		return reconcile.Result{}, nil
	}

	cluster := &wutongv1alpha1.WutongCluster{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: cpt.Namespace, Name: constants.WutongClusterName}, cluster); err != nil {
		condition := clusterCondition(err)
		changed := cpt.Status.UpdateCondition(condition)
		if changed {
			r.Recorder.Event(cpt, corev1.EventTypeWarning, condition.Reason, condition.Message)
			return reconcile.Result{RequeueAfter: 3 * time.Second}, mgr.UpdateStatus()
		}
		return reconcile.Result{RequeueAfter: 3 * time.Second}, nil
	}

	if !cluster.Spec.ConfigCompleted {
		log.V(6).Info("WutongCluster configuration is not complete")
		condition := wutongv1alpha1.NewWutongComponentCondition(wutongv1alpha1.ClusterConfigCompeleted,
			corev1.ConditionFalse, "ConfigNotCompleted", "WutongCluster configuration is not complete")
		changed := cpt.Status.UpdateCondition(condition)
		if changed {
			r.Recorder.Event(cpt, corev1.EventTypeWarning, condition.Reason, condition.Message)
			return reconcile.Result{RequeueAfter: 3 * time.Second}, mgr.UpdateStatus()
		}
		return reconcile.Result{RequeueAfter: 3 * time.Second}, err
	}
	mgr.SetConfigCompletedCondition()

	var pkg *wutongv1alpha1.WutongPackage
	if cluster.Spec.InstallMode != wutongv1alpha1.InstallationModeFullOnline {
		pkg = &wutongv1alpha1.WutongPackage{}
		if err := r.Get(ctx, types.NamespacedName{Namespace: cpt.Namespace, Name: constants.WutongPackageName}, pkg); err != nil {
			condition := packageCondition(err)
			changed := cpt.Status.UpdateCondition(condition)
			if changed {
				r.Recorder.Event(cpt, corev1.EventTypeWarning, condition.Reason, condition.Message)
				return reconcile.Result{RequeueAfter: 3 * time.Second}, mgr.UpdateStatus()
			}

			return reconcile.Result{RequeueAfter: 3 * time.Second}, nil
		}
	}
	mgr.SetPackageReadyCondition(pkg)

	if !mgr.CheckPrerequisites(cluster, pkg) {
		condition := wutongv1alpha1.NewWutongComponentCondition(wutongv1alpha1.WutongComponentReady, corev1.ConditionFalse,
			"PrerequisitesFailed", "failed to check prerequisites")
		changed := cpt.Status.UpdateCondition(condition)
		if changed {
			r.Recorder.Event(cpt, corev1.EventTypeWarning, condition.Reason, condition.Message)
			return reconcile.Result{RequeueAfter: 3 * time.Second}, mgr.UpdateStatus()
		}
		return reconcile.Result{RequeueAfter: 3 * time.Second}, nil
	}

	hdl := fn(ctx, r.Client, cpt, cluster)
	if err := hdl.Before(); err != nil {
		// TODO: merge with mgr.checkPrerequisites
		if chandler.IsIgnoreError(err) {
			log.V(7).Info("checking the prerequisites", "msg", err.Error())
		} else {
			log.V(6).Info("checking the prerequisites", "msg", err.Error())
		}

		condition := wutongv1alpha1.NewWutongComponentCondition(wutongv1alpha1.WutongComponentReady, corev1.ConditionFalse, "PrerequisitesFailed", err.Error())
		changed := cpt.Status.UpdateCondition(condition)
		if changed {
			r.Recorder.Event(cpt, corev1.EventTypeWarning, condition.Reason, condition.Message)
			return reconcile.Result{RequeueAfter: 3 * time.Second}, mgr.UpdateStatus()
		}
		return reconcile.Result{RequeueAfter: 3 * time.Second}, nil
	}

	resourcesDeleter, ok := hdl.(chandler.ResourcesDeleter)
	if ok {
		result, err := mgr.DeleteResources(resourcesDeleter)
		if err != nil {
			return reconcile.Result{}, err
		}
		if result != nil {
			return *result, nil
		}
	}

	resourceCreator, ok := hdl.(chandler.ResourcesCreator)
	if ok {
		log.V(6).Info("ResourcesCreator create resources create if not exists")
		resourcesCreateIfNotExists := resourceCreator.ResourcesCreateIfNotExists()
		for _, res := range resourcesCreateIfNotExists {
			if res == nil {
				continue
			}
			// Set WutongComponent cpt as the owner and controller
			if err := controllerutil.SetControllerReference(cpt, res.(metav1.Object), r.Scheme); err != nil {
				log.Error(err, "set controller reference")
				condition := wutongv1alpha1.NewWutongComponentCondition(wutongv1alpha1.WutongComponentReady,
					corev1.ConditionFalse, "SetControllerReferenceFailed", err.Error())
				changed := cpt.Status.UpdateCondition(condition)
				if changed {
					r.Recorder.Event(cpt, corev1.EventTypeWarning, condition.Reason, condition.Message)
					return reconcile.Result{Requeue: true}, mgr.UpdateStatus()
				}
				return reconcile.Result{}, err
			}
			if err := mgr.ResourceCreateIfNotExists(res); err != nil {
				log.Error(err, "create resouce if not exists")
				condition := wutongv1alpha1.NewWutongComponentCondition(wutongv1alpha1.WutongComponentReady,
					corev1.ConditionFalse, "ErrCreateResources", err.Error())
				changed := cpt.Status.UpdateCondition(condition)
				if changed {
					r.Recorder.Event(cpt, corev1.EventTypeWarning, condition.Reason, condition.Message)
					return reconcile.Result{Requeue: true}, mgr.UpdateStatus()
				}
				return reconcile.Result{}, err
			}
		}
	}

	resourceCreatorClusterScope, ok := hdl.(chandler.ClusterScopedResourcesCreator)
	if ok {
		log.V(6).Info("ClusterScopedResourcesCreator create resources create if not exists")
		resourcesCreateIfNotExists := resourceCreatorClusterScope.CreateClusterScoped()
		for _, res := range resourcesCreateIfNotExists {
			if res == nil {
				continue
			}

			if err := mgr.ResourceCreateIfNotExists(res); err != nil {
				log.Error(err, "create resouce if not exists")
				condition := wutongv1alpha1.NewWutongComponentCondition(wutongv1alpha1.WutongComponentReady,
					corev1.ConditionFalse, "ErrCreateResources", err.Error())
				changed := cpt.Status.UpdateCondition(condition)
				if changed {
					r.Recorder.Event(cpt, corev1.EventTypeWarning, condition.Reason, condition.Message)
					return reconcile.Result{Requeue: true}, mgr.UpdateStatus()
				}
				return reconcile.Result{}, err
			}
		}
	}

	replicaser, ok := hdl.(chandler.Replicaser)
	if ok {
		mgr.SetReplicaser(replicaser)
	}

	resources := hdl.Resources()
	for _, res := range resources {
		if res == nil {
			continue
		}
		// Set WutongComponent cpt as the owner and controller
		if err := controllerutil.SetControllerReference(cpt, res.(metav1.Object), r.Scheme); err != nil {
			log.Error(err, "set controller reference")
			condition := wutongv1alpha1.NewWutongComponentCondition(wutongv1alpha1.WutongComponentReady, corev1.ConditionFalse,
				"SetControllerReferenceFailed", err.Error())
			changed := cpt.Status.UpdateCondition(condition)
			if changed {
				r.Recorder.Event(cpt, corev1.EventTypeWarning, condition.Reason, condition.Message)
				return reconcile.Result{Requeue: true}, mgr.UpdateStatus()
			}
			return reconcile.Result{}, err
		}
		// Check if the resource already exists, if not create a new one
		reconcileResult, err := mgr.UpdateOrCreateResource(res)
		if err != nil {
			log.Error(err, "update or create resource")
			condition := wutongv1alpha1.NewWutongComponentCondition(wutongv1alpha1.WutongComponentReady, corev1.ConditionFalse, "ErrCreateResources", err.Error())
			changed := cpt.Status.UpdateCondition(condition)
			if changed {
				r.Recorder.Event(cpt, corev1.EventTypeWarning, condition.Reason, condition.Message)
				return reconcile.Result{Requeue: true}, mgr.UpdateStatus()
			}
			return reconcileResult, err
		}
	}

	if err := hdl.After(); err != nil {
		log.Error(err, "failed to execute after process")
		condition := wutongv1alpha1.NewWutongComponentCondition(wutongv1alpha1.WutongComponentReady, corev1.ConditionFalse,
			"ErrAfterProcess", err.Error())
		changed := cpt.Status.UpdateCondition(condition)
		if changed {
			r.Recorder.Event(cpt, corev1.EventTypeWarning, condition.Reason, condition.Message)
			return reconcile.Result{Requeue: true}, mgr.UpdateStatus()
		}
		return reconcile.Result{Requeue: true}, err
	}

	pods, err := hdl.ListPods()
	if err != nil {
		condition := wutongv1alpha1.NewWutongComponentCondition(wutongv1alpha1.WutongComponentReady, corev1.ConditionFalse,
			"ErrListPods", err.Error())
		changed := cpt.Status.UpdateCondition(condition)
		if changed {
			r.Recorder.Event(cpt, corev1.EventTypeWarning, condition.Reason, condition.Message)
			return reconcile.Result{Requeue: true}, mgr.UpdateStatus()
		}
		return reconcile.Result{Requeue: true}, err
	}

	mgr.GenerateStatus(pods)

	if err := mgr.UpdateStatus(); err != nil {
		log.Error(err, "update wutong component status failure %s")
	}

	if !mgr.IsWutongComponentReady() {
		return reconcile.Result{RequeueAfter: 5 * time.Second}, nil
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *WutongComponentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&wutongv1alpha1.WutongComponent{}).
		Complete(r)
}

func clusterCondition(err error) *wutongv1alpha1.WutongComponentCondition {
	reason := "ClusterNotFound"
	msg := "WutongCluster not found"
	if !k8sErrors.IsNotFound(err) {
		reason = "UnknownErr"
		msg = fmt.Sprintf("failed to get WutongCluster: %v", err)
	}

	return wutongv1alpha1.NewWutongComponentCondition(wutongv1alpha1.ClusterConfigCompeleted, corev1.ConditionFalse, reason, msg)
}

func packageCondition(err error) *wutongv1alpha1.WutongComponentCondition {
	reason := "PackageNotFound"
	msg := "WutongPackage not found"
	if !k8sErrors.IsNotFound(err) {
		reason = "UnknownErr"
		msg = fmt.Sprintf("failed to get WutongPackage: %v", err)
	}
	return wutongv1alpha1.NewWutongComponentCondition(wutongv1alpha1.WutongPackageReady, corev1.ConditionFalse, reason, msg)
}
