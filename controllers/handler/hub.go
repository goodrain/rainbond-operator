package handler

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/utils/pointer"

	v2 "github.com/goodrain/rainbond-operator/api/v2"
	"github.com/sirupsen/logrus"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"

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

	pvcParametersRWO *pvcParameters
	storageRequest   int64
}

var _ ComponentHandler = &hub{}
var _ StorageClassRWOer = &hub{}

// NewHub nw hub
func NewHub(ctx context.Context, client client.Client, component *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster) ComponentHandler {
	return &hub{
		component:      component,
		cluster:        cluster,
		client:         client,
		ctx:            ctx,
		labels:         LabelsForRainbondComponent(component),
		storageRequest: getStorageRequest("HUB_DATA_STORAGE_REQUEST", 1),
	}
}

func (h *hub) Before() error {
	// Check if using custom image repository (not goodrain.me or goodrain.me:port)
	domain := h.cluster.Spec.ImageHub.Domain
	if domain != constants.DefImageRepository && !strings.HasPrefix(domain, constants.DefImageRepository+":") {
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

	if err := setStorageCassName(h.ctx, h.client, h.component.Namespace, h); err != nil {
		return err
	}

	return nil
}

func (h *hub) SetStorageClassNameRWO(pvcParameters *pvcParameters) {
	h.pvcParametersRWO = pvcParameters
}

func (h *hub) Resources() []client.Object {
	resources := []client.Object{
		h.secretForHub(), // important! create secret before ingress.
		h.passwordSecret(),
		h.deployment(),
		h.serviceForHub(),
		h.hostsJob(),
		h.hubImageRepository(), // 绑定这个镜像仓库的secret
		h.ingressForHub(),      //创建这个域名的路由
		createPersistentVolumeClaimRWO(h.component.Namespace, hubDataPvcName, h.pvcParametersRWO, h.labels, h.storageRequest),
	}

	// Add PVC if using local-path storage
	if h.pvcParametersRWO != nil {
		hubPVC := createPersistentVolumeClaimRWO(h.component.Namespace, hubDataPvcName, h.pvcParametersRWO, h.labels, h.storageRequest)
		resources = append([]client.Object{hubPVC}, resources...)
	}

	return resources
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
							ServicePort:        intstr.FromInt(5000),
							ServiceName:        "rbd-hub",
							ResolveGranularity: "service",
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
			Name:  "REGISTRY_HTTP_SECRET",
			Value: "rainbond-registry-secret",
		},
		{
			Name:  "REGISTRY_STORAGE",
			Value: "filesystem",
		},
		{
			Name:  "REGISTRY_STORAGE_FILESYSTEM_ROOTDIRECTORY",
			Value: "/var/lib/registry",
		},
		{
			Name:  "REGISTRY_STORAGE_REDIRECT_DISABLE",
			Value: "true",
		},
		{
			Name:  "REGISTRY_STORAGE_DELETE_ENABLED",
			Value: "true",
		},
		{
			Name:  "REGISTRY_HTTP_RELATIVEURLS",
			Value: "true",
		},
	}
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "htpasswd",
			MountPath: "/auth",
			ReadOnly:  true,
		},
		{
			Name:      "data",
			MountPath: "/var/lib/registry",
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
		{
			Name: "data",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: hubDataPvcName,
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
					Affinity:                      h.component.Spec.Affinity,
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

func (h *hub) hostsJob() client.Object {
	gatewayNodes := h.cluster.Spec.NodesForGateway
	var gatewayHost string
	if gatewayNodes != nil && len(gatewayNodes) > 0 {
		gatewayHost = gatewayNodes[0].InternalIP
	}
	hostCMD := fmt.Sprintf("grep -qxF '%v goodrain.me' /etc/hosts || echo '%v goodrain.me' >> /etc/hosts;", gatewayHost, gatewayHost)
	nodeList := &corev1.NodeList{}
	if err := h.client.List(context.TODO(), nodeList, &client.ListOptions{}); err != nil {
		logrus.Errorf("get node len failure: %v", err)
		return nil
	}

	// 创建 Job 对象
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "hosts-job",
			Namespace: h.component.Namespace,
		},
		Spec: batchv1.JobSpec{
			TTLSecondsAfterFinished: pointer.Int32Ptr(60),
			Parallelism:             pointer.Int32Ptr(int32(len(nodeList.Items))), // 并行数应等于节点数
			Completions:             pointer.Int32Ptr(int32(len(nodeList.Items))), // 确保每个节点完成一次
			BackoffLimit:            pointer.Int32Ptr(0),                          // 设置重试次数
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: h.labels,
				},
				Spec: corev1.PodSpec{
					Affinity: &corev1.Affinity{
						NodeAffinity: &corev1.NodeAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
								NodeSelectorTerms: []corev1.NodeSelectorTerm{
									{
										MatchExpressions: []corev1.NodeSelectorRequirement{
											{
												Key:      "kubernetes.io/hostname",
												Operator: corev1.NodeSelectorOpExists, // 确保在所有节点上运行
											},
										},
									},
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:            "hosts-job",
							Image:           os.Getenv("RAINBOND_IMAGE_REPOSITORY") + "/alpine:latest",
							ImagePullPolicy: corev1.PullIfNotPresent,
							Command: []string{
								"/bin/sh", "-c", hostCMD,
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "hosts",
									MountPath: "/etc/hosts",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "hosts",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/etc/hosts",
								},
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyOnFailure,
				},
			},
		},
	}
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
			Selector:        h.labels,
			SessionAffinity: corev1.ServiceAffinityClientIP,
			SessionAffinityConfig: &corev1.SessionAffinityConfig{
				ClientIP: &corev1.ClientIPConfig{
					TimeoutSeconds: commonutil.Int32(600),
				},
			},
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
