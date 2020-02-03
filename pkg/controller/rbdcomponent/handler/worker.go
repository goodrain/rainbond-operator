package handler

import (
	"context"
	"fmt"
	"github.com/GLYASAI/rainbond-operator/pkg/util/commonutil"
	"github.com/GLYASAI/rainbond-operator/pkg/util/constants"
	"strings"

	rainbondv1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var WorkerName = "rbd-worker"

type worker struct {
	ctx        context.Context
	client     client.Client
	component  *rainbondv1alpha1.RbdComponent
	cluster    *rainbondv1alpha1.RainbondCluster
	labels     map[string]string
	db         *rainbondv1alpha1.Database
	etcdSecret *corev1.Secret
}

func NewWorker(ctx context.Context, client client.Client, component *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster) ComponentHandler {
	return &worker{
		ctx:       ctx,
		client:    client,
		component: component,
		cluster:   cluster,
		labels:    component.GetLabels(),
	}
}

func (w *worker) Before() error {
	w.db = getDefaultDBInfo(w.cluster.Spec.RegionDatabase)

	secret, err := etcdSecret(w.ctx, w.client, w.cluster)
	if err != nil {
		return fmt.Errorf("failed to get etcd secret: %v", err)
	}
	w.etcdSecret = secret

	return isPhaseOK(w.cluster)
}

func (w *worker) Resources() []interface{} {
	return []interface{}{
		w.daemonSetForWorker(),
	}
}

func (w *worker) After() error {
	return nil
}

func (w *worker) daemonSetForWorker() interface{} {
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

	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      WorkerName,
			Namespace: w.component.Namespace,
			Labels:    w.labels,
		},
		Spec: appsv1.DaemonSetSpec{
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
					Tolerations: []corev1.Toleration{
						{
							Key:    w.cluster.Status.MasterRoleLabel,
							Effect: corev1.TaintEffectNoSchedule,
						},
					},
					NodeSelector: w.cluster.Status.MasterNodeLabel(),
					Containers: []corev1.Container{
						{
							Name:            WorkerName,
							Image:           w.component.Spec.Image,
							ImagePullPolicy: w.component.ImagePullPolicy(),
							Env: []corev1.EnvVar{
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
							},
							Args:         args,
							VolumeMounts: volumeMounts,
						},
					},
					Volumes: volumes,
				},
			},
		},
	}

	return ds
}
