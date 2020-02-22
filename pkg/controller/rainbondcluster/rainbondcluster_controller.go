package rainbondcluster

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/goodrain/rainbond-operator/pkg/util/k8sutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"net"
	"net/http"
	"net/url"
	"time"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/goodrain/rainbond-operator/pkg/util/constants"
	rbdutil "github.com/goodrain/rainbond-operator/pkg/util/rbduitl"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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

	if rainbondcluster.Status != nil && len(rainbondcluster.Status.NodeAvailPorts) > 0 && rainbondcluster.Spec.ImageHub != nil {
		return reconcile.Result{}, nil
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

	if rainbondcluster.Spec.ImageHub == nil {
		reqLogger.Info("image hub is empty, do sth.")
		imageHub, err := r.getImageHub(rainbondcluster)
		if err != nil {
			reqLogger.Error(err, "set image hub info")
			return reconcile.Result{RequeueAfter: time.Second * 2}, err
		}
		rainbondcluster.Spec.ImageHub = imageHub
		if err = r.client.Update(ctx, rainbondcluster); err != nil {
			reqLogger.Error(err, "update rainbondcluster")
			return reconcile.Result{RequeueAfter: time.Second * 2}, err
		}
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

func isPortOccupied(address string) bool {
	l, err := net.Listen("tcp", address)
	if err != nil {
		return false
	}
	defer func() { _ = l.Close() }()
	return true
}

// generateRainbondClusterStatus creates the final rainbondcluster status for a rainbondcluster, given the
// internal rainbondcluster status.
func (r *ReconcileRainbondCluster) generateRainbondClusterStatus(ctx context.Context, rainbondCluster *rainbondv1alpha1.RainbondCluster) (*rainbondv1alpha1.RainbondClusterStatus, error) {
	reqLogger := log.WithValues("Namespace", rainbondCluster.Namespace, "Name", rainbondCluster.Name)
	reqLogger.V(6).Info("Generating status")

	masterRoleLabel, err := r.getMasterRoleLabel(ctx)
	if err != nil {
		return nil, fmt.Errorf("get master role label: %v", err)
	}

	s := &rainbondv1alpha1.RainbondClusterStatus{
		MasterRoleLabel: masterRoleLabel,
		StorageClasses:  r.availableStorageClasses(),
	}
	s.GatewayAvailableNodes = rainbondv1alpha1.AvailableNodes{
		SpecifiedNodes: r.listSpecifiedGatewayNodes(ctx),
		MasterNodes:    r.listMasterNodesForGateway(ctx, masterRoleLabel),
	}
	s.ChaosAvailableNodes = rainbondv1alpha1.AvailableNodes{
		SpecifiedNodes: r.listSpecifiedChaosNodes(ctx),
		MasterNodes:    r.listMasterNodes(ctx, masterRoleLabel),
	}

	return s, nil
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

func (r *ReconcileRainbondCluster) getImageHub(cluster *rainbondv1alpha1.RainbondCluster) (*rainbondv1alpha1.ImageHub, error) {
	imageHubReady := func() error {
		httpClient := &http.Client{
			Timeout: 1 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, // TODO: can't ignore TLS
				},
			},
		}

		domain := rbdutil.GetImageRepository(cluster)
		u, err := url.Parse(fmt.Sprintf("https://%s/v2/", cluster.GatewayIngressIP()))
		if err != nil {
			return fmt.Errorf("failed to parse url %s: %v", fmt.Sprintf("https://%s/v2/", domain), err)
		}

		request := &http.Request{URL: u, Host: domain}
		res, err := httpClient.Do(request)
		if err != nil {
			return fmt.Errorf("image repository unavailable: %v", err)
		}
		if res.StatusCode != http.StatusOK {
			return fmt.Errorf("image repository unavailable. http status code: %d", res.StatusCode)
		}
		return nil
	}
	if err := imageHubReady(); err != nil {
		return nil, fmt.Errorf("image repository not ready: %v", err)
	}

	return &rainbondv1alpha1.ImageHub{
		Domain: "goodrain.me",
	}, nil
}

func (r *ReconcileRainbondCluster) listSpecifiedGatewayNodes(ctx context.Context) []*rainbondv1alpha1.K8sNode {
	nodes := r.listNodesByLabels(ctx, map[string]string{
		constants.SpecialGatewayLabelKey: "",
	})
	// Filtering nodes with port conflicts
	// check gateway ports
	return filterNodesWithPortConflicts(nodes)
}

func (r *ReconcileRainbondCluster) listMasterNodesForGateway(ctx context.Context, masterLabel string) []*rainbondv1alpha1.K8sNode {
	nodes := r.listMasterNodes(ctx, masterLabel)
	// Filtering nodes with port conflicts
	// check gateway ports
	return filterNodesWithPortConflicts(nodes)
}

func filterNodesWithPortConflicts(nodes []*rainbondv1alpha1.K8sNode) []*rainbondv1alpha1.K8sNode {
	var result []*rainbondv1alpha1.K8sNode
	gatewayPorts := []int{80, 443, 10254, 18080, 8443, 6060, 7070}
	for idx := range nodes {
		node := nodes[idx]
		ok := true
		for _, port := range gatewayPorts {
			if isPortOccupied(fmt.Sprintf("%s:%d", node.InternalIP, port)) {
				log.Info("The port is occupied", "InternalIP", node.InternalIP, "Port", port)
				ok = false
				break
			}
		}
		if ok {
			result = append(result, node)
		}
	}
	return result
}

func (r *ReconcileRainbondCluster) listSpecifiedChaosNodes(ctx context.Context) []*rainbondv1alpha1.K8sNode {
	return r.listNodesByLabels(ctx, map[string]string{
		constants.SpecialChaosLabelKey: "",
	})
}

func (r *ReconcileRainbondCluster) listMasterNodes(ctx context.Context, masterRoleLabelKey string) []*rainbondv1alpha1.K8sNode {
	labels := k8sutil.MaterRoleLabel(masterRoleLabelKey)
	return r.listNodesByLabels(ctx, labels)
}

func (r *ReconcileRainbondCluster) listNodesByLabels(ctx context.Context, labels map[string]string) []*rainbondv1alpha1.K8sNode {
	nodeList := &corev1.NodeList{}
	listOpts := []client.ListOption{
		client.MatchingLabels(labels),
	}
	if err := r.client.List(ctx, nodeList, listOpts...); err != nil {
		log.Error(err, "list nodes")
		return nil
	}

	findIP := func(addresses []corev1.NodeAddress, addressType corev1.NodeAddressType) string {
		for _, address := range addresses {
			if address.Type == addressType {
				return address.Address
			}
		}
		return ""
	}

	var k8sNodes []*rainbondv1alpha1.K8sNode
	for _, node := range nodeList.Items {
		k8sNode := &rainbondv1alpha1.K8sNode{
			Name:       node.Name,
			InternalIP: findIP(node.Status.Addresses, corev1.NodeInternalIP),
			ExternalIP: findIP(node.Status.Addresses, corev1.NodeExternalIP),
		}
		k8sNodes = append(k8sNodes, k8sNode)
	}

	return k8sNodes
}
