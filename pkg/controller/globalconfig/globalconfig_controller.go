package globalconfig

import (
	"context"
	"fmt"
	"net"
	"reflect"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/go-logr/logr"

	rainbondv1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"
)

var log = logf.Log.WithName("controller_globalconfig")

// Add creates a new GlobalConfig Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileGlobalConfig{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("globalconfig-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource GlobalConfig
	err = c.Watch(&source.Kind{Type: &rainbondv1alpha1.GlobalConfig{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileGlobalConfig implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileGlobalConfig{}

// ReconcileGlobalConfig reconciles a GlobalConfig object
type ReconcileGlobalConfig struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a GlobalConfig object and makes changes based on the state read
// and what is in the GlobalConfig.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileGlobalConfig) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling GlobalConfig")

	// Fetch the GlobalConfig instance
	globalConfig := &rainbondv1alpha1.GlobalConfig{}
	err := r.client.Get(context.TODO(), request.NamespacedName, globalConfig)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	nodeAvailPorts := r.listNodeAavailPorts(reqLogger)

	newGlobalConfigSpec := globalConfig.Spec.DeepCopy()
	newGlobalConfigSpec.NodeAvailPorts = nodeAvailPorts
	if !reflect.DeepEqual(globalConfig.Spec, newGlobalConfigSpec) {
		globalConfig.Spec = *newGlobalConfigSpec
		if err := r.updateGlobalConfig(reqLogger, globalConfig); err != nil {
			// Error updating the object - requeue the request.
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileGlobalConfig) listNodeAavailPorts(reqLogger logr.Logger) []rainbondv1alpha1.NodeAvailPorts {
	reqLogger.V(4).Info("Start checking rbd-gateway ports")
	// list all node
	nodeList := &corev1.NodeList{}
	listOpts := []client.ListOption{
		client.MatchingLabels(map[string]string{
			"node-role.kubernetes.io/master": "", // TODO: This label does not necessarily exist. At this time, the user needs to specify
		}),
	}
	if err := r.client.List(context.TODO(), nodeList, listOpts...); err != nil {
		reqLogger.V(2).Info("failed to list nodes", err.Error())
		return nil
	}

	gatewayPorts := []int{80, 443, 10254, 18080}
	var nodeAvailPorts []rainbondv1alpha1.NodeAvailPorts
	for _, n := range nodeList.Items {
		for _, addr := range n.Status.Addresses {
			if addr.Type != corev1.NodeExternalIP {
				continue
			}
			reqLogger.V(4).Info("Node name", n.Name, "Found external ip: ", addr.Address)
			node := rainbondv1alpha1.NodeAvailPorts{
				NodeName: n.Name,
				NodeIP:   addr.Address,
			}

			// check gateway ports
			for _, gwport := range gatewayPorts {
				if !checkPortOccupation(fmt.Sprintf("%s:%d", node.NodeIP, gwport)) {
					node.Ports = append(node.Ports, gwport)
				}
			}

			nodeAvailPorts = append(nodeAvailPorts, node)
			break
		}
	}

	return nodeAvailPorts
}

func checkPortOccupation(address string) bool {
	l, err := net.Listen("tcp", address)
	if err != nil {
		return true
	}
	defer l.Close()
	return false
}

func (r *ReconcileGlobalConfig) updateGlobalConfig(reqLogger logr.Logger, globalConfig *rainbondv1alpha1.GlobalConfig) error {
	reqLogger.V(4).Info("Start updating globalconfig.")
	if err := r.client.Update(context.TODO(), globalConfig); err != nil {
		reqLogger.V(0).Info("Update globalconfig", err.Error())
		return err
	}
	return nil
}
