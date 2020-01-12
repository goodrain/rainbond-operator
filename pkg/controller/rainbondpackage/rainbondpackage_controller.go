package rainbondpackage

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/pquerna/ffjson/ffjson"
	"io"
	"io/ioutil"

	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	rainbondv1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/GLYASAI/rainbond-operator/pkg/util/commonutil"
	"github.com/GLYASAI/rainbond-operator/pkg/util/k8sutil"
	"github.com/GLYASAI/rainbond-operator/pkg/util/retryutil"

	dtypes "github.com/docker/docker/api/types"
	dclient "github.com/docker/docker/client"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_rainbondpackage")

var pkgDst = "/opt/rainbond/pkg"

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
		p.status.Reason = "NotMeetPrerequisites"
		p.status.Message = "not meet the prerequisites"
		if err := p.updateCRStatus(); err != nil {
			reqLogger.Error(err, "failed to update rainbondpackage status.")
		}
		return reconcile.Result{Requeue: true}, nil
	}

	if p.status.Phase == rainbondv1alpha1.RainbondPackageCompleted {
		return reconcile.Result{}, nil
	}

	// handle package, extract, load images and push images
	if err = p.handle(); err != nil {
		reqLogger.Error(err, "failed to handle rainbond package.")
		p.status.Message = fmt.Sprintf("failed to handle rainbond package %s: %v", p.pkg.Spec.PkgPath, err)
		p.reportFailedStatus()
		return reconcile.Result{RequeueAfter: 5 * time.Second}, nil
	}

	if p.status.Phase == rainbondv1alpha1.RainbondPackageCompleted {
		return reconcile.Result{}, nil
	}

	return reconcile.Result{RequeueAfter: 5 * time.Second}, nil
}

type pkg struct {
	client  client.Client
	pkg     *rainbondv1alpha1.RainbondPackage
	status  *rainbondv1alpha1.RainbondPackageStatus
	cluster *rainbondv1alpha1.RainbondCluster
}

func newpkg(client client.Client, p *rainbondv1alpha1.RainbondPackage) *pkg {
	pkg := &pkg{
		client: client,
		pkg:    p,
	}
	pkg.status = p.Status.DeepCopy()
	if pkg.status == nil {
		pkg.status = &rainbondv1alpha1.RainbondPackageStatus{
			ImageStatus: map[string]rainbondv1alpha1.ImageStatus{},
		}
	}
	return pkg
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
			log.Info(fmt.Sprintf("retry report status in %v: fail to update: %v", retryInterval, err))
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
			log.Info(fmt.Sprintf("retry report status in %v: fail to get latest version: %v", retryInterval, err))
			return false, nil
		}

		p.pkg = rp
		return false, nil
	}

	_ = retryutil.Retry(retryInterval, 3, f)
}

func (p *pkg) updateCRStatus() error {
	//if reflect.DeepEqual(p.pkg.Status, p.status) {
	//	return nil
	//}

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

func (p *pkg) clearMessageAndReason() {
	p.status.Message = ""
	p.status.Reason = ""
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

func (p *pkg) handle() error {
	log.Info("start handling rainbond package.", "phase", p.status.Phase)

	if p.status.Phase == "" || p.status.Phase == rainbondv1alpha1.RainbondPackageFailed ||
		p.status.Phase == rainbondv1alpha1.RainbondPackageWaiting {
		p.status.Phase = rainbondv1alpha1.RainbondPackageExtracting
		p.clearMessageAndReason()
		if err := p.updateCRStatus(); err != nil {
			p.status.Reason = "ErrUpdatePhase"
			return fmt.Errorf("failed to update phase %s: %v", rainbondv1alpha1.RainbondPackageExtracting, err)
		}
	}

	if p.status.Phase == rainbondv1alpha1.RainbondPackageExtracting {
		if err := p.untartar(); err != nil {
			p.status.Reason = "ErrPkgExtract"
			return fmt.Errorf("failed to untar %s: %v", p.pkg.Spec.PkgPath, err)
		}
		log.Info("successfully extract rainbond package.")

		p.status.Phase = rainbondv1alpha1.RainbondPackageLoading
		p.clearMessageAndReason()
		if err := p.updateCRStatus(); err != nil {
			p.status.Reason = "ErrUpdatePhase"
			return fmt.Errorf("failed to update phase %s: %v", rainbondv1alpha1.RainbondPackageLoading, err)
		}
	}

	if p.status.Phase == rainbondv1alpha1.RainbondPackageLoading {
		if err := p.loadImages(); err != nil {
			p.status.Reason = "ErrImageLoad"
			return fmt.Errorf("failed to load images: %v", err)
		}
		log.Info("successfully load rainbond images")

		p.status.Phase = rainbondv1alpha1.RainbondPackagePushing
		p.clearMessageAndReason()
		if err := p.updateCRStatus(); err != nil {
			p.status.Reason = "ErrUpdatePhase"
			return fmt.Errorf("failed to update phase %s: %v", rainbondv1alpha1.RainbondPackagePushing, err)
		}
	}

	if p.status.Phase == rainbondv1alpha1.RainbondPackagePushing {
		if err := p.pushImages(); err != nil {
			p.status.Reason = "ErrImagePush"
			return fmt.Errorf("failed to push images: %v", err)
		}
		log.Info("successfully push rainbond images")

		p.status.Phase = rainbondv1alpha1.RainbondPackageCompleted
		p.clearMessageAndReason()
		if err := p.updateCRStatus(); err != nil {
			p.status.Reason = "ErrUpdatePhase"
			return fmt.Errorf("failed to update phase %s: %v", rainbondv1alpha1.RainbondPackagePushing, err)
		}
	}

	return nil
}

func (p *pkg) untartar() error {
	log.Info(fmt.Sprintf("start untartaring %s", p.pkg.Spec.PkgPath))

	file, err := os.Open(p.pkg.Spec.PkgPath)
	if err != nil {
		return fmt.Errorf("open file: %v", err)
	}

	pkgDir := path.Join(pkgDst, strings.Replace(path.Base(p.pkg.Spec.PkgPath), ".tgz", "", -1))
	if err := os.RemoveAll(pkgDir); err != nil {
		return fmt.Errorf("failed to cleanup package directory %s: %v", pkgDir, err)
	}

	if err := commonutil.Untar(file, pkgDst); err != nil {
		return err
	}
	return nil
}

func (p *pkg) loadImages() error {
	cli, err := newDockerClient()
	if err != nil {
		return fmt.Errorf("failed to create docker client: %v", err)
	}

	dir := pkgDir(p.pkg.Spec.PkgPath, pkgDst)
	mfile, err := os.Open(path.Join(dir, "metadata.json"))
	if err != nil && !os.IsExist(err) {
		return fmt.Errorf("failed to open %s: %v", path.Join(dir, "metadata.json"), err)
	}
	bytes, err := ioutil.ReadAll(mfile)
	if err != nil {
		return fmt.Errorf("failed to read from file: %v", err)
	}

	var images []string
	if err := ffjson.Unmarshal(bytes, &images); err != nil {
		return fmt.Errorf("failed to unmarshal images info: %v", err)
	}
	imageStatus := p.status.ImageStatus
	if imageStatus == nil {
		imageStatus = make(map[string]rainbondv1alpha1.ImageStatus, len(images))
	}
	for _, image := range images {
		image = trimLatest(image)
		if _, ok := imageStatus[image]; ok {
			continue
		}
		imageStatus[image] = rainbondv1alpha1.ImageStatus{}
	}
	p.status.ImageStatus = imageStatus
	if err := p.updateCRStatus(); err != nil {
		return fmt.Errorf("failed to update image status: %v", err)
	}

	return filepath.Walk(dir, func(pstr string, info os.FileInfo, err error) error {
		l := log.WithValues("file", pstr)
		if err != nil {
			l.Info(fmt.Sprintf("prevent panic by handling failure accessing a path %q: %v\n", pstr, err))
			return fmt.Errorf("prevent panic by handling failure accessing a path %q: %v\n", pstr, err)
		}
		if !commonutil.IsFile(pstr) {
			return nil
		}
		base := path.Base(pstr)
		if path.Ext(base) != ".tgz" || strings.HasPrefix(base, "._") {
			l.Info("invalid file, skip it")
			return nil
		}

		f, err := os.Open(pstr)
		if err != nil {
			return fmt.Errorf("open file %s: %v", pstr, err)
		}
		log.Info("start loading image", "file", pstr)
		ctx := context.Background()
		res, err := cli.ImageLoad(ctx, f, true)
		if err != nil {
			return fmt.Errorf("path: %s; failed to load images: %v", pstr, err)
		}
		var lastMsg string
		if res.Body != nil {
			defer res.Body.Close()
			dec := json.NewDecoder(res.Body)
			for {
				select {
				case <-ctx.Done():
					log.Error(ctx.Err(), "error form context")
					return ctx.Err()
				default:
				}
				var jm JSONMessage
				if err := dec.Decode(&jm); err != nil {
					if err == io.EOF {
						break
					}
					return fmt.Errorf("failed to decode json message: %v", err)
				}
				if jm.Error != nil {
					return fmt.Errorf("error detail: %v", jm.Error)
				}
				lastMsg = jm.JSONString()
				l.Info("response from image loading", "msg", lastMsg)
			}
		}
		l.Info(fmt.Sprintf("last message: %s", lastMsg))
		imageName, err := parseImageName(lastMsg)
		if err != nil {
			return fmt.Errorf("failed to parse image name: %v", err)
		}

		status, ok := p.status.ImageStatus[imageName]
		if ok {
			status.Loaded = true
			p.status.ImageStatus[imageName] = status
			if err := p.updateCRStatus(); err != nil {
				return fmt.Errorf("status: %#v; failed to update image status: %v", status, err)
			}
		} else {
			log.Info(fmt.Sprintf("%s is not image in metadata", imageName))
		}

		log.Info("successfully load images", "images", images)
		return nil
	})
}

func (p *pkg) pushImages() error {
	cli, err := newDockerClient() // TODO: put it into pkg
	if err != nil {
		return fmt.Errorf("failed to create docker client: %v", err)
	}

	var opts dtypes.ImagePushOptions
	registryAuth, err := encodeAuthToBase64(dtypes.AuthConfig{
		ServerAddress: "goodrain.me",
	})
	if err != nil {
		return fmt.Errorf("failed to encode auth config: %v", err)
	}
	opts.RegistryAuth = registryAuth

	ctx := context.Background()
	for image, status := range p.status.ImageStatus {
		if status.Pushed {
			log.Info("has been pushed", "image", image)
			continue
		}

		newImage := strings.Replace(image, "rainbond", "goodrain.me", -1)
		if err := cli.ImageTag(ctx, image, newImage); err != nil {
			log.Error(err, fmt.Sprintf("rename image %s", image))
			return fmt.Errorf("rename image %s: %v", image, err)
		}

		res, err := cli.ImagePush(ctx, newImage, opts)
		if err != nil {
			log.Error(err, "failed to push image", "image", newImage)
			return fmt.Errorf("push image %s: %v", newImage, err)
		}
		if res != nil {
			defer res.Close()

			dec := json.NewDecoder(res)
			for {
				select {
				case <-ctx.Done():
					log.Error(ctx.Err(), "error form context")
					return ctx.Err()
				default:
				}
				var jm JSONMessage
				if err := dec.Decode(&jm); err != nil {
					if err == io.EOF {
						break
					}
					return fmt.Errorf("failed to decode json message: %v", err)
				}
				if jm.Error != nil {
					return fmt.Errorf("error detail: %v", jm.Error)
				}
				log.Info("response from image pushing", "msg", jm.JSONString())
			}
		}

		status.Pushed = true
		p.status.ImageStatus[image] = status
		if err := p.updateCRStatus(); err != nil {
			return fmt.Errorf("status: %#v; failed to update image status: %v", status, err)
		}
		log.Info(fmt.Sprintf("Image %s pushed", newImage))
	}

	return nil
}

func pkgDir(pkgPath, dst string) string {
	dirName := strings.Replace(path.Base(pkgPath), ".tgz", "", -1)
	return path.Join(dst, dirName)
}

func newDockerClient() (*dclient.Client, error) {
	cli, err := dclient.NewClientWithOpts(dclient.FromEnv)
	if err != nil {
		log.Error(err, "create new docker client")
		return nil, fmt.Errorf("create new docker client: %v", err)
	}
	cli.NegotiateAPIVersion(context.TODO())

	return cli, nil
}

func parseImageName(s string) (string, error) {
	// {"stream":"Loaded image: abewang/rainbond-operator:v0.0.1\n"}
	m := map[string]string{}
	if err := ffjson.Unmarshal([]byte(s), &m); err != nil {
		return "", err
	}
	str, ok := m["stream"]
	if !ok {
		return "", fmt.Errorf("wrong format")
	}
	str = strings.Replace(str, "Loaded image: ", "", -1)
	str = strings.Replace(str, "\n", "", -1)
	str = trimLatest(str)
	return str, nil
}

func encodeAuthToBase64(authConfig dtypes.AuthConfig) (string, error) {
	buf, err := json.Marshal(authConfig)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(buf), nil
}

func trimLatest(str string) string {
	if !strings.HasSuffix(str, ":latest") {
		return str
	}
	return str[:len(str)-len(":latest")]
}
