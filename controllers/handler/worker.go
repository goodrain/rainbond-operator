package handler

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/wutong-paas/wutong-operator/util/probeutil"

	"github.com/wutong-paas/wutong-operator/util/commonutil"
	"github.com/wutong-paas/wutong-operator/util/constants"

	wutongv1alpha1 "github.com/wutong-paas/wutong-operator/api/v1alpha1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// WorkerName name for wt-worker.
var WorkerName = "wt-worker"

type worker struct {
	ctx        context.Context
	client     client.Client
	component  *wutongv1alpha1.WutongComponent
	cluster    *wutongv1alpha1.WutongCluster
	labels     map[string]string
	db         *wutongv1alpha1.Database
	etcdSecret *corev1.Secret

	pvcParametersRWX *pvcParameters
	storageRequest   int64
}

var _ ComponentHandler = &worker{}
var _ StorageClassRWXer = &worker{}

// NewWorker creates a new wt-worker hanlder.
func NewWorker(ctx context.Context, client client.Client, component *wutongv1alpha1.WutongComponent, cluster *wutongv1alpha1.WutongCluster) ComponentHandler {
	return &worker{
		ctx:            ctx,
		client:         client,
		component:      component,
		cluster:        cluster,
		labels:         LabelsForWutongComponent(component),
		storageRequest: getStorageRequest("GRDATA_STORAGE_REQUEST", 40),
	}
}

func (w *worker) Before() error {
	db, err := getDefaultDBInfo(w.ctx, w.client, w.cluster.Spec.RegionDatabase, w.component.Namespace, DBName)
	if err != nil {
		return fmt.Errorf("get db info: %v", err)
	}
	if db.Name == "" {
		db.Name = RegionDatabaseName
	}
	w.db = db

	secret, err := etcdSecret(w.ctx, w.client, w.cluster)
	if err != nil {
		return fmt.Errorf("failed to get etcd secret: %v", err)
	}
	w.etcdSecret = secret

	if w.component.Labels["persistentVolumeClaimAccessModes"] == string(corev1.ReadWriteOnce) {
		sc, err := storageClassNameFromWutongVolumeRWO(w.ctx, w.client, w.component.Namespace)
		if err != nil {
			return err
		}
		w.SetStorageClassNameRWX(sc)
		return nil
	}

	return setStorageCassName(w.ctx, w.client, w.component.Namespace, w)
}

func (w *worker) Resources() []client.Object {
	return []client.Object{
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

func (w *worker) ResourcesCreateIfNotExists() []client.Object {
	if w.component.Labels["persistentVolumeClaimAccessModes"] == string(corev1.ReadWriteOnce) {
		return []client.Object{
			createPersistentVolumeClaimRWO(w.component.Namespace, constants.GrDataPVC, w.pvcParametersRWX, w.labels, w.storageRequest),
		}
	}
	return []client.Object{
		// pvc is immutable after creation except resources.requests for bound claims
		createPersistentVolumeClaimRWX(w.component.Namespace, constants.GrDataPVC, w.pvcParametersRWX, w.labels),
	}
}

func (w *worker) deployment() client.Object {
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
		"--host-ip=$(POD_IP)",
		"--node-name=$(POD_IP)",
		w.db.RegionDataSource(),
		"--etcd-endpoints=" + strings.Join(etcdEndpoints(w.cluster), ","),
		"--wt-system-namespace=" + w.component.Namespace,
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
			Name: "IMAGE_PULL_SECRET",
			Value: func() string {
				if w.cluster.Status.ImagePullSecret != nil {
					return w.cluster.Status.ImagePullSecret.Name
				}
				return ""
			}(),
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
					ServiceAccountName:            "wutong-operator",
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
						},
					},
					Volumes: volumes,
				},
			},
		},
	}

	return ds
}
