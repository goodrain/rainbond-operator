package handler

import (
	"context"
	"fmt"
	"github.com/GLYASAI/rainbond-operator/pkg/util/constants"

	rainbondv1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/GLYASAI/rainbond-operator/pkg/util/k8sutil"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var NodeName = "rbd-node"

type node struct {
	component *rainbondv1alpha1.RbdComponent
	cluster   *rainbondv1alpha1.RainbondCluster
	labels    map[string]string
}

func NewNode(ctx context.Context, client client.Client, component *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster) ComponentHandler {
	return &node{
		component: component,
		cluster:   cluster,
		labels:    component.Labels(),
	}
}

func (n *node) Before() error {
	// TODO: check prerequisites
	return nil
}

func (n *node) Resources() []interface{} {
	return []interface{}{
		n.daemonSetForRainbondNode(),
	}
}

func (n *node) After() error {
	return nil
}

func (n *node) daemonSetForRainbondNode() interface{} {
	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      NodeName,
			Namespace: n.component.Namespace,
			Labels:    n.labels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: n.labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   NodeName,
					Labels: n.labels,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "rainbond-operator",
					HostNetwork:        true,
					HostPID:            true,
					DNSPolicy:          corev1.DNSClusterFirstWithHostNet,
					Tolerations: []corev1.Toleration{
						{
							Key:    "node-role.kubernetes.io/master",
							Effect: corev1.TaintEffectNoSchedule,
						},
					},
					Containers: []corev1.Container{
						{
							Name:            NodeName,
							Image:           n.component.Spec.Image,
							ImagePullPolicy: n.component.ImagePullPolicy(),
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
									Name: "NODE_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "spec.nodeName",
										},
									},
								},
								{
									Name:  "RBD_DOCKER_SECRET",
									Value: hubImageRepository,
								},
							},
							Args: []string{
								fmt.Sprintf("--log-level=%s", n.component.LogLevel()),
								"--etcd=http://etcd0:2379", // TODO: do not hard code
								"--hostIP=$(POD_IP)",
								"--run-mode master",
								"--noderule manage,compute", // TODO: Let rbd-node recognize itself
								"--nodeid=$(NODE_NAME)",
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "grdata",
									MountPath: "/grdata",
								},
								{
									Name:      "proc",
									MountPath: "/proc",
								},
								{
									Name:      "sys",
									MountPath: "/sys",
								},
								{
									Name:      "dockersock",
									MountPath: "/var/run/docker.sock",
								},
								{
									Name:      "docker", // for container logs
									MountPath: "/var/lib/docker",
								}, {
									Name:      "dockercert", // for container logs
									MountPath: "/etc/docker/certs.d",
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
							Name: "proc",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/proc",
									Type: k8sutil.HostPath(corev1.HostPathDirectory),
								},
							},
						},
						{
							Name: "sys",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/sys",
									Type: k8sutil.HostPath(corev1.HostPathDirectory),
								},
							},
						},
						{
							Name: "docker",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/lib/docker",
									Type: k8sutil.HostPath(corev1.HostPathDirectory),
								},
							},
						}, {
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
									Type: k8sutil.HostPath(corev1.HostPathFile),
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
