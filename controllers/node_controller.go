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
	reqLogger.Info("NodeReconciler triggered for node event")

	// Get the node to verify it exists and is ready
	node := &corev1.Node{}
	err := r.Get(ctx, request.NamespacedName, node)
	if err != nil {
		if errors.IsNotFound(err) {
			// Node was deleted, we don't need to do anything
			reqLogger.Info("node not found, it may have been deleted")
			return reconcile.Result{}, nil
		}
		reqLogger.Error(err, "failed to get node")
		return reconcile.Result{}, err
	}

	reqLogger.Info("successfully retrieved node information")

	// Check if node is ready
	nodeReady := false
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady && condition.Status == corev1.ConditionTrue {
			nodeReady = true
			break
		}
	}

	if !nodeReady {
		reqLogger.Info("node is not ready yet, skipping hosts-job recreation")
		return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
	}

	reqLogger.Info("node is ready, proceeding with hosts-job recreation")

	// Get the rainbond namespace
	namespace := rbdutil.GetenvDefault("RBD_NAMESPACE", constants.Namespace)
	reqLogger.Info("checking RainbondCluster existence", "namespace", namespace)

	// Check if RainbondCluster exists
	clusterList := &rainbondv1alpha1.RainbondClusterList{}
	if err := r.List(ctx, clusterList, &client.ListOptions{Namespace: namespace}); err != nil {
		reqLogger.Error(err, "failed to list RainbondCluster")
		return reconcile.Result{}, err
	}

	if len(clusterList.Items) == 0 {
		reqLogger.Info("no RainbondCluster found, skipping hosts-job recreation", "namespace", namespace)
		return reconcile.Result{}, nil
	}

	reqLogger.Info("found RainbondCluster, proceeding to check hosts-job", "clusterCount", len(clusterList.Items))

	// Delete the hosts-job to trigger recreation
	job := &batchv1.Job{}
	jobKey := types.NamespacedName{
		Namespace: namespace,
		Name:      "hosts-job",
	}

	reqLogger.Info("looking for hosts-job", "namespace", namespace, "name", "hosts-job")

	err = r.Get(ctx, jobKey, job)
	if err != nil {
		if errors.IsNotFound(err) {
			// Job doesn't exist, nothing to delete
			reqLogger.Info("hosts-job not found, no need to delete - it may not have been created yet")
			return reconcile.Result{}, nil
		}
		reqLogger.Error(err, "error getting hosts-job")
		return reconcile.Result{}, err
	}

	// Delete the job
	reqLogger.Info("found hosts-job, deleting it to trigger recreation for new node", "jobName", job.Name)
	if err := r.Delete(ctx, job, client.PropagationPolicy("Background")); err != nil {
		if errors.IsNotFound(err) {
			// Job was already deleted
			reqLogger.Info("hosts-job was already deleted")
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
	log := r.Log.WithName("SetupWithManager")

	// Only react to node creation events
	nodePredicate := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			// React to new nodes
			node, ok := e.Object.(*corev1.Node)
			if ok {
				log.Info("NodeReconciler Predicate: Node created", "node", node.Name)
			}
			return true
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Only react to status changes (when node becomes ready)
			oldNode, oldOK := e.ObjectOld.(*corev1.Node)
			newNode, newOK := e.ObjectNew.(*corev1.Node)
			if !oldOK || !newOK {
				log.Info("NodeReconciler Predicate: Update event but not a Node object")
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
			shouldTrigger := !oldReady && newReady
			if shouldTrigger {
				log.Info("NodeReconciler Predicate: Node became ready", "node", newNode.Name)
			} else {
				log.V(6).Info("NodeReconciler Predicate: Node updated but not ready transition", "node", newNode.Name, "oldReady", oldReady, "newReady", newReady)
			}
			return shouldTrigger
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			// Don't react to node deletion
			node, ok := e.Object.(*corev1.Node)
			if ok {
				log.Info("NodeReconciler Predicate: Node deleted (ignoring)", "node", node.Name)
			}
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
