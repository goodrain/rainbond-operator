package handler

import (
	"context"
	"fmt"

	rainbondv1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/GLYASAI/rainbond-operator/pkg/util/constants"
	"github.com/GLYASAI/rainbond-operator/pkg/util/k8sutil"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var ChaosName = "rbd-chaos"

type chaos struct {
	component *rainbondv1alpha1.RbdComponent
	cluster   *rainbondv1alpha1.RainbondCluster
	labels    map[string]string
	db        *rainbondv1alpha1.Database
}

func NewChaos(ctx context.Context, client client.Client, component *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster) ComponentHandler {
	return &chaos{
		component: component,
		cluster:   cluster,
		labels:    component.Labels(),
	}
}

func (c *chaos) Before() error {
	c.db = getDefaultDBInfo(c.cluster.Spec.UIDatabase)

	return isPhaseOK(c.cluster)
}

func (c *chaos) Resources() []interface{} {
	return []interface{}{
		c.daemonSetForChaos(),
	}
}

func (c *chaos) After() error {
	return nil
}

func (c *chaos) daemonSetForChaos() interface{} {
	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ChaosName,
			Namespace: c.component.Namespace,
			Labels:    c.labels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: c.labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   ChaosName,
					Labels: c.labels,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "rainbond-operator",
					Tolerations: []corev1.Toleration{
						{
							Key:    "node-role.kubernetes.io/master",
							Effect: corev1.TaintEffectNoSchedule,
						},
					},
					NodeSelector: map[string]string{
						"node-role.kubernetes.io/master": "",
					},
					Containers: []corev1.Container{
						{
							Name:            ChaosName,
							Image:           c.component.Spec.Image,
							ImagePullPolicy: c.component.ImagePullPolicy(),
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
									Name:  "SOURCE_DIR",
									Value: "/cache/source",
								},
								{
									Name:  "CACHE_DIR",
									Value: "/cache",
								},
							},
							Args: []string{
								"--etcd-endpoints=http://etcd0:2379", // TODO: DO NOT HARD CODE
								"--hostIP=$(POD_IP)",
								fmt.Sprintf("--log-level=%s", c.component.LogLevel()),
								c.cluster.RegionDataSource(),
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "grdata",
									MountPath: "/grdata",
								},
								{
									Name:      "dockersock",
									MountPath: "/var/run/docker.sock",
								}, {
									Name:      "cache",
									MountPath: "/cache",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "grdata",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: constants.GrDataPVC,
								},
							},
						},
						{
							Name: "dockersock",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/run/docker.sock",
									Type: k8sutil.HostPath(corev1.HostPathFile),
								},
							},
						},
						{
							Name: "cache",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/cache",
									Type: k8sutil.HostPath(corev1.HostPathDirectoryOrCreate),
								},
							},
						},
					},
				},
			},
		},
	}

	return ds
}
