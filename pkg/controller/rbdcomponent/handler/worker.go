package handler

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/goodrain/rainbond-operator/pkg/util/commonutil"
	"github.com/goodrain/rainbond-operator/pkg/util/constants"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
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

	pvcParametersRWX *pvcParameters
	storageRequest   int64
}

var _ ComponentHandler = &worker{}
var _ StorageClassRWXer = &worker{}

// NewWorker creates a new rbd-worker hanlder.
func NewWorker(ctx context.Context, client client.Client, component *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster) ComponentHandler {
	return &worker{
		ctx:            ctx,
		client:         client,
		component:      component,
		cluster:        cluster,
		labels:         LabelsForRainbondComponent(component),
		storageRequest: getStorageRequest("GRDATA_STORAGE_REQUEST", 40),
	}
}

func (w *worker) Before() error {
	db, err := getDefaultDBInfo(w.ctx, w.client, w.cluster.Spec.RegionDatabase, w.component.Namespace, DBName)
	if err != nil {
		return fmt.Errorf("get db info: %v", err)
	}
	w.db = db

	secret, err := etcdSecret(w.ctx, w.client, w.cluster)
	if err != nil {
		return fmt.Errorf("failed to get etcd secret: %v", err)
	}
	w.etcdSecret = secret

	if err := setStorageCassName(w.ctx, w.client, w.component.Namespace, w); err != nil {
		return err
	}

	return nil
}

func (w *worker) Resources() []interface{} {
	return []interface{}{
		w.deployment(),
	}
}

func (w *worker) After() error {
	return nil
}

func (w *worker) ListPods() ([]corev1.Pod, error) {
	return listPods(w.ctx, w.client, w.component.Namespace, w.labels)
}

func (w *worker) SetStorageClassNameRWX(pvcParameters *pvcParameters) {
	w.pvcParametersRWX = pvcParameters
}

func (w *worker) ResourcesCreateIfNotExists() []interface{} {
	return []interface{}{
		// pvc is immutable after creation except resources.requests for bound claims
		createPersistentVolumeClaimRWX(w.component.Namespace, constants.GrDataPVC, w.pvcParametersRWX, w.labels),
	}
}

func (w *worker) deployment() interface{} {
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "grdata",
			MountPath: "/grdata",
		},
	}
	volumes := []corev1.Volume{
		{
			Name: "grdata",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: constants.GrDataPVC,
				},
			},
		},
	}
	args := []string{
		fmt.Sprintf("--log-level=%s", w.component.LogLevel()),
		"--host-ip=$(POD_IP)",
		"--node-name=$(HOST_IP)",
		w.db.RegionDataSource(),
		"--etcd-endpoints=" + strings.Join(etcdEndpoints(w.cluster), ","),
	}
	if w.etcdSecret != nil {
		volume, mount := volumeByEtcd(w.etcdSecret)
		volumeMounts = append(volumeMounts, mount)
		volumes = append(volumes, volume)
		args = append(args, etcdSSLArgs()...)
	}

	env := []corev1.EnvVar{
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
			Name:  "BUILD_IMAGE_REPOSTORY_USER",
			Value: "admin",
		},
		{
			Name: "BUILD_IMAGE_REPOSTORY_PASS",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: hubImageRepository,
					},
					Key: "password",
				},
			},
		},
	}
	if imageHub := w.cluster.Spec.ImageHub; imageHub != nil {
		env = append(env, corev1.EnvVar{
			Name:  "BUILD_IMAGE_REPOSTORY_DOMAIN",
			Value: path.Join(imageHub.Domain, imageHub.Namespace),
		})
		// TODO: huangrh
		// env = append(env, corev1.EnvVar{
		// 	Name:  "BUILD_IMAGE_REPOSTORY_USER",
		// 	Value: imageHub.Username,
		// })
		// env = append(env, corev1.EnvVar{
		// 	Name:  "BUILD_IMAGE_REPOSTORY_PASS",
		// 	Value: imageHub.Password,
		// })
	}

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
					ServiceAccountName:            "rainbond-operator",
					ImagePullSecrets:              imagePullSecrets(w.component, w.cluster),
					Containers: []corev1.Container{
						{
							Name:            WorkerName,
							Image:           w.component.Spec.Image,
							ImagePullPolicy: w.component.ImagePullPolicy(),
							Env:             env,
							Args:            args,
							VolumeMounts:    volumeMounts,
							LivenessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{Path: "/worker/health", Port: intstr.FromInt(6369)},
								},
								InitialDelaySeconds: 30,
								PeriodSeconds:       10,
								TimeoutSeconds:      5,
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
