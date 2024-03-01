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

	pvcParametersRWX *pvcParameters
	storageRequest   int64
}

var _ ComponentHandler = &hub{}
var _ StorageClassRWXer = &hub{}

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

func (a *hub) Before() error {
	if a.cluster.Spec.ImageHub != nil && a.cluster.Spec.ImageHub.Domain != constants.DefImageRepository {
		return NewIgnoreError("use custom image repository")
	}

	if a.cluster.Spec.ImageHub == nil {
		return NewIgnoreError("imageHub is empty")
	}

	htpasswd, err := a.generateHtpasswd()
	if err != nil {
		return fmt.Errorf("generate htpasswd: %v", err)
	}
	a.htpasswd = htpasswd

	if a.component.Labels["persistentVolumeClaimAccessModes"] == string(corev1.ReadWriteOnce) {
		sc, err := storageClassNameFromRainbondVolumeRWO(a.ctx, a.client, a.component.Namespace)
		if err != nil {
			return err
		}
		a.SetStorageClassNameRWX(sc)
		return nil
	}

	return setStorageCassName(a.ctx, a.client, a.component.Namespace, a)
}

func (a *hub) Resources() []client.Object {
	return []client.Object{
		a.secretForHub(), // important! create secret before ingress.
		a.passwordSecret(),
		a.deployment(),
		a.serviceForHub(),
		a.persistentVolumeClaimForHub(),
		a.hubImageRepository(), // 绑定这个镜像仓库的secret
		a.ingressForHub(),      //创建这个域名的路由
	}
}

func (a *hub) hubImageRepository() client.Object {
	const Name = "hub-image-repository"
	return &v2.ApisixTls{
		ObjectMeta: metav1.ObjectMeta{
			Name:      Name,
			Namespace: constants.Namespace,
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
				Namespace: constants.Namespace,
			},
		},
	}
}

func (a *hub) ingressForHub() client.Object {
	return &v2.ApisixRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rbd-hub",
			Namespace: constants.Namespace,
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

func (a *hub) After() error {
	return nil
}

func (a *hub) ListPods() ([]corev1.Pod, error) {
	return listPods(a.ctx, a.client, a.component.Namespace, a.labels)
}

func (a *hub) SetStorageClassNameRWX(pvcParameters *pvcParameters) {
	a.pvcParametersRWX = pvcParameters
}

func (a *hub) deployment() client.Object {
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

	env = mergeEnvs(env, a.component.Spec.Env)
	volumeMounts = mergeVolumeMounts(volumeMounts, a.component.Spec.VolumeMounts)
	volumes = mergeVolumes(volumes, a.component.Spec.Volumes)

	ds := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      HubName,
			Namespace: a.component.Namespace,
			Labels:    a.labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: a.component.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: a.labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   HubName,
					Labels: a.labels,
				},
				Spec: corev1.PodSpec{
					ImagePullSecrets:              imagePullSecrets(a.component, a.cluster),
					TerminationGracePeriodSeconds: commonutil.Int64(0),
					Containers: []corev1.Container{
						{
							Name:            "rbd-hub",
							Image:           a.component.Spec.Image,
							ImagePullPolicy: a.component.ImagePullPolicy(),
							Env:             env,
							VolumeMounts:    volumeMounts,
							Resources:       a.component.Spec.Resources,
						},
					},
					Volumes: volumes,
				},
			},
		},
	}

	return ds
}

func (a *hub) serviceForHub() client.Object {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      HubName,
			Namespace: a.component.Namespace,
			Labels:    a.labels,
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
			Selector: a.labels,
		},
	}

	return svc
}

func (a *hub) persistentVolumeClaimForHub() *corev1.PersistentVolumeClaim {
	if a.component.Labels["persistentVolumeClaimAccessModes"] == string(corev1.ReadWriteOnce) {
		return createPersistentVolumeClaimRWO(a.component.Namespace, hubDataPvcName, a.pvcParametersRWX, a.labels, a.storageRequest)
	}
	return createPersistentVolumeClaimRWX(a.component.Namespace, hubDataPvcName, a.pvcParametersRWX, a.labels, a.storageRequest)
}

func (a *hub) secretForHub() client.Object {
	secret, err := a.getSecret(hubImageRepository)
	if secret != nil {
		// never update hub secret
		return nil
	}
	if err != nil && !k8sErrors.IsNotFound(err) {
		logrus.Errorf("get secret %s: %v", hubImageRepository, err)
		return nil
	}
	labels := copyLabels(a.labels)
	labels["name"] = hubImageRepository
	_, pem, key, _ := commonutil.DomainSign(nil, rbdutil.GetImageRepository(a.cluster))
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      hubImageRepository,
			Namespace: a.component.Namespace,
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

func (a *hub) passwordSecret() client.Object {
	labels := copyLabels(a.labels)
	labels["name"] = hubPasswordSecret
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      hubPasswordSecret,
			Namespace: a.component.Namespace,
			Labels:    labels,
		},
		Data: map[string][]byte{
			"HTPASSWD": a.htpasswd,
			"password": []byte(a.password),
		},
	}
}

func (a *hub) getSecret(name string) (*corev1.Secret, error) {
	return getSecret(a.ctx, a.client, a.component.Namespace, name)
}

func (a *hub) generateHtpasswd() ([]byte, error) {
	cmd := exec.Command("htpasswd", "-Bbn", a.cluster.Spec.ImageHub.Username, a.cluster.Spec.ImageHub.Password)
	return cmd.CombinedOutput()
}
