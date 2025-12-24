package controllers

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
	"github.com/goodrain/rainbond-operator/util/constants"
	"github.com/goodrain/rainbond-operator/util/rbdutil"
)

// NodeReconciler watches for node changes and recreates hosts-job when new nodes are added
type NodeReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups="",resources=nodes,verbs=get;list;watch
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;delete

// Reconcile handles node events and recreates hosts-job when nodes change
func (r *NodeReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("node", request.Name)

	// Get the node to verify it exists and is ready
	node := &corev1.Node{}
	err := r.Get(ctx, request.NamespacedName, node)
	if err != nil {
		if errors.IsNotFound(err) {
			// Node was deleted, we don't need to do anything
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// Check if node is ready
	nodeReady := false
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady && condition.Status == corev1.ConditionTrue {
			nodeReady = true
			break
		}
	}

	if !nodeReady {
		reqLogger.V(6).Info("node is not ready yet, skipping hosts-job recreation")
		return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	}

	// Get the rainbond namespace
	namespace := rbdutil.GetenvDefault("RBD_NAMESPACE", constants.Namespace)

	// Check if RainbondCluster exists
	clusterList := &rainbondv1alpha1.RainbondClusterList{}
	if err := r.List(ctx, clusterList, &client.ListOptions{Namespace: namespace}); err != nil {
		reqLogger.Error(err, "failed to list RainbondCluster")
		return reconcile.Result{}, err
	}

	if len(clusterList.Items) == 0 {
		reqLogger.V(6).Info("no RainbondCluster found, skipping hosts-job recreation")
		return reconcile.Result{}, nil
	}

	// Delete the hosts-job to trigger recreation
	job := &batchv1.Job{}
	jobKey := types.NamespacedName{
		Namespace: namespace,
		Name:      "hosts-job",
	}

	err = r.Get(ctx, jobKey, job)
	if err != nil {
		if errors.IsNotFound(err) {
			// Job doesn't exist, nothing to delete
			reqLogger.V(6).Info("hosts-job not found, no need to delete")
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// Delete the job
	reqLogger.Info("deleting hosts-job to trigger recreation for new node")
	if err := r.Delete(ctx, job, client.PropagationPolicy("Background")); err != nil {
		if errors.IsNotFound(err) {
			// Job was already deleted
			return reconcile.Result{}, nil
		}
		reqLogger.Error(err, "failed to delete hosts-job")
		return reconcile.Result{}, err
	}

	reqLogger.Info("successfully deleted hosts-job, it will be recreated with updated node count")
	return reconcile.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager
func (r *NodeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Only react to node creation events
	nodePredicate := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			// React to new nodes
			return true
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Only react to status changes (when node becomes ready)
			oldNode, oldOK := e.ObjectOld.(*corev1.Node)
			newNode, newOK := e.ObjectNew.(*corev1.Node)
			if !oldOK || !newOK {
				return false
			}

			// Check if node just became ready
			oldReady := false
			newReady := false

			for _, condition := range oldNode.Status.Conditions {
				if condition.Type == corev1.NodeReady && condition.Status == corev1.ConditionTrue {
					oldReady = true
					break
				}
			}

			for _, condition := range newNode.Status.Conditions {
				if condition.Type == corev1.NodeReady && condition.Status == corev1.ConditionTrue {
					newReady = true
					break
				}
			}

			// Only trigger if node just became ready
			return !oldReady && newReady
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			// Don't react to node deletion
			return false
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return false
		},
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Node{}).
		WithEventFilter(nodePredicate).
		Complete(r)
}
