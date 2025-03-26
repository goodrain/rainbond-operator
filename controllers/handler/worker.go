package handler

import (
	"context"
	"fmt"
	"path"

	checksqllite "github.com/goodrain/rainbond-operator/util/check-sqllite"
	"github.com/goodrain/rainbond-operator/util/rbdutil"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/goodrain/rainbond-operator/util/probeutil"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
	"github.com/goodrain/rainbond-operator/util/commonutil"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// WorkerName name for rbd-worker.
var WorkerName = "rbd-worker"

type worker struct {
	ctx        context.Context
	client     client.Client
	component  *rainbondv1alpha1.RbdComponent
	cluster    *rainbondv1alpha1.RainbondCluster
	labels     map[string]string
	db         *rainbondv1alpha1.Database
	etcdSecret *corev1.Secret

	storageRequest int64
}

var _ ComponentHandler = &worker{}

// NewWorker creates a new rbd-worker hanlder.
func NewWorker(ctx context.Context, client client.Client, component *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster) ComponentHandler {
	return &worker{
		ctx:            ctx,
		client:         client,
		component:      component,
		cluster:        cluster,
		labels:         LabelsForRainbondComponent(component),
		storageRequest: getStorageRequest("GRDATA_STORAGE_REQUEST", 20),
	}
}

func (w *worker) Before() error {
	if !checksqllite.IsSQLLite() {
		db, err := getDefaultDBInfo(w.ctx, w.client, w.cluster.Spec.RegionDatabase, w.component.Namespace, DBName)
		if err != nil {
			return fmt.Errorf("get db info: %v", err)
		}
		if db.Name == "" {
			db.Name = RegionDatabaseName
		}
		w.db = db
	}

	secret, err := etcdSecret(w.ctx, w.client, w.cluster)
	if err != nil {
		return fmt.Errorf("failed to get etcd secret: %v", err)
	}
	w.etcdSecret = secret
	return nil
}

func (w *worker) Resources() []client.Object {
	return []client.Object{
		w.deployment(),
		w.service(),
	}
}

func (w *worker) After() error {
	return nil
}

func (w *worker) ListPods() ([]corev1.Pod, error) {
	return listPods(w.ctx, w.client, w.component.Namespace, w.labels)
}

func (w *worker) ResourcesCreateIfNotExists() []client.Object {
	return []client.Object{}
}

func (w *worker) deployment() client.Object {
	var volumeMounts []corev1.VolumeMount
	var volumes []corev1.Volume
	args := []string{
		"--hostIP=$(POD_IP)",
		"--node-name=$(POD_IP)",
		"--rbd-namespace=" + w.component.Namespace,
	}
	if !checksqllite.IsSQLLite() {
		args = append(args, w.db.RegionDataSource())
	}
	if w.etcdSecret != nil {
		volume, mount := volumeByEtcd(w.etcdSecret)
		volumeMounts = append(volumeMounts, mount)
		volumes = append(volumes, volume)
		args = append(args, etcdSSLArgs()...)
	}

	env := []corev1.EnvVar{
		{
			Name:  "RBD_NAMESPACE",
			Value: w.component.Namespace,
		},
		{
			Name: "POD_IP",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "status.podIP",
				},
			},
		},
		{
			Name: "HOST_IP",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "status.hostIP",
				},
			},
		},
		{
			Name: "IMAGE_PULL_SECRET",
			Value: func() string {
				if w.cluster.Status.ImagePullSecret != nil {
					return w.cluster.Status.ImagePullSecret.Name
				}
				return ""
			}(),
		},
		{
			Name:  "SERVICE_ID",
			Value: "rbd-worker",
		},
		{
			Name:  "LOGGER_DRIVER_NAME",
			Value: "streamlog",
		},
	}
	if imageHub := w.cluster.Spec.ImageHub; imageHub != nil {
		env = append(env, corev1.EnvVar{
			Name:  "BUILD_IMAGE_REPOSTORY_DOMAIN",
			Value: path.Join(imageHub.Domain, imageHub.Namespace),
		})
		env = append(env, corev1.EnvVar{
			Name:  "BUILD_IMAGE_REPOSTORY_USER",
			Value: imageHub.Username,
		})
		env = append(env, corev1.EnvVar{
			Name:  "BUILD_IMAGE_REPOSTORY_PASS",
			Value: imageHub.Password,
		})
	}

	args = mergeArgs(args, w.component.Spec.Args)
	env = mergeEnvs(env, w.component.Spec.Env)
	volumeMounts = mergeVolumeMounts(volumeMounts, w.component.Spec.VolumeMounts)
	volumes = mergeVolumes(volumes, w.component.Spec.Volumes)

	// prepare probe
	readinessProbe := probeutil.MakeReadinessProbeHTTP("", "/worker/health", 6369)
	ds := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      WorkerName,
			Namespace: w.component.Namespace,
			Labels:    w.labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: w.component.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: w.labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   WorkerName,
					Labels: w.labels,
				},
				Spec: corev1.PodSpec{
					TerminationGracePeriodSeconds: commonutil.Int64(0),
					ServiceAccountName:            rbdutil.GetenvDefault("SERVICE_ACCOUNT_NAME", "rainbond-operator"),
					ImagePullSecrets:              imagePullSecrets(w.component, w.cluster),
					Containers: []corev1.Container{
						{
							Name:            WorkerName,
							Image:           w.component.Spec.Image,
							ImagePullPolicy: w.component.ImagePullPolicy(),
							Env:             env,
							Args:            args,
							VolumeMounts:    volumeMounts,
							ReadinessProbe:  readinessProbe,
							Resources:       w.component.Spec.Resources,
							LivenessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/worker/health",
										Port: intstr.FromInt(6369),
									},
								},
								InitialDelaySeconds: 30,
								TimeoutSeconds:      2,
								PeriodSeconds:       10,
								SuccessThreshold:    1,
								FailureThreshold:    3,
							},
						},
					},
					Volumes: volumes,
				},
			},
		},
	}

	return ds
}

func (w *worker) service() client.Object {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      WorkerName,
			Namespace: w.component.Namespace,
			Labels:    w.labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       WorkerName,
					Port:       6535,
					TargetPort: intstr.FromInt(6535),
				},
				{
					Name:       WorkerName + "metrics",
					Port:       6369,
					TargetPort: intstr.FromInt(6369),
				},
			},
			Selector: w.labels,
		},
	}
}
