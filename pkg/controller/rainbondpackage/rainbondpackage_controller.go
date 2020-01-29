package rainbondpackage

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/GLYASAI/rainbond-operator/pkg/util/tarutil"
	"github.com/pquerna/ffjson/ffjson"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	rainbondv1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/GLYASAI/rainbond-operator/pkg/util/commonutil"
	"github.com/GLYASAI/rainbond-operator/pkg/util/k8sutil"
	"github.com/GLYASAI/rainbond-operator/pkg/util/retryutil"

	"github.com/docker/distribution/reference"
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

var pkgDst = "/opt/rainbond/pkg/files"

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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Fetch the RainbondPackage instance
	pkg := &rainbondv1alpha1.RainbondPackage{}
	err := r.client.Get(ctx, request.NamespacedName, pkg)
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

	dcli, err := newDockerClient(ctx)
	if err != nil {
		reqLogger.Error(err, "failed to create docker client")
		return reconcile.Result{Requeue: true}, nil
	}

	p := newpkg(ctx, r.client, dcli, pkg)
	if p.status.Phase == rainbondv1alpha1.RainbondPackageCompleted {
		return reconcile.Result{}, nil
	}

	// check prerequisites
	cluster := &rainbondv1alpha1.RainbondCluster{}
	if err := r.client.Get(ctx, types.NamespacedName{Namespace: pkg.Namespace, Name: "rainbondcluster"}, cluster); err != nil {
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

	// handle package, extract, load images and push images
	if err = p.handle(); err != nil {
		reqLogger.Error(err, "failed to handle rainbond package.")
		p.status.Message = fmt.Sprintf("failed to handle rainbond package %s: %v", p.pkg.Spec.PkgPath, err)
		p.reportFailedStatus()
		return reconcile.Result{RequeueAfter: 5 * time.Second}, nil
	}

	return reconcile.Result{RequeueAfter: 5 * time.Second}, nil
}

type pkg struct {
	ctx     context.Context
	client  client.Client
	dcli    *dclient.Client
	pkg     *rainbondv1alpha1.RainbondPackage
	status  *rainbondv1alpha1.RainbondPackageStatus
	cluster *rainbondv1alpha1.RainbondCluster
}

func newpkg(ctx context.Context, client client.Client, dcli *dclient.Client, p *rainbondv1alpha1.RainbondPackage) *pkg {
	pkg := &pkg{
		ctx:    ctx,
		client: client,
		pkg:    p,
		dcli:   dcli,
	}
	pkg.status = p.Status.DeepCopy()
	if pkg.status == nil {
		pkg.status = &rainbondv1alpha1.RainbondPackageStatus{
			FilesNumber: 20,
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
		err = p.client.Get(p.ctx, types.NamespacedName{Namespace: p.pkg.Namespace, Name: p.pkg.Name}, rp)
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
	// TODO: retry
	newPackage := p.pkg
	newPackage.Status = p.status
	err := p.client.Status().Update(p.ctx, newPackage)
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

		p.status.Phase = rainbondv1alpha1.RainbondPackagePushing
		p.status.NumberExtracted = p.status.FilesNumber
		p.clearMessageAndReason()
		if err := p.updateCRStatus(); err != nil {
			p.status.Reason = "ErrUpdatePhase"
			return fmt.Errorf("failed to update phase %s: %v", rainbondv1alpha1.RainbondPackagePushing, err)
		}
	}

	if p.status.Phase == rainbondv1alpha1.RainbondPackagePushing {
		if err := p.imagesLoadAndPush(); err != nil {
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
	errCh := make(chan error)
	go func(errCh chan error) {
		_ = os.RemoveAll(pkgDst)
		_ = os.MkdirAll(pkgDst, os.ModePerm)
		if err := tarutil.Untartar(p.pkg.Spec.PkgPath, pkgDst); err != nil {
			errCh <- err
			return
		}
		errCh <- nil
	}(errCh)

	tick := time.Tick(1 * time.Second)
	for {
		select {
		case err := <-errCh:
			return err
		case <-tick:
			// count files
			num := countImages(pkgDst)
			if p.status.NumberExtracted == num {
				continue
			}
			p.status.NumberExtracted = num
			if err := p.updateCRStatus(); err != nil {
				// ignore error
				log.Info("update number extracted: %v", err)
			}
		}
	}
}

func (p *pkg) imagesLoadAndPush() error {
	p.status.ImagesNumber = countImages(pkgDst)
	p.status.ImagesPushed = make(map[string]struct{})
	if err := p.updateCRStatus(); err != nil {
		return fmt.Errorf("failed to update image status: %v", err)
	}

	walkFn := func(pstr string, info os.FileInfo, err error) error {
		l := log.WithValues("file", pstr)
		if err != nil {
			l.Info(fmt.Sprintf("prevent panic by handling failure accessing a path %q: %v\n", pstr, err))
			return fmt.Errorf("prevent panic by handling failure accessing a path %q: %v\n", pstr, err)
		}
		if !commonutil.IsFile(pstr) {
			return nil
		}
		if !validateFile(pstr) {
			l.Info("invalid file, skip it1")
			return nil
		}

		f := func() (bool, error) {
			image, err := p.imageLoad(pstr)
			if err != nil {
				l.Error(err, "load image")
				return false, fmt.Errorf("load image: %v", err)
			}

			newImage := newImageWithNewDomain(image, p.cluster.ImageRepository())

			if err := p.dcli.ImageTag(p.ctx, image, newImage); err != nil {
				l.Error(err, "tag image", "source", image, "target", newImage)
				return false, fmt.Errorf("tag image: %v", err)
			}

			if err = p.imagePush(newImage); err != nil {
				l.Error(err, "push image", "image", newImage)
				return false, fmt.Errorf("push image %s: %v", newImage, err)
			}

			p.status.ImagesPushed[newImage] = struct{}{}
			if err := p.updateCRStatus(); err != nil {
				return false, fmt.Errorf("update cr status: %v", err)
			}
			l.Info("successfully load image", "image", newImage)
			return true, nil
		}

		return retryutil.Retry(1*time.Second, 3, f)
	}

	return filepath.Walk(pkgDst, walkFn)
}

func (p *pkg) imageLoad(file string) (string, error) {
	log.Info("start loading image", "file", file)
	f, err := os.Open(file)
	if err != nil {
		return "", fmt.Errorf("open file %s: %v", file, err)
	}
	res, err := p.dcli.ImageLoad(p.ctx, f, true) // load one, push one.
	if err != nil {
		return "", fmt.Errorf("path: %s; failed to load images: %v", file, err)
	}
	var imageName string
	if res.Body != nil {
		defer res.Body.Close()
		dec := json.NewDecoder(res.Body)
		for {
			select {
			case <-p.ctx.Done():
				log.Error(p.ctx.Err(), "error form context")
				return "", p.ctx.Err()
			default:
			}
			var jm JSONMessage
			if err := dec.Decode(&jm); err != nil {
				if err == io.EOF {
					break
				}
				return "", fmt.Errorf("failed to decode json message: %v", err)
			}
			if jm.Error != nil {
				return "", fmt.Errorf("error detail: %v", jm.Error)
			}
			msg := jm.JSONString()
			log.Info("response from image loading", "msg", msg)
			if imageName == "" {
				imageName, err = parseImageName(msg)
				if err != nil {
					return "", fmt.Errorf("failed to parse image name: %v", err)
				}
			}
		}
	}
	return imageName, nil
}

func (p *pkg) imagePush(image string) error {
	var opts dtypes.ImagePushOptions
	authConfig := dtypes.AuthConfig{
		ServerAddress: p.cluster.ImageRepository(),
	}
	if p.cluster.Spec.ImageHub != nil {
		authConfig.Username = p.cluster.Spec.ImageHub.Username
		authConfig.Password = p.cluster.Spec.ImageHub.Password
	}
	registryAuth, err := encodeAuthToBase64(authConfig)
	if err != nil {
		return fmt.Errorf("failed to encode auth config: %v", err)
	}
	opts.RegistryAuth = registryAuth

	res, err := p.dcli.ImagePush(p.ctx, image, opts)
	if err != nil {
		log.Error(err, "failed to push image", "image", image)
		return fmt.Errorf("push image %s: %v", image, err)
	}
	if res != nil {
		defer res.Close()

		dec := json.NewDecoder(res)
		for {
			select {
			case <-p.ctx.Done():
				log.Error(p.ctx.Err(), "error form context")
				return p.ctx.Err()
			default:
			}
			// TODO: use  github.com/docker/docker/pkg/jsonmessage
			// ref: https://stackoverflow.com/questions/44452679/golang-docker-api-parse-result-of-imagepull/44509434
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

	return nil
}

func pkgDir(pkgPath, dst string) string {
	ext := path.Ext(pkgPath)
	dirName := strings.Replace(path.Base(pkgPath), ext, "", -1)
	return path.Join(dst, dirName)
}

func newDockerClient(ctx context.Context) (*dclient.Client, error) {
	cli, err := dclient.NewClientWithOpts(dclient.FromEnv)
	if err != nil {
		log.Error(err, "create new docker client")
		return nil, fmt.Errorf("create new docker client: %v", err)
	}
	cli.NegotiateAPIVersion(ctx)

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
	if strings.HasPrefix(str, "Loaded image ID:") {
		return "", nil
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

func countImages(dir string) int32 {
	l := log.WithName("count images")
	var count int32
	_ = filepath.Walk(dir, func(pstr string, info os.FileInfo, err error) error {
		if err != nil {
			l.Info(fmt.Sprintf("walk path %s: %v", pstr, err))
			return nil
		}
		if !commonutil.IsFile(pstr) {
			return nil
		}
		if !validateFile(pstr) {
			return nil
		}
		count++
		return nil
	})

	return count
}

func validateFile(file string) bool {
	base := path.Base(file)
	if path.Ext(base) != ".tgz" || strings.HasPrefix(base, "._") {
		return false
	}
	return true
}

func newImageWithNewDomain(image string, newDomain string) string {
	repo, _ := reference.Parse(image)
	named := repo.(reference.Named)
	remoteName := reference.Path(named)
	tag := "latest"
	if t, ok := repo.(reference.Tagged); ok {
		tag = t.Tag()
	}
	return path.Join(newDomain, remoteName+":"+tag)
}
