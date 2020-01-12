package rainbondpackage

import (
	"context"
	"fmt"
	"github.com/GLYASAI/rainbond-operator/pkg/util/k8sutil"
	"github.com/GLYASAI/rainbond-operator/pkg/util/retryutil"
	"k8s.io/apimachinery/pkg/types"
	"math"
	"reflect"
	"time"

	rainbondv1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_rainbondpackage")

// Add creates a new RainbondPackage Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileRainbondPackage{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("rainbondpackage-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource RainbondPackage
	err = c.Watch(&source.Kind{Type: &rainbondv1alpha1.RainbondPackage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rainbondpackage",
			Namespace: "rbd-system",
		},
	}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileRainbondPackage implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileRainbondPackage{}

// ReconcileRainbondPackage reconciles a RainbondPackage object
type ReconcileRainbondPackage struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a RainbondPackage object and makes changes based on the state read
// and what is in the RainbondPackage.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileRainbondPackage) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling RainbondPackage")

	// Fetch the RainbondPackage instance
	pkg := &rainbondv1alpha1.RainbondPackage{}
	err := r.client.Get(context.TODO(), request.NamespacedName, pkg)
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

	p := newpkg(r.client, pkg)

	// check prerequisites
	cluster := &rainbondv1alpha1.RainbondCluster{}
	if err := r.client.Get(context.TODO(), types.NamespacedName{Namespace: pkg.Namespace, Name: "rainbondcluster"}, cluster); err != nil {
		reqLogger.Error(err, "failed to get rainbondcluster.")
		p.setMessage(fmt.Sprintf("failed to get rainbondcluster: %v", err))
		p.reportFailedStatus()
		return reconcile.Result{Requeue: true}, nil
	}
	p.setCluster(cluster)
	if !p.preCheck() {
		p.status.Phase = rainbondv1alpha1.RainbondPackageWaiting
		if err := p.updateCRStatus(); err != nil {
			reqLogger.Error(err, "failed to update rainbondpackage status.")
		}
		return reconcile.Result{Requeue: true}, nil
	}

	return reconcile.Result{}, nil
}

type pkg struct {
	client  client.Client
	pkg     *rainbondv1alpha1.RainbondPackage
	status  *rainbondv1alpha1.RainbondPackageStatus
	cluster *rainbondv1alpha1.RainbondCluster
}

func newpkg(client client.Client, p *rainbondv1alpha1.RainbondPackage) *pkg {
	return &pkg{
		client: client,
		pkg:    p,
		status: &rainbondv1alpha1.RainbondPackageStatus{},
	}
}

func (p *pkg) setCluster(c *rainbondv1alpha1.RainbondCluster) {
	p.cluster = c
}

func (p *pkg) reportFailedStatus() {
	log.Info("rainbondpackage failed. Reporting failed reason...")

	retryInterval := 5 * time.Second
	f := func() (bool, error) {
		p.status.Phase = rainbondv1alpha1.RainbondPackageFailed
		err := p.updateCRStatus()
		if err == nil || k8sutil.IsKubernetesResourceNotFoundError(err) {
			return true, nil
		}

		if !errors.IsConflict(err) {
			log.Info("retry report status in %v: fail to update: %v", retryInterval, err)
			return false, nil
		}

		rp := &rainbondv1alpha1.RainbondPackage{}
		err = p.client.Get(context.TODO(), types.NamespacedName{Namespace: p.pkg.Namespace, Name: p.pkg.Name}, rp)
		if err != nil {
			// Update (PUT) will return conflict even if object is deleted since we have UID set in object.
			// Because it will check UID first and return something like:
			// "Precondition failed: UID in precondition: 0xc42712c0f0, UID in object meta: ".
			if k8sutil.IsKubernetesResourceNotFoundError(err) {
				return true, nil
			}
			log.Info("retry report status in %v: fail to get latest version: %v", retryInterval, err)
			return false, nil
		}

		p.pkg = rp
		return false, nil
	}

	_ = retryutil.Retry(retryInterval, math.MaxInt64, f)
}

func (p *pkg) updateCRStatus() error {
	if reflect.DeepEqual(p.pkg.Status, p.status) {
		return nil
	}

	newPackage := p.pkg
	newPackage.Status = p.status
	err := p.client.Status().Update(context.TODO(), newPackage)
	if err != nil {
		return fmt.Errorf("failed to update rainbondpackage status: %v", err)
	}

	p.pkg = newPackage

	return nil
}

func (p *pkg) setMessage(msg string) {
	p.status.Message = msg
}

func (p *pkg) preCheck() bool {
	if p.cluster == nil {
		return false
	}

	c := p.findCondition(rainbondv1alpha1.ImageRepositoryInstalled)
	if c == nil || c.Status != rainbondv1alpha1.ConditionTrue {
		return false
	}

	return true
}

func (p *pkg) findCondition(typ3 rainbondv1alpha1.RainbondClusterConditionType) *rainbondv1alpha1.RainbondClusterCondition {
	for _, condition := range p.cluster.Status.Conditions {
		if condition.Type == typ3 {
			return &condition
		}
	}
	return nil
}
