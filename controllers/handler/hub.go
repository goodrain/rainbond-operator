package handler

import (
	"context"
	"fmt"
	v2 "github.com/goodrain/rainbond-operator/api/v2"
	"github.com/sirupsen/logrus"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"os/exec"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
	"github.com/goodrain/rainbond-operator/util/commonutil"
	"github.com/goodrain/rainbond-operator/util/constants"
	"github.com/goodrain/rainbond-operator/util/rbdutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// HubName name
var HubName = "rbd-hub"
var hubDataPvcName = "rbd-hub"
var hubImageRepository = "hub-image-repository"
var hubPasswordSecret = "hub-password"

type hub struct {
	ctx       context.Context
	client    client.Client
	component *rainbondv1alpha1.RbdComponent
	cluster   *rainbondv1alpha1.RainbondCluster
	labels    map[string]string

	password string
	htpasswd []byte

	storageRequest int64
}

var _ ComponentHandler = &hub{}

// NewHub nw hub
func NewHub(ctx context.Context, client client.Client, component *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster) ComponentHandler {
	return &hub{
		component:      component,
		cluster:        cluster,
		client:         client,
		ctx:            ctx,
		labels:         LabelsForRainbondComponent(component),
		storageRequest: getStorageRequest("HUB_DATA_STORAGE_REQUEST", 10),
	}
}

func (h *hub) Before() error {
	if h.cluster.Spec.ImageHub != nil && h.cluster.Spec.ImageHub.Domain != constants.DefImageRepository {
		return NewIgnoreError("use custom image repository")
	}

	if h.cluster.Spec.ImageHub == nil {
		return NewIgnoreError("imageHub is empty")
	}

	htpasswd, err := h.generateHtpasswd()
	if err != nil {
		return fmt.Errorf("generate htpasswd: %v", err)
	}
	h.htpasswd = htpasswd

	return nil
}

func (h *hub) Resources() []client.Object {
	return []client.Object{
		h.secretForHub(), // important! create secret before ingress.
		h.passwordSecret(),
		h.deployment(),
		h.serviceForHub(),
		h.hubImageRepository(), // 绑定这个镜像仓库的secret
		h.ingressForHub(),      //创建这个域名的路由
	}
}

func (h *hub) hubImageRepository() client.Object {
	const Name = "hub-image-repository"
	return &v2.ApisixTls{
		ObjectMeta: metav1.ObjectMeta{
			Name:      Name,
			Namespace: rbdutil.GetenvDefault("RBD_NAMESPACE", constants.Namespace),
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       constants.ApisixTls,
			APIVersion: constants.APISixAPIVersion,
		},
		Spec: &v2.ApisixTlsSpec{
			Hosts: []v2.HostType{
				"goodrain.me",
			},
			Secret: v2.ApisixSecret{
				Name:      Name,
				Namespace: rbdutil.GetenvDefault("RBD_NAMESPACE", constants.Namespace),
			},
		},
	}
}

func (h *hub) ingressForHub() client.Object {
	return &v2.ApisixRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rbd-hub",
			Namespace: rbdutil.GetenvDefault("RBD_NAMESPACE", constants.Namespace),
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       constants.ApisixRoute,
			APIVersion: constants.APISixAPIVersion,
		},
		Spec: v2.ApisixRouteSpec{
			HTTP: []v2.ApisixRouteHTTP{
				{
					Name: "rbd-hub",
					Match: v2.ApisixRouteHTTPMatch{
						Hosts: []string{
							"goodrain.me",
						},
						Paths: []string{
							"/*",
						},
					},
					Backends: []v2.ApisixRouteHTTPBackend{
						{
							ServicePort: intstr.FromInt(5000),
							ServiceName: "rbd-hub",
						},
					},
					Authentication: v2.ApisixRouteAuthentication{
						Enable: false,
						Type:   "basicAuth",
					},
				},
			},
		},
	}
}

func (h *hub) After() error {
	return nil
}

func (h *hub) ListPods() ([]corev1.Pod, error) {
	return listPods(h.ctx, h.client, h.component.Namespace, h.labels)
}

func (h *hub) deployment() client.Object {
	env := []corev1.EnvVar{
		{
			Name:  "RBD_NAMESPACE",
			Value: h.component.Namespace,
		},
		{
			Name:  "REGISTRY_AUTH",
			Value: "htpasswd",
		},
		{
			Name:  "REGISTRY_AUTH_HTPASSWD_REALM",
			Value: "Registry Realm",
		},
		{
			Name:  "REGISTRY_AUTH_HTPASSWD_PATH",
			Value: "/auth/htpasswd",
		},
		{
			Name:  "REGISTRY_STORAGE",
			Value: "s3",
		},
		{
			Name:  "REGISTRY_STORAGE_S3_REGION",
			Value: "rainbond",
		},
		{
			Name:  "REGISTRY_STORAGE_S3_ACCESSKEY",
			Value: "minioadmin",
		},
		{
			Name:  "REGISTRY_STORAGE_S3_SECRETKEY",
			Value: "minioadmin",
		},
		{
			Name:  "REGISTRY_STORAGE_S3_REGIONENDPOINT",
			Value: "http://minio-service:9000",
		},
		{
			Name:  "REGISTRY_STORAGE_S3_BUCKET",
			Value: "rbd-hub",
		},
		{
			Name:  "REGISTRY_STORAGE_REDIRECT_DISABLE",
			Value: "true",
		},
	}
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "htpasswd",
			MountPath: "/auth",
			ReadOnly:  true,
		},
	}
	volumes := []corev1.Volume{
		{
			Name: "htpasswd",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: hubPasswordSecret,
					Items: []corev1.KeyToPath{
						{
							Key:  "HTPASSWD",
							Path: "htpasswd",
						},
					},
				},
			},
		},
	}

	env = mergeEnvs(env, h.component.Spec.Env)
	volumeMounts = mergeVolumeMounts(volumeMounts, h.component.Spec.VolumeMounts)
	volumes = mergeVolumes(volumes, h.component.Spec.Volumes)

	ds := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      HubName,
			Namespace: h.component.Namespace,
			Labels:    h.labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: h.component.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: h.labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   HubName,
					Labels: h.labels,
				},
				Spec: corev1.PodSpec{
					ImagePullSecrets:              imagePullSecrets(h.component, h.cluster),
					TerminationGracePeriodSeconds: commonutil.Int64(0),
					Containers: []corev1.Container{
						{
							Name:            "rbd-hub",
							Image:           h.component.Spec.Image,
							ImagePullPolicy: h.component.ImagePullPolicy(),
							Env:             env,
							VolumeMounts:    volumeMounts,
							Resources:       h.component.Spec.Resources,
						},
					},
					Volumes: volumes,
				},
			},
		},
	}

	return ds
}

func (h *hub) serviceForHub() client.Object {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      HubName,
			Namespace: h.component.Namespace,
			Labels:    h.labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name: "main",
					Port: 5000,
					TargetPort: intstr.IntOrString{
						IntVal: 5000,
					},
				},
			},
			Selector: h.labels,
		},
	}

	return svc
}

func (h *hub) secretForHub() client.Object {
	secret, err := h.getSecret(hubImageRepository)
	if secret != nil {
		// never update hub secret
		return nil
	}
	if err != nil && !k8sErrors.IsNotFound(err) {
		logrus.Errorf("get secret %s: %v", hubImageRepository, err)
		return nil
	}
	labels := copyLabels(h.labels)
	labels["name"] = hubImageRepository
	_, pem, key, _ := commonutil.DomainSign(nil, rbdutil.GetImageRepository(h.cluster))
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      hubImageRepository,
			Namespace: h.component.Namespace,
			Labels:    labels,
		},
		Data: map[string][]byte{
			"tls.crt": pem,
			"tls.key": key,
			"cert":    pem,
			"key":     key,
		},
	}
}

func (h *hub) passwordSecret() client.Object {
	labels := copyLabels(h.labels)
	labels["name"] = hubPasswordSecret
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      hubPasswordSecret,
			Namespace: h.component.Namespace,
			Labels:    labels,
		},
		Data: map[string][]byte{
			"HTPASSWD": h.htpasswd,
			"password": []byte(h.password),
		},
	}
}

func (h *hub) getSecret(name string) (*corev1.Secret, error) {
	return getSecret(h.ctx, h.client, h.component.Namespace, name)
}

func (h *hub) generateHtpasswd() ([]byte, error) {
	cmd := exec.Command("htpasswd", "-Bbn", h.cluster.Spec.ImageHub.Username, h.cluster.Spec.ImageHub.Password)
	return cmd.CombinedOutput()
}
