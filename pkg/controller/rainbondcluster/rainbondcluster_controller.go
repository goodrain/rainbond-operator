package rainbondcluster

import (
	"context"
	"fmt"
	rainbondv1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/GLYASAI/rainbond-operator/pkg/controller/rainbondcluster/status"
	"github.com/GLYASAI/rainbond-operator/pkg/util/format"
	"net"
	"time"

	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_rainbondcluster")

// Add creates a new RainbondCluster Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileRainbondCluster{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("rainbondcluster-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource RainbondCluster
	err = c.Watch(&source.Kind{Type: &rainbondv1alpha1.RainbondCluster{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(haungrh): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner RainbondCluster
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &rainbondv1alpha1.RainbondCluster{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileRainbondCluster implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileRainbondCluster{}

// ReconcileRainbondCluster reconciles a RainbondCluster object
type ReconcileRainbondCluster struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a RainbondCluster object and makes changes based on the state read
// and what is in the RainbondCluster.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileRainbondCluster) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling RainbondCluster")

	// Fetch the RainbondCluster instance
	rainbondcluster := &rainbondv1alpha1.RainbondCluster{}
	err := r.client.Get(context.TODO(), request.NamespacedName, rainbondcluster)
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

	rainbondcluster.Status = r.generateRainbondClusterStatus(rainbondcluster)
	if err := r.client.Status().Update(context.TODO(), rainbondcluster); err != nil {
		klog.Error("Error updating rainbondcluster status: %v", err)
		return reconcile.Result{Requeue: true}, err
	}

	return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
}

func (r *ReconcileRainbondCluster) availableStorageClasses() []*rainbondv1alpha1.StorageClass {
	klog.V(3).Info("Start listing available storage classes")

	storageClassList := &storagev1.StorageClassList{}
	var opts []client.ListOption
	if err := r.client.List(context.TODO(), storageClassList, opts...); err != nil {
		klog.V(2).Info("Start listing available storage classes")
		return nil
	}

	var storageClasses []*rainbondv1alpha1.StorageClass
	for _, sc := range storageClassList.Items {
		storageClass := &rainbondv1alpha1.StorageClass{
			Name:        sc.Name,
			Provisioner: sc.Provisioner,
		}
		storageClasses = append(storageClasses, storageClass)
	}

	return storageClasses
}

func (r *ReconcileRainbondCluster) listNodeAvailablePorts() []*rainbondv1alpha1.NodeAvailPorts {
	klog.V(3).Info("Start checking rbd-gateway ports")
	// list all node
	nodeList := &corev1.NodeList{}
	listOpts := []client.ListOption{
		client.MatchingLabels(map[string]string{
			"node-role.kubernetes.io/master": "", // TODO: This label does not necessarily exist. At this time, the user needs to specify
		}),
	}
	if err := r.client.List(context.TODO(), nodeList, listOpts...); err != nil {
		klog.Error(err, "list nodes")
		return nil
	}
	klog.V(3).Info("Found nodes", nodeList)

	checkPortOccupation := func(address string) bool {
		klog.V(3).Info("Start check port occupation", "Address: ", address)
		conn, err := net.Dial("tcp", address)
		if err != nil {
			klog.Error("check port occupation", "error", err.Error())
			return false
		}
		defer conn.Close()
		return true
	}

	gatewayPorts := []int{80, 443, 10254, 18080} // TODO: do not hard code
	var nodeAvailPorts []*rainbondv1alpha1.NodeAvailPorts
	for _, n := range nodeList.Items {
		for _, addr := range n.Status.Addresses {
			if addr.Type != corev1.NodeInternalIP {
				continue
			}
			klog.V(3).Info("Node name", n.Name, "Found internal ip: ", addr.Address)
			node := &rainbondv1alpha1.NodeAvailPorts{
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

// generateRainbondClusterStatus creates the final rainbondcluster status for a rainbondcluster, given the
// internal rainbondcluster status.
func (r *ReconcileRainbondCluster) generateRainbondClusterStatus(rainbondCluster *rainbondv1alpha1.RainbondCluster) *rainbondv1alpha1.RainbondClusterStatus {
	klog.V(3).Infof("Generating status for %q", format.RainbondCluster(rainbondCluster))

	s := &rainbondv1alpha1.RainbondClusterStatus{
		StorageClasses: r.availableStorageClasses(),
		NodeAvailPorts: r.listNodeAvailablePorts(),
	}

	s.Conditions = append(s.Conditions, status.GenerateRainbondClusterStorageReadyCondition())
	s.Conditions = append(s.Conditions, status.GenerateRainbondClusterImageRepositoryReadyCondition(rainbondCluster))

	return s
}
