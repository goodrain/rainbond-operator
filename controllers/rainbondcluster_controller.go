package controllers

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
	clustermgr "github.com/goodrain/rainbond-operator/controllers/cluster-mgr"
	"github.com/goodrain/rainbond-operator/util/commonutil"
	"github.com/goodrain/rainbond-operator/util/constants"
	"github.com/goodrain/rainbond-operator/util/rbdutil"
	"github.com/goodrain/rainbond-operator/util/uuidutil"
	"github.com/juju/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/retry"
	"net"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"
)

// RainbondClusterReconciler reconciles a RainbondCluster object
type RainbondClusterReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=rainbond.io,resources=rainbondclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rainbond.io,resources=rainbondclusters/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=rainbond.io,resources=rainbondclusters/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the RainbondCluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.0/pkg/reconcile
func (r *RainbondClusterReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	reqLogger := r.Log.WithValues("rainbondcluster", request.NamespacedName)

	// Fetch the RainbondCluster instance
	rainbondcluster := &rainbondv1alpha1.RainbondCluster{}
	err := r.Get(ctx, request.NamespacedName, rainbondcluster)
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

	mgr := clustermgr.NewClusterMgr(ctx, r.Client, reqLogger, rainbondcluster, r.Scheme)

	// generate status for rainbond cluster
	reqLogger.V(6).Info("start generate status")
	status, err := mgr.GenerateRainbondClusterStatus()
	if err != nil {
		reqLogger.Error(err, "failed to generate rainbondcluster status")
		return reconcile.Result{RequeueAfter: time.Second * 2}, err
	}

	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		rc := &rainbondv1alpha1.RainbondCluster{}
		if err := r.Get(ctx, request.NamespacedName, rc); err != nil {
			return err
		}
		rc.Status = *status
		return r.Status().Update(ctx, rc)
	}); err != nil {
		reqLogger.Error(err, "update rainbondcluster status")
		return reconcile.Result{RequeueAfter: time.Second * 2}, err
	}
	reqLogger.V(6).Info("update status success")

	// handle enterprise ID
	if rainbondcluster.Annotations != nil {
		if _, ok := rainbondcluster.Annotations["meta.helm.sh/release-name"]; ok {
			if _, ok := rainbondcluster.Annotations["enterprise_id"]; !ok {
				if os.Getenv("ENTERPRISE_ID") == "" {
					rainbondcluster.Annotations["enterprise_id"] = uuidutil.NewUUID()
					os.Setenv("ENTERPRISE_ID", rainbondcluster.Annotations["enterprise_id"])
					if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
						rc := &rainbondv1alpha1.RainbondCluster{}
						if err := r.Get(ctx, request.NamespacedName, rc); err != nil {
							return err
						}
						rc.Annotations = rainbondcluster.Annotations
						return r.Update(ctx, rc)
					}); err != nil {
						reqLogger.Error(err, "update rainbondcluster status")
						return reconcile.Result{RequeueAfter: time.Second * 2}, err
					}
				}
			}
			os.Setenv("INSTALL_VERSION", rainbondcluster.Spec.InstallVersion)
		}
	}

	// setup nodesForGateway nodesForChaos gatewayIngressIP if empty
	if rainbondcluster.Spec.NodesForGateway == nil || rainbondcluster.Spec.NodesForChaos == nil || rainbondcluster.Spec.GatewayIngressIPs == nil {
		gatewayNodes, chaosNodes := r.GetRainbondGatewayNodeAndChaosNodes()
		if gatewayNodes == nil || chaosNodes == nil {
			return reconcile.Result{RequeueAfter: time.Second * 3}, err
		}
		if rainbondcluster.Spec.NodesForGateway == nil {
			rainbondcluster.Spec.NodesForGateway = gatewayNodes
		}
		if rainbondcluster.Spec.NodesForChaos == nil {
			rainbondcluster.Spec.NodesForChaos = chaosNodes
		}
		if rainbondcluster.Spec.GatewayIngressIPs == nil {
			rainbondcluster.Spec.GatewayIngressIPs = func() (re []string) {
				for _, n := range rainbondcluster.Spec.NodesForGateway {
					if n.ExternalIP != "" {
						re = append(re, n.ExternalIP)
					}
				}
				if len(re) == 0 {
					for _, n := range rainbondcluster.Spec.NodesForGateway {
						if n.InternalIP != "" {
							re = append(re, n.InternalIP)
						}
					}
				}
				return
			}()
		}
		if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			rc := &rainbondv1alpha1.RainbondCluster{}
			if err := r.Get(ctx, request.NamespacedName, rc); err != nil {
				return err
			}
			rc.Spec.GatewayIngressIPs = rainbondcluster.Spec.GatewayIngressIPs
			rc.Spec.NodesForGateway = rainbondcluster.Spec.NodesForGateway
			rc.Spec.NodesForChaos = rainbondcluster.Spec.NodesForChaos
			return r.Update(ctx, rc)
		}); err != nil {
			reqLogger.Error(err, "update rainbondcluster")
			return reconcile.Result{RequeueAfter: time.Second * 1}, err
		}
		return reconcile.Result{Requeue: true}, err
	}
	// set gatewayIngressIP to local host file
	hostPath := "/etc/hosts"
	NodesForGateway := rainbondcluster.Spec.NodesForGateway[0].InternalIP
	commonutil.WriteHosts(hostPath, NodesForGateway)
	// setup imageHub if empty
	if rainbondcluster.Spec.ImageHub == nil {
		reqLogger.V(6).Info("create new image hub info")
		imageHub, err := r.getImageHub(rainbondcluster)
		if err != nil {
			reqLogger.V(6).Info(fmt.Sprintf("set image hub info: %v", err))
			return reconcile.Result{RequeueAfter: time.Second * 1}, nil
		}
		if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			rc := &rainbondv1alpha1.RainbondCluster{}
			if err := r.Get(ctx, request.NamespacedName, rc); err != nil {
				return err
			}
			rc.Spec.ImageHub = imageHub
			rainbondcluster = rc
			return r.Update(ctx, rc)
		}); err != nil {
			reqLogger.Error(err, "update rainbondcluster")
			return reconcile.Result{RequeueAfter: time.Second * 1}, err
		}
		reqLogger.V(6).Info("create new image hub info success")
		// Put it back in the queue.
		return reconcile.Result{Requeue: true}, err
	}

	if rainbondcluster.Spec.SuffixHTTPHost == "" {
		var ip string
		if len(rainbondcluster.Spec.NodesForGateway) > 0 {
			ip = rainbondcluster.Spec.NodesForGateway[0].InternalIP
		}
		if len(rainbondcluster.Spec.GatewayIngressIPs) > 0 && rainbondcluster.Spec.GatewayIngressIPs[0] != "" {
			ip = rainbondcluster.Spec.GatewayIngressIPs[0]
		}
		if ip != "" {
			rainbondcluster.Spec.SuffixHTTPHost = ip + rbdutil.GetenvDefault("DNS_SERVER", ".nip.io")
			rc := &rainbondv1alpha1.RainbondCluster{}
			if err := r.Get(ctx, request.NamespacedName, rc); err != nil {
				return reconcile.Result{RequeueAfter: time.Second * 1}, err
			}
			rc.Spec.SuffixHTTPHost = rainbondcluster.Spec.SuffixHTTPHost
			return reconcile.Result{}, r.Update(ctx, rc)
		}
		rc := &rainbondv1alpha1.RainbondCluster{}
		if err := r.Get(ctx, request.NamespacedName, rc); err != nil {
			return reconcile.Result{RequeueAfter: time.Second * 1}, err
		}
		rc.Spec.SuffixHTTPHost = constants.DefHTTPDomainSuffix
		return reconcile.Result{}, r.Update(ctx, rc)
	}

	// create secret for pulling images.
	if rainbondcluster.Spec.ImageHub != nil && rainbondcluster.Spec.ImageHub.Username != "" && rainbondcluster.Spec.ImageHub.Password != "" {
		err := mgr.CreateImagePullSecret()
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	for _, con := range rainbondcluster.Status.Conditions {
		if con.Status != corev1.ConditionTrue {
			return reconcile.Result{RequeueAfter: 5 * time.Second}, nil
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RainbondClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&rainbondv1alpha1.RainbondCluster{}).
		Complete(r)
}

func (r *RainbondClusterReconciler) getImageHub(cluster *rainbondv1alpha1.RainbondCluster) (*rainbondv1alpha1.ImageHub, error) {
	return &rainbondv1alpha1.ImageHub{
		Domain:   constants.DefImageRepository,
		Username: "admin",
		Password: rbdutil.GetenvDefault("RBD_HUB_PASSWORD", uuidutil.NewUUID()[0:8]),
	}, nil
}

// GetRainbondGatewayNodeAndChaosNodes get gateway nodes
func (r *RainbondClusterReconciler) GetRainbondGatewayNodeAndChaosNodes() (gatewayNodes, chaosNodes []*rainbondv1alpha1.K8sNode) {
	nodeList := &corev1.NodeList{}
	reqLogger := r.Log.WithValues("rainbondcluster", types.NamespacedName{Name: rbdutil.GetenvDefault("RBD_NAMESPACE", constants.Namespace)})
	err := r.Client.List(context.Background(), nodeList)
	if err != nil {
		reqLogger.V(4).Error(err, "get rainbond gatewayNodes or ChaosNodes")
		return
	}
	nodes := nodeList.Items
	for _, node := range nodes {
		if node.Annotations["rainbond.io/gateway-node"] == "true" {
			gatewayNodes = append(gatewayNodes, getK8sNode(node))
		}
		if node.Annotations["rainbond.io/chaos-node"] == "true" {
			chaosNodes = append(chaosNodes, getK8sNode(node))
		}
	}
	if len(gatewayNodes) == 0 {
		if len(nodes) < 2 {
			gatewayNodes = []*rainbondv1alpha1.K8sNode{
				getK8sNode(nodes[0]),
			}
		} else {
			gatewayNodes = []*rainbondv1alpha1.K8sNode{
				getK8sNode(nodes[0]),
				getK8sNode(nodes[1]),
			}
		}
	}
	if len(chaosNodes) == 0 {
		if len(nodes) < 2 {
			chaosNodes = []*rainbondv1alpha1.K8sNode{
				getK8sNode(nodes[0]),
			}
		} else {
			chaosNodes = []*rainbondv1alpha1.K8sNode{
				getK8sNode(nodes[0]),
				getK8sNode(nodes[1]),
			}
		}
	}
	gatewayNodes = r.ChoiceAvailableGatewayNode(gatewayNodes)
	return
}

func getK8sNode(node corev1.Node) *rainbondv1alpha1.K8sNode {
	var Knode rainbondv1alpha1.K8sNode
	for _, address := range node.Status.Addresses {
		if address.Type == corev1.NodeInternalIP {
			Knode.InternalIP = address.Address
		}
		if address.Type == corev1.NodeExternalIP {
			Knode.ExternalIP = address.Address
		}
		if address.Type == corev1.NodeHostName {
			Knode.Name = address.Address
		}
	}
	return &Knode
}

// ChoiceAvailableGatewayNode choice nodes as gateway which some ports not in used
func (r *RainbondClusterReconciler) ChoiceAvailableGatewayNode(nodes []*rainbondv1alpha1.K8sNode) []*rainbondv1alpha1.K8sNode {
	var availableGatewayNodes []*rainbondv1alpha1.K8sNode
	portOccupiedNode := make(map[string]struct{})
	ports := []string{rbdutil.GetenvDefault("GATEWAY_HTTP_PORT", "80"), rbdutil.GetenvDefault("GATEWAY_HTTPS_PORT", "443"), rbdutil.GetenvDefault("API_WS_PORT", "6060"), rbdutil.GetenvDefault("API_PORT", "8443")}
	if os.Getenv("CONSOLE_DOMAIN") == "" {
		ports = append(ports, "7070")
	}
	for _, node := range nodes {
		for _, port := range ports {
			address := net.JoinHostPort(node.InternalIP, port)
			conn, err := net.DialTimeout("tcp", address, 1*time.Second)
			if err != nil {
				continue
			}
			if conn != nil {
				r.Log.Info(fmt.Sprintf("Node [%s] port [%s] is already in use and cannot be used as a gateway node", node.Name, port))
				portOccupiedNode[node.Name] = struct{}{}
				_ = conn.Close()
				break
			}
		}
		if _, portOccupied := portOccupiedNode[node.Name]; !portOccupied {
			availableGatewayNodes = append(availableGatewayNodes, node)
		}
	}
	return availableGatewayNodes
}
