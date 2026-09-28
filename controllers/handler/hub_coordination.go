package handler

import (
	"errors"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/docker/distribution/reference"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/validation"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var errRegistryCoordination = errors.New("invalid registry coordination configuration")
var registryIdentity = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.:-]{0,63}$`)
var registryConsoleIdentity = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)
var registryPinnedImage = regexp.MustCompile(`^[^\s@]+@sha256:[a-f0-9]{64}$`)

func pathWithin(root, child string) bool {
	return child == root || strings.HasPrefix(child, strings.TrimSuffix(root, "/")+"/")
}

func (h *hub) applyRegistryCoordination(deployment *appsv1.Deployment) error {
	cfg := h.component.Spec.RegistryCoordination
	if cfg == nil {
		return nil
	}
	if h.component.Name != HubName || h.cluster.Spec.ImageHub == nil || !registryPinnedImage.MatchString(cfg.Image) ||
		!registryPinnedImage.MatchString(h.component.Spec.Image) || !registryConsoleIdentity.MatchString(cfg.EnterpriseID) ||
		!registryConsoleIdentity.MatchString(cfg.RegionName) || !registryIdentity.MatchString(cfg.StorageID) ||
		!registryIdentity.MatchString(cfg.Generation) || !registryIdentity.MatchString(cfg.VolumeUID) ||
		cfg.ControlSecret == cfg.PermitSecret || len(validation.IsDNS1123Subdomain(cfg.ControlSecret)) != 0 ||
		len(validation.IsDNS1123Subdomain(cfg.PermitSecret)) != 0 ||
		cfg.RegistryPath == "/" || !path.IsAbs(cfg.RegistryPath) || path.Clean(cfg.RegistryPath) != cfg.RegistryPath {
		return errRegistryCoordination
	}
	if replicas := h.component.Spec.Replicas; replicas != nil && (*replicas < 0 || *replicas > 1) {
		return errRegistryCoordination
	}
	origin, err := url.Parse(cfg.ConsoleURL)
	if err != nil || origin.Host == "" || origin.User != nil || origin.RawQuery != "" || origin.Fragment != "" ||
		(origin.Path != "" && origin.Path != "/") || (origin.Scheme != "https" && !(origin.Scheme == "http" && cfg.AllowHTTP)) {
		return errRegistryCoordination
	}
	coordinatorImage, err := reference.ParseNormalizedNamed(cfg.Image)
	if err != nil {
		return errRegistryCoordination
	}
	nativeImage, err := reference.ParseNormalizedNamed(h.component.Spec.Image)
	if err != nil {
		return errRegistryCoordination
	}
	imageOrigin := reference.Domain(coordinatorImage)
	protectedOrigin := h.cluster.Spec.ImageHub.Domain
	imageHost := (&url.URL{Host: imageOrigin}).Hostname()
	protectedHost := (&url.URL{Host: protectedOrigin}).Hostname()
	if imageHost == protectedHost || imageHost == "rbd-hub" || strings.HasPrefix(imageHost, "rbd-hub.") {
		return errRegistryCoordination
	}
	for _, reserved := range []string{"/registry-coordinator", "/registry-gc", "/bin/registry", "/control", "/permit", "/tmp"} {
		if pathWithin(cfg.RegistryPath, reserved) || pathWithin(reserved, cfg.RegistryPath) {
			return errRegistryCoordination
		}
	}
	pod := &deployment.Spec.Template.Spec
	if len(pod.Containers) != 1 || pod.HostNetwork || pod.HostIPC || pod.HostPID {
		return errRegistryCoordination
	}
	native := &pod.Containers[0]
	native.Image = nativeImage.String()
	required := map[string]string{"REGISTRY_HTTP_ADDR": "127.0.0.1:5000", "REGISTRY_STORAGE": "filesystem", "REGISTRY_STORAGE_FILESYSTEM_ROOTDIRECTORY": cfg.RegistryPath, "REGISTRY_STORAGE_MAINTENANCE_UPLOADPURGING_ENABLED": "false"}
	for _, e := range native.Env {
		if value, protected := required[e.Name]; protected && (e.ValueFrom != nil || e.Value != value) {
			// Only unset protected fields are supplied by this feature.
			return errRegistryCoordination
		}
	}
	native.Env = mergeEnvs(native.Env, []corev1.EnvVar{{Name: "REGISTRY_HTTP_ADDR", Value: "127.0.0.1:5000"}, {Name: "REGISTRY_STORAGE_MAINTENANCE_UPLOADPURGING_ENABLED", Value: "false"}})
	for _, v := range pod.Volumes {
		if v.Name == "cleanup-control" || v.Name == "cleanup-permit" {
			return errRegistryCoordination
		}
	}
	var dataMount *corev1.VolumeMount
	for _, m := range native.VolumeMounts {
		if m.MountPath != cfg.RegistryPath && pathWithin(cfg.RegistryPath, m.MountPath) {
			return errRegistryCoordination
		}
		if pathWithin(m.MountPath, cfg.RegistryPath) && (dataMount == nil || len(m.MountPath) > len(dataMount.MountPath)) {
			copy := m
			dataMount = &copy
		}
	}
	if dataMount == nil || dataMount.ReadOnly || dataMount.SubPathExpr != "" ||
		(dataMount.MountPropagation != nil && *dataMount.MountPropagation != corev1.MountPropagationNone) {
		return errRegistryCoordination
	}
	if dataMount.SubPath != "" && (path.IsAbs(dataMount.SubPath) || path.Clean(dataMount.SubPath) != dataMount.SubPath || strings.HasPrefix(dataMount.SubPath, "..")) {
		return errRegistryCoordination
	}
	found := false
	for _, v := range pod.Volumes {
		if v.Name == dataMount.Name && v.PersistentVolumeClaim != nil {
			found = true
		}
	}
	if !found {
		return errRegistryCoordination
	}
	relative := strings.TrimPrefix(strings.TrimPrefix(cfg.RegistryPath, dataMount.MountPath), "/")
	if relative != "" {
		dataMount.SubPath = path.Join(dataMount.SubPath, relative)
	}
	dataMount.MountPath = "/registry"
	no, yes := false, true
	mode := int32(0444)
	for _, s := range []struct{ name, secret string }{{"cleanup-control", cfg.ControlSecret}, {"cleanup-permit", cfg.PermitSecret}} {
		pod.Volumes = append(pod.Volumes, corev1.Volume{Name: s.name, VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{SecretName: s.secret, DefaultMode: &mode, Items: []corev1.KeyToPath{{Key: "key", Path: "key"}}}}})
	}
	binding := []string{"--storage-root=/registry", "--storage-id=" + cfg.StorageID, "--storage-generation=" + cfg.Generation, "--volume-uid=" + cfg.VolumeUID, "--registry-path=" + cfg.RegistryPath}
	security := &corev1.SecurityContext{AllowPrivilegeEscalation: &no, ReadOnlyRootFilesystem: &yes, Capabilities: &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}}}
	resources := setDefaultResources(cfg.Resources)
	init := corev1.Container{Name: "registry-coordination-init", Image: coordinatorImage.String(), ImagePullPolicy: corev1.PullIfNotPresent, Command: []string{"/registry-coordinator"}, Args: append(append([]string{}, binding...), "--initialize-storage-identity"), VolumeMounts: []corev1.VolumeMount{*dataMount}, SecurityContext: security.DeepCopy(), Resources: *resources.DeepCopy()}
	pod.InitContainers = append(pod.InitContainers, init)
	dataMount.ReadOnly = true
	args := append(append([]string{}, binding...), "--listen=:5001", "--upstream=http://127.0.0.1:5000", "--owner=registry-coordinator", "--coordination-api="+cfg.ConsoleURL, "--console-system-identity=true", "--console-enterprise="+cfg.EnterpriseID, "--console-region="+cfg.RegionName, "--allow-internal-http="+strconv.FormatBool(cfg.AllowHTTP), "--credential-file=/control/key", "--permit-key-file=/permit/key")
	readiness := &corev1.Probe{Handler: corev1.Handler{HTTPGet: &corev1.HTTPGetAction{Path: "/readyz", Port: intstr.FromInt(5001)}}, InitialDelaySeconds: 2, PeriodSeconds: 5, TimeoutSeconds: 5, FailureThreshold: 3}
	sidecar := corev1.Container{Name: "registry-coordinator", Image: coordinatorImage.String(), ImagePullPolicy: corev1.PullIfNotPresent, Command: []string{"/registry-coordinator"}, Args: args,
		Ports:           []corev1.ContainerPort{{Name: "coordinated", ContainerPort: 5001}},
		Env:             []corev1.EnvVar{{Name: "POD_NAME", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.name"}}}, {Name: "POD_UID", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.uid"}}}},
		VolumeMounts:    []corev1.VolumeMount{*dataMount, {Name: "cleanup-control", MountPath: "/control", ReadOnly: true}, {Name: "cleanup-permit", MountPath: "/permit", ReadOnly: true}},
		SecurityContext: security, Resources: resources, ReadinessProbe: readiness,
		LivenessProbe: &corev1.Probe{Handler: corev1.Handler{HTTPGet: &corev1.HTTPGetAction{Path: "/healthz", Port: intstr.FromInt(5001)}}, InitialDelaySeconds: 10, PeriodSeconds: 10, TimeoutSeconds: 5, FailureThreshold: 3},
	}
	native.ReadinessProbe = readiness.DeepCopy()
	pod.Containers = append(pod.Containers, sidecar)
	pod.AutomountServiceAccountToken = &no
	grace := int64(30)
	pod.TerminationGracePeriodSeconds = &grace
	deployment.Spec.Strategy = appsv1.DeploymentStrategy{Type: appsv1.RecreateDeploymentStrategyType}
	if deployment.Spec.Template.Annotations == nil {
		deployment.Spec.Template.Annotations = map[string]string{}
	}
	deployment.Spec.Template.Annotations["rainbond.io/registry-gc-executor"] = "v1"
	return nil
}

// A removed optional field must not expose native Registry traffic while an
// earlier coordinated deletion or maintenance operation might still exist.
func (h *hub) protectRegistryCoordination() error {
	if h.component.Spec.RegistryCoordination != nil {
		return nil
	}
	if h.client == nil {
		return errRegistryCoordination
	}
	var existing appsv1.Deployment
	err := h.client.Get(h.ctx, client.ObjectKey{Namespace: h.component.Namespace, Name: HubName}, &existing)
	if kerrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if existing.Spec.Template.Annotations["rainbond.io/registry-gc-executor"] == "v1" {
		return errors.New("registry coordination removal requires a verified migration")
	}
	for _, c := range existing.Spec.Template.Spec.Containers {
		if c.Name == "registry-coordinator" || (len(c.Command) > 0 && c.Command[0] == "/registry-coordinator") {
			return errors.New("registry coordination removal requires a verified migration")
		}
	}
	return nil
}
