package handler

import (
	"context"
	"fmt"
	"os/exec"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
	"github.com/goodrain/rainbond-operator/util/commonutil"
	"github.com/goodrain/rainbond-operator/util/constants"
	"github.com/goodrain/rainbond-operator/util/rbdutil"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//HubName name
var HubName = "rbd-hub"
var hubDataPvcName = "rbd-hub"
var hubImageRepository = "hub-image-repository"

type hub struct {
	ctx       context.Context
	client    client.Client
	component *rainbondv1alpha1.RbdComponent
	cluster   *rainbondv1alpha1.RainbondCluster
	labels    map[string]string

	password string
	htpasswd []byte

	pvcParametersRWX *pvcParameters
	storageRequest   int64
}

var _ ComponentHandler = &hub{}
var _ StorageClassRWXer = &hub{}

//NewHub nw hub
func NewHub(ctx context.Context, client client.Client, component *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster) ComponentHandler {
	return &hub{
		component:      component,
		cluster:        cluster,
		client:         client,
		ctx:            ctx,
		labels:         LabelsForRainbondComponent(component),
		storageRequest: getStorageRequest("HUB_DATA_STORAGE_REQUEST", 40),
	}
}

func (h *hub) Before() error {
	if h.cluster.Spec.ImageHub != nil && h.cluster.Spec.ImageHub.Domain != constants.DefImageRepository {
		return NewIgnoreError("use custom image repository")
	}

	if err := setStorageCassName(h.ctx, h.client, h.component.Namespace, h); err != nil {
		return err
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
		h.deployment(),
		h.serviceForHub(),
		h.persistentVolumeClaimForHub(),
		h.ingressForHub(),
	}
}

func (h *hub) After() error {
	return nil
}

func (h *hub) ListPods() ([]corev1.Pod, error) {
	return listPods(h.ctx, h.client, h.component.Namespace, h.labels)
}

func (h *hub) SetStorageClassNameRWX(pvcParameters *pvcParameters) {
	h.pvcParametersRWX = pvcParameters
}

func (h *hub) deployment() client.Object {
	env := []corev1.EnvVar{
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
	}
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "hubdata",
			MountPath: "/var/lib/registry",
		},
		{
			Name:      "htpasswd",
			MountPath: "/auth",
			ReadOnly:  true,
		},
	}
	volumes := []corev1.Volume{
		{
			Name: "hubdata",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: hubDataPvcName,
				},
			},
		},
		{
			Name: "htpasswd",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: "hub-image-repository",
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

func (h *hub) persistentVolumeClaimForHub() *corev1.PersistentVolumeClaim {
	return createPersistentVolumeClaimRWX(h.component.Namespace, hubDataPvcName, h.pvcParametersRWX, h.labels)
}

func (h *hub) ingressForHub() client.Object {
	ing := &extensions.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      HubName,
			Namespace: h.component.Namespace,
			Annotations: map[string]string{
				"nginx.ingress.kubernetes.io/weight":                       "100",
				"nginx.ingress.kubernetes.io/upstream-hash-by":             "$remote_addr", // consistent hashing
				"nginx.ingress.kubernetes.io/proxy-body-size":              "0",
				"nginx.ingress.kubernetes.io/set-header-Host":              "$http_host",
				"nginx.ingress.kubernetes.io/set-header-X-Forwarded-Host":  "$http_host",
				"nginx.ingress.kubernetes.io/set-header-X-Forwarded-Proto": "https",
				"nginx.ingress.kubernetes.io/set-header-X-Scheme":          "$scheme",
			},
			Labels: h.labels,
		},
		Spec: extensions.IngressSpec{
			Rules: []extensions.IngressRule{
				{
					Host: constants.DefImageRepository,
					IngressRuleValue: extensions.IngressRuleValue{
						HTTP: &extensions.HTTPIngressRuleValue{
							Paths: []extensions.HTTPIngressPath{
								{
									Path: "/v2/",
									Backend: extensions.IngressBackend{
										ServiceName: HubName,
										ServicePort: intstr.FromInt(5000),
									},
								},
							},
						},
					},
				},
			},
			TLS: []extensions.IngressTLS{
				{
					Hosts:      []string{rbdutil.GetImageRepository(h.cluster)},
					SecretName: hubImageRepository,
				},
			},
		},
	}

	return ing
}

func (h *hub) secretForHub() client.Object {
	secret, _ := h.getSecret(hubImageRepository)
	if secret != nil && string(secret.Data["HTPASSWD"]) == string(h.htpasswd) {
		return nil
	}
	labels := copyLabels(h.labels)
	labels["name"] = hubImageRepository
	_, pem, key, _ := commonutil.DomainSign(nil, rbdutil.GetImageRepository(h.cluster))
	secret = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      hubImageRepository,
			Namespace: h.component.Namespace,
			Labels:    labels,
		},
		Data: map[string][]byte{
			"tls.crt":  pem,
			"tls.key":  key,
			"cert":     pem,
			"HTPASSWD": h.htpasswd,
			"password": []byte(h.password),
		},
	}

	return secret
}

func (h *hub) getSecret(name string) (*corev1.Secret, error) {
	return getSecret(h.ctx, h.client, h.component.Namespace, name)
}

func (h *hub) generateHtpasswd() ([]byte, error) {
	cmd := exec.Command("htpasswd", "-Bbn", h.cluster.Spec.ImageHub.Username, h.cluster.Spec.ImageHub.Password)
	return cmd.CombinedOutput()
}
