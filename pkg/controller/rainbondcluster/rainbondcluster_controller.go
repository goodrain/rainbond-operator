package rainbondcluster

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/goodrain/rainbond-operator/pkg/util/constants"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/retry"
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
			Name: "rainbondcluster",
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
	reqLogger.V(6).Info("Reconciling RainbondCluster")

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

	mgr := newRbdcomponentMgr(ctx, r.client, reqLogger, rainbondcluster, r.scheme)

	// generate status for rainbond cluster
	status, err := mgr.generateRainbondClusterStatus()
	if err != nil {
		reqLogger.Error(err, "failed to generate rainbondcluster status")
		return reconcile.Result{RequeueAfter: time.Second * 2}, err
	}
	rainbondcluster.Status = status
	if err := r.client.Status().Update(ctx, rainbondcluster); err != nil {
		reqLogger.Error(err, "failed to update rainbondcluster status")
		return reconcile.Result{RequeueAfter: time.Second * 2}, err
	}

	if rainbondcluster.Spec.ImageHub == nil && rainbondcluster.Spec.ConfigCompleted {
		reqLogger.V(6).Info("image hub is empty, do sth.")

		if err := mgr.checkIfRbdNodeReady(); err != nil {
			reqLogger.V(6).Info("rbd-node not ready: %v", err)
			return reconcile.Result{RequeueAfter: time.Second * 1}, nil
		}

		imageHub, err := r.getImageHub(rainbondcluster)
		if err != nil {
			reqLogger.V(6).Info(fmt.Sprintf("set image hub info: %v", err))
			return reconcile.Result{RequeueAfter: time.Second * 1}, nil
		}
		rainbondcluster.Spec.ImageHub = imageHub
		if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			return r.client.Update(ctx, rainbondcluster)
		}); err != nil {
			reqLogger.Error(err, "update rainbondcluster")
			return reconcile.Result{RequeueAfter: time.Second * 1}, err
		}
	}

	// create secret for pulling images.
	if rainbondcluster.Spec.ImageHub != nil && rainbondcluster.Spec.ImageHub.Username != "" && rainbondcluster.Spec.ImageHub.Password != "" {
		err := mgr.createImagePullSecret()
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{RequeueAfter: 3 * time.Second}, nil
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
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
		defer cancel()
		eip := cluster.FirstGatewayEIP()
		if eip == "" {
			return fmt.Errorf("no external ip found for gateway")
		}
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("https://%s/v2/", cluster.FirstGatewayEIP()), nil)
		if err != nil {
			return fmt.Errorf("new request failure %s", err.Error())
		}
		request.Host = constants.DefImageRepository
		res, err := httpClient.Do(request)
		if err != nil {
			return fmt.Errorf("image repository unavailable: %v", err)
		}
		if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusUnauthorized {
			return fmt.Errorf("image repository unavailable. http status code: %d", res.StatusCode)
		}
		return nil
	}
	if err := imageHubReady(); err != nil {
		return nil, fmt.Errorf("image repository not ready: %v", err)
	}

	return &rainbondv1alpha1.ImageHub{
		Domain:   constants.DefImageRepository,
		Username: cluster.Status.ImagePullUsername,
		Password: cluster.Status.ImagePullPassword,
	}, nil
}
