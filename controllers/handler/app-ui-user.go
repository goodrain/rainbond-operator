package handler

import (
	"context"
	"fmt"
	"github.com/goodrain/rainbond-operator/util/k8sutil"
	"os"
	"strconv"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
	"github.com/goodrain/rainbond-operator/util/commonutil"
	"github.com/goodrain/rainbond-operator/util/probeutil"
	"github.com/goodrain/rainbond-operator/util/rbdutil"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// AppUINameUSER name for rbd-app-ui resources.
var AppUINameUSER = "rbd-app-ui-user"

type appuiuser struct {
	ctx       context.Context
	client    client.Client
	labels    map[string]string
	db        *rainbondv1alpha1.Database
	component *rainbondv1alpha1.RbdComponent
	cluster   *rainbondv1alpha1.RainbondCluster

	pvcParametersRWX *pvcParameters
	pvcName          string
	storageRequest   int64
}

var _ ComponentHandler = &appuiuser{}
var _ StorageClassRWXer = &appuiuser{}

// NewAppUI creates a new rbd-app-ui handler.
func NewAppUIUser(ctx context.Context, client client.Client, component *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster) ComponentHandler {
	return &appuiuser{
		ctx:            ctx,
		client:         client,
		component:      component,
		cluster:        cluster,
		labels:         LabelsForRainbondComponent(component),
		pvcName:        "rbd-app-ui",
		storageRequest: getStorageRequest("APP_UI_DATA_STORAGE_REQUEST", 10),
	}
}

func (a *appuiuser) Before() error {
	db, err := getDefaultDBInfo(a.ctx, a.client, a.cluster.Spec.UIDatabase, a.component.Namespace, DBName)
	if err != nil {
		return fmt.Errorf("get db info: %v", err)
	}
	if db.Name == "" {
		db.Name = ConsoleDatabaseName
	}
	a.db = db

	if err := setStorageCassName(a.ctx, a.client, a.component.Namespace, a); err != nil {
		return err
	}

	if err := isUIDBReady(a.ctx, a.client, a.component, a.cluster); err != nil {
		return err
	}

	if a.cluster.Spec.ImageHub == nil {
		return NewIgnoreError("image repository not ready")
	}

	return nil
}

func (a *appuiuser) Resources() []client.Object {
	res := []client.Object{
		a.serviceForAppUI(int32(7071)),
		a.serviceForProxy(),
	}

	// 获取所有的node name
	names, err := getNodeNames(a.client)

	if err == nil {
		for i, name := range names {
			res = append(res, a.nodeJob(affinityForRequiredNodes([]string{name}), i))
		}
	} else {
		log.Error(err, "get node names step")
	}

	if err := isUIDBMigrateOK(a.ctx, a.client, a.component); err != nil {
		if IsIgnoreError(err) {
			log.V(6).Info(fmt.Sprintf("check if ui db migrations is ok: %v", err))
		} else {
			log.Error(err, "check if ui db migrations is ok")
		}
	} else {
		res = append(res, a.deploymentForAppUI())
	}

	return res
}

func (a *appuiuser) After() error {
	return nil
}

func (a *appuiuser) ListPods() ([]corev1.Pod, error) {
	return listPods(a.ctx, a.client, a.component.Namespace, a.labels)
}

func (a *appuiuser) SetStorageClassNameRWX(pvcParameters *pvcParameters) {
	a.pvcParametersRWX = pvcParameters
}

func (a *appuiuser) ResourcesCreateIfNotExists() []client.Object {
	return []client.Object{
		// pvc is immutable after creation except resources.requests for bound claims
		createPersistentVolumeClaimRWX(a.component.Namespace, a.pvcName, a.pvcParametersRWX, a.labels, a.storageRequest),
	}
}

func (a *appuiuser) deploymentForAppUI() client.Object {
	cpt := a.component

	envs := []corev1.EnvVar{
		{
			Name:  "CRYPTOGRAPHY_ALLOW_OPENSSL_102",
			Value: "true",
		},
		{
			Name:  "MYSQL_HOST",
			Value: a.db.Host,
		},
		{
			Name:  "PLATFORM_ROLE",
			Value: "user",
		},
		{
			Name:  "MYSQL_PORT",
			Value: strconv.Itoa(a.db.Port),
		},
		{
			Name:  "MYSQL_USER",
			Value: a.db.Username,
		},
		{
			Name:  "MYSQL_PASS",
			Value: a.db.Password,
		},
		{
			Name:  "MYSQL_DB",
			Value: a.db.Name,
		},
		{
			Name:  "REGION_URL",
			Value: fmt.Sprintf("https://rbd-api-api:%s", rbdutil.GetenvDefault("API_PORT", "8443")),
		},
		{
			Name:  "REGION_WS_URL",
			Value: fmt.Sprintf("ws://%s:%s", a.cluster.GatewayIngressIP(), rbdutil.GetenvDefault("API_WS_PORT", "6060")),
		},
		{
			Name:  "REGION_HTTP_DOMAIN",
			Value: a.cluster.Spec.SuffixHTTPHost,
		},
		{
			Name:  "REGION_TCP_DOMAIN",
			Value: a.cluster.GatewayIngressIP(),
		},
		{
			Name:  "IMAGE_REPO",
			Value: a.cluster.Spec.ImageHub.Domain,
		},
	}
	volumes := []corev1.Volume{
		{
			Name: "ssl",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: apiClientSecretName,
				},
			},
		},
		{
			Name: "app",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: a.pvcName,
				},
			},
		},
	}
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "app",
			MountPath: "/app/data",
			SubPath:   "data",
		},
		{
			Name:      "ssl",
			MountPath: "/app/region/ssl",
		},
		{
			Name:      "app",
			MountPath: "/app/logs/",
			SubPath:   "logs",
		},
		{
			Name:      "app",
			MountPath: "/app/lock",
			SubPath:   "lock",
		},
	}

	envs = mergeEnvs(envs, a.component.Spec.Env)
	volumeMounts = mergeVolumeMounts(volumeMounts, a.component.Spec.VolumeMounts)
	volumes = mergeVolumes(volumes, a.component.Spec.Volumes)

	// prepare probe
	readinessProbe := probeutil.MakeReadinessProbeTCP("", 7070)
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      AppUINameUSER,
			Namespace: cpt.Namespace,
			Labels:    a.labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: a.component.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: a.labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   AppUINameUSER,
					Labels: a.labels,
				},
				Spec: corev1.PodSpec{
					ImagePullSecrets:              imagePullSecrets(a.component, a.cluster),
					TerminationGracePeriodSeconds: commonutil.Int64(0),
					Containers: []corev1.Container{
						{
							Name:            AppUINameUSER,
							Image:           cpt.Spec.Image,
							ImagePullPolicy: cpt.ImagePullPolicy(),
							Env:             envs,
							VolumeMounts:    volumeMounts,
							ReadinessProbe:  readinessProbe,
							Resources:       a.component.Spec.Resources,
						},
						{
							Name:  "proxy-pass",
							Image: "registry.cn-hangzhou.aliyuncs.com/goodrain/proxy-pass",
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 80,
								},
							},
						},
					},
					Volumes: volumes,
				},
			},
		},
	}

	if a.cluster.Annotations != nil {
		if enterpriseID, ok := a.cluster.Annotations["enterprise_id"]; ok {
			deploy.Spec.Template.Spec.Containers[0].Env = append(deploy.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{Name: "ENTERPRISE_ID", Value: enterpriseID})
		}
		if os.Getenv("ENTERPRISE_ID") != "" {
			deploy.Spec.Template.Spec.Containers[0].Env = append(deploy.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{Name: "ENTERPRISE_ID", Value: os.Getenv("ENTERPRISE_ID")})
		}
	}

	return deploy
}

func (a *appuiuser) serviceForAppUI(port int32) client.Object {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      AppUINameUSER,
			Namespace: a.component.Namespace,
			Labels:    a.labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name: "http",
					Port: port,
					TargetPort: intstr.IntOrString{
						IntVal: 7070,
					},
				},
			},
			Selector: a.labels,
		},
	}

	return svc
}

// serviceForProxy 将容器内的80暴露到6060端口
func (a *appuiuser) serviceForProxy() client.Object {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rbd-app-ui-proxy-user",
			Namespace: a.component.Namespace,
			Labels:    a.labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name: "proxy",
					Port: 6060,
					TargetPort: intstr.IntOrString{
						IntVal: 80,
					},
				},
			},
			Selector: a.labels,
		},
	}
	return svc
}

// 需要每一个节点都运行一遍
func (a *appuiuser) nodeJob(aff *corev1.Affinity, index int) *batchv1.Job {

	name := "node-job-" + strconv.Itoa(index)
	labels := map[string]string{
		"name": name,
	}

	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "etc",
			MountPath: "/newetc",
		},
	}
	volumes := []corev1.Volume{
		{
			Name: "etc",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/etc",
					Type: k8sutil.HostPath(corev1.HostPathDirectory),
				},
			},
		},
	}

	runtimeVolumes, runtimeVolumeMounts := a.getDockerVolumes()

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "rbd-system",
			Labels:    labels,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					SecurityContext: &corev1.PodSecurityContext{
						RunAsUser: commonutil.Int64(0),
					},
					Affinity:           aff,
					ServiceAccountName: "rainbond-operator",
					Volumes:            append(volumes, runtimeVolumes...),
					Containers: []corev1.Container{
						{
							Name:         name,
							Image:        "registry.cn-hangzhou.aliyuncs.com/goodrain/node-job",
							VolumeMounts: append(volumeMounts, runtimeVolumeMounts...),
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
		},
	}
	return job
}

func (a *appuiuser) getDockerVolumes() ([]corev1.Volume, []corev1.VolumeMount) {
	volumes := []corev1.Volume{
		{
			Name: "docker",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/var/lib/docker",
					Type: k8sutil.HostPath(corev1.HostPathDirectoryOrCreate),
				},
			},
		},
		{
			Name: "vardocker",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/var/docker/lib",
					Type: k8sutil.HostPath(corev1.HostPathDirectoryOrCreate),
				},
			},
		},
		{
			Name: "dockercert",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/etc/docker/certs.d",
					Type: k8sutil.HostPath(corev1.HostPathDirectoryOrCreate),
				},
			},
		},
		{
			Name: "dockersock",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/var/run/docker.sock",
					Type: k8sutil.HostPath(corev1.HostPathSocket),
				},
			},
		},
	}
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "dockersock",
			MountPath: "/var/run/docker.sock",
		},
		{
			Name:      "docker", // for container logs, ubuntu
			MountPath: "/var/lib/docker",
		},
		{
			Name:      "vardocker", // for container logs, centos
			MountPath: "/var/docker/lib",
		},
		{
			Name:      "dockercert",
			MountPath: "/etc/docker/certs.d",
		},
	}
	return volumes, volumeMounts
}