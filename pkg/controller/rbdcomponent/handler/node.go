package handler

import (
	"context"
	"fmt"
	"strings"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/goodrain/rainbond-operator/pkg/util/commonutil"
	"github.com/goodrain/rainbond-operator/pkg/util/constants"
	"github.com/goodrain/rainbond-operator/pkg/util/k8sutil"
	"github.com/goodrain/rainbond-operator/pkg/util/rbdutil"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NodeName name for rbd-node
var NodeName = "rbd-node"

type node struct {
	ctx        context.Context
	client     client.Client
	labels     map[string]string
	etcdSecret *corev1.Secret

	cluster   *rainbondv1alpha1.RainbondCluster
	component *rainbondv1alpha1.RbdComponent

	pvcParametersRWX     *pvcParameters
	grdataStorageRequest int64
}

var _ ComponentHandler = &node{}
var _ StorageClassRWXer = &node{}
var _ K8sResourcesInterface = &node{}

// NewNode creates a new rbd-node handler.
func NewNode(ctx context.Context, client client.Client, component *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster) ComponentHandler {
	return &node{
		ctx:                  ctx,
		client:               client,
		component:            component,
		cluster:              cluster,
		labels:               LabelsForRainbondComponent(component),
		grdataStorageRequest: getStorageRequest("GRDATA_STORAGE_REQUEST", 40),
	}
}

func (n *node) Before() error {
	secret, err := etcdSecret(n.ctx, n.client, n.cluster)
	if err != nil {
		return fmt.Errorf("failed to get etcd secret: %v", err)
	}
	n.etcdSecret = secret

	if err := setStorageCassName(n.ctx, n.client, n.component.Namespace, n); err != nil {
		return err
	}

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

func (n *node) ListPods() ([]corev1.Pod, error) {
	return listPods(n.ctx, n.client, n.component.Namespace, n.labels)
}

func (n *node) SetStorageClassNameRWX(pvcParameters *pvcParameters) {
	n.pvcParametersRWX = pvcParameters
}

func (n *node) ResourcesCreateIfNotExists() []interface{} {
	return []interface{}{
		// pvc is immutable after creation except resources.requests for bound claims
		createPersistentVolumeClaimRWX(n.component.Namespace, constants.GrDataPVC, n.pvcParametersRWX, n.labels),
	}
}

func (n *node) daemonSetForRainbondNode() interface{} {
	volumeMounts := []corev1.VolumeMount{
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
		{
			Name:      "etc",
			MountPath: "/newetc",
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
					Type: k8sutil.HostPath(corev1.HostPathFile),
				},
			},
		},
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
	args := []string{
		fmt.Sprintf("--log-level=%s", n.component.LogLevel()),
		"--etcd=" + strings.Join(etcdEndpoints(n.cluster), ","),
		"--hostIP=$(POD_IP)",
		"--run-mode master",
		"--noderule manage,compute", // TODO: Let rbd-node recognize itself
		"--nodeid=$(NODE_NAME)",
		"--image-repo-host=" + rbdutil.GetImageRepository(n.cluster),
		"--hostsfile=/newetc/hosts",
	}
	if n.etcdSecret != nil {
		volume, mount := volumeByEtcd(n.etcdSecret)
		volumeMounts = append(volumeMounts, mount)
		volumes = append(volumes, volume)
		args = append(args, etcdSSLArgs()...)
	}
	envs := []corev1.EnvVar{
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
			Name:  "RBD_NAMESPACE",
			Value: n.component.Namespace,
		},
	}
	if n.cluster.Spec.ImageHub == nil || n.cluster.Spec.ImageHub.Domain == constants.DefImageRepository {
		envs = append(envs, corev1.EnvVar{
			Name:  "RBD_DOCKER_SECRET",
			Value: hubImageRepository,
		})
	}

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
					TerminationGracePeriodSeconds: commonutil.Int64(0),
					ServiceAccountName:            "rainbond-operator",
					HostAliases:                   hostsAliases(n.cluster),
					HostNetwork:                   true,
					HostPID:                       true,
					DNSPolicy:                     corev1.DNSClusterFirstWithHostNet,
					Tolerations: []corev1.Toleration{
						{
							Operator: corev1.TolerationOpExists, // tolerate everything.
						},
					},
					Containers: []corev1.Container{
						{
							Name:            NodeName,
							Image:           n.component.Spec.Image,
							ImagePullPolicy: n.component.ImagePullPolicy(),
							Env:             envs,
							Args:            args,
							VolumeMounts:    volumeMounts,
						},
					},
					Volumes: volumes,
				},
			},
		},
	}

	return ds
}
