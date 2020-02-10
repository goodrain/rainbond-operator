package rainbondcluster

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/goodrain/rainbond-operator/pkg/util/commonutil"
	"github.com/goodrain/rainbond-operator/pkg/util/constants"
	"github.com/goodrain/rainbond-operator/pkg/util/k8sutil"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/goodrain/rainbond-operator/pkg/util/format"
	rbdutil "github.com/goodrain/rainbond-operator/pkg/util/rbduitl"

	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	// Only watch rainbondcluster, because only support one rainbond cluster.
	err = c.Watch(&source.Kind{Type: &rainbondv1alpha1.RainbondCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rainbondcluster",
			Namespace: "rbd-system",
		},
	}}, &handler.EnqueueRequestForObject{})
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Fetch the RainbondCluster instance
	rainbondcluster := &rainbondv1alpha1.RainbondCluster{}
	err := r.client.Get(ctx, request.NamespacedName, rainbondcluster)
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

	if rainbondcluster.Status != nil && len(rainbondcluster.Status.NodeAvailPorts) > 0 {
		return reconcile.Result{}, nil
	}

	claims := r.claims(rainbondcluster)
	for i := range claims {
		claim := claims[i]
		// Set RbdComponent cpt as the owner and controller
		if err := controllerutil.SetControllerReference(rainbondcluster, claim, r.scheme); err != nil {
			reqLogger.Error(err, "set controller reference")
			return reconcile.Result{RequeueAfter: time.Second * 2}, err
		}
		if err = k8sutil.UpdateOrCreateResource(ctx, r.client, reqLogger, claim, claim); err != nil {
			reqLogger.Error(err, "update or create pvc")
			return reconcile.Result{RequeueAfter: time.Second * 2}, err
		}
	}

	status, err := r.generateRainbondClusterStatus(ctx, rainbondcluster)
	if err != nil {
		reqLogger.Error(err, "failed to generate rainbondcluster status")
		return reconcile.Result{RequeueAfter: time.Second * 2}, err
	}
	rainbondcluster.Status = status
	if err := r.client.Status().Update(ctx, rainbondcluster); err != nil {
		reqLogger.Error(err, "failed to update rainbondcluster status")
		return reconcile.Result{RequeueAfter: time.Second * 2}, err
	}

	return reconcile.Result{Requeue: false}, nil
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

func (r *ReconcileRainbondCluster) listMasterNodeNames(masterRoleLabel map[string]string) []string {
	klog.V(3).Info("Start listing node names")
	nodeList := &corev1.NodeList{}
	listOpts := []client.ListOption{
		client.MatchingLabels(masterRoleLabel),
	}
	if err := r.client.List(context.TODO(), nodeList, listOpts...); err != nil {
		klog.Error(err, "list nodes")
		return nil
	}

	var nodeNames []string
	for _, node := range nodeList.Items {
		nodeNames = append(nodeNames, node.Name)
	}
	return nodeNames
}

func (r *ReconcileRainbondCluster) listNodeAvailablePorts(masterRoleLabel map[string]string) []*rainbondv1alpha1.NodeAvailPorts {
	klog.V(3).Info("Start checking rbd-gateway ports")
	// list all node
	nodeList := &corev1.NodeList{}
	listOpts := []client.ListOption{
		client.MatchingLabels(masterRoleLabel),
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
			klog.V(5).Info("check port occupation", "error", err.Error())
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
func (r *ReconcileRainbondCluster) generateRainbondClusterStatus(ctx context.Context, rainbondCluster *rainbondv1alpha1.RainbondCluster) (*rainbondv1alpha1.RainbondClusterStatus, error) {
	klog.Infof("Generating status for %q", format.RainbondCluster(rainbondCluster))

	masterRoleLabel, err := r.getMasterRoleLabel(ctx)
	if err != nil {
		return nil, fmt.Errorf("get master role label: %v", err)
	}
	klog.Infof("master role label: %s", masterRoleLabel)

	s := &rainbondv1alpha1.RainbondClusterStatus{
		MasterRoleLabel: masterRoleLabel,
		StorageClasses:  r.availableStorageClasses(),
	}
	s.NodeAvailPorts = r.listNodeAvailablePorts(s.MasterNodeLabel())
	s.MasterNodeNames = r.listMasterNodeNames(s.MasterNodeLabel())

	return s, nil
}

func (r *ReconcileRainbondCluster) claims(cluster *rainbondv1alpha1.RainbondCluster) []*corev1.PersistentVolumeClaim {
	storageRequest := resource.NewQuantity(10, resource.BinarySI) // TODO: size

	grdata := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: constants.Namespace,
			Name:      constants.GrDataPVC,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteMany,
			},
			Resources: corev1.ResourceRequirements{
				Requests: map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceStorage: *storageRequest,
				},
			},
			StorageClassName: commonutil.String(rbdutil.GetStorageClass(cluster)),
		},
	}

	cache := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: constants.Namespace,
			Name:      "cache",
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteMany,
			},
			Resources: corev1.ResourceRequirements{
				Requests: map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceStorage: *storageRequest,
				},
			},
			StorageClassName: commonutil.String(rbdutil.GetStorageClass(cluster)),
		},
	}

	return []*corev1.PersistentVolumeClaim{grdata, cache}
}

func (r *ReconcileRainbondCluster) getMasterRoleLabel(ctx context.Context) (string, error) {
	nodes := &corev1.NodeList{}
	if err := r.client.List(ctx, nodes); err != nil {
		log.Error(err, "list nodes: %v", err)
		return "", nil
	}
	var label string
	for _, node := range nodes.Items {
		for key := range node.Labels {
			if key == rainbondv1alpha1.LabelNodeRolePrefix+"master" {
				label = key
			}
			if key == rainbondv1alpha1.NodeLabelRole && label != rainbondv1alpha1.LabelNodeRolePrefix+"master" {
				label = key
			}
		}
	}
	return label, nil
}
