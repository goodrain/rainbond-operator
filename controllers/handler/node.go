package handler

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	wutongv1alpha1 "github.com/wutong-paas/wutong-operator/api/v1alpha1"
	"github.com/wutong-paas/wutong-operator/util/commonutil"
	"github.com/wutong-paas/wutong-operator/util/constants"
	"github.com/wutong-paas/wutong-operator/util/containerutil"
	"github.com/wutong-paas/wutong-operator/util/k8sutil"
	"github.com/wutong-paas/wutong-operator/util/probeutil"
	"github.com/wutong-paas/wutong-operator/util/wtutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NodeName name for wt-node
var NodeName = "wt-node"

type node struct {
	ctx    context.Context
	client client.Client
	log    logr.Logger

	labels     map[string]string
	etcdSecret *corev1.Secret
	cluster    *wutongv1alpha1.WutongCluster
	component  *wutongv1alpha1.WutongComponent

	pvcParametersRWX     *pvcParameters
	wtdataStorageRequest int64
	containerRuntime     string
}

var _ ComponentHandler = &node{}
var _ StorageClassRWXer = &node{}
var _ ResourcesCreator = &node{}
var _ Replicaser = &node{}

// NewNode creates a new wt-node handler.
func NewNode(ctx context.Context, client client.Client, component *wutongv1alpha1.WutongComponent, cluster *wutongv1alpha1.WutongCluster) ComponentHandler {
	return &node{
		ctx:                  ctx,
		client:               client,
		log:                  log.WithValues("Name: %s", component.Name),
		component:            component,
		cluster:              cluster,
		labels:               LabelsForWutongComponent(component),
		wtdataStorageRequest: getStorageRequest("WTDATA_STORAGE_REQUEST", 40),
		containerRuntime:     containerutil.GetContainerRuntime(),
	}
}

func (n *node) Before() error {
	secret, err := etcdSecret(n.ctx, n.client, n.cluster)
	if err != nil {
		return fmt.Errorf("failed to get etcd secret: %v", err)
	}
	n.etcdSecret = secret

	if n.component.Labels["persistentVolumeClaimAccessModes"] == string(corev1.ReadWriteOnce) {
		sc, err := storageClassNameFromWutongVolumeRWO(n.ctx, n.client, n.component.Namespace)
		if err != nil {
			return err
		}
		n.SetStorageClassNameRWX(sc)
		return nil
	}
	return setStorageCassName(n.ctx, n.client, n.component.Namespace, n)
}

func (n *node) Resources() []client.Object {
	return []client.Object{
		n.daemonSetForWutongNode(),
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

func (n *node) ResourcesCreateIfNotExists() []client.Object {
	if n.component.Labels["persistentVolumeClaimAccessModes"] == string(corev1.ReadWriteOnce) {
		return []client.Object{
			createPersistentVolumeClaimRWO(n.component.Namespace, constants.WTDataPVC, n.pvcParametersRWX, n.labels, n.wtdataStorageRequest),
		}
	}
	return []client.Object{
		// pvc is immutable after creation except resources.requests for bound claims
		createPersistentVolumeClaimRWX(n.component.Namespace, constants.WTDataPVC, n.pvcParametersRWX, n.labels, n.wtdataStorageRequest),
	}
}

func (n *node) Replicas() *int32 {
	nodeList := &corev1.NodeList{}
	if err := n.client.List(n.ctx, nodeList); err != nil {
		n.log.V(6).Info(fmt.Sprintf("list nodes: %v", err))
		return nil
	}
	return commonutil.Int32(int32(len(nodeList.Items)))
}

func (n *node) getDockerVolumes() ([]corev1.Volume, []corev1.VolumeMount) {
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

func (n *node) getContainerdVolumes() ([]corev1.Volume, []corev1.VolumeMount) {
	volumes := []corev1.Volume{
		{
			Name: "containerdsock",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/run/containerd/containerd.sock",
					Type: k8sutil.HostPath(corev1.HostPathSocket),
				},
			},
		},
		{
			Name: "varlog",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/var/log", // for container logs
					Type: k8sutil.HostPath(corev1.HostPathDirectoryOrCreate),
				},
			},
		},
	}
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "containerdsock", // default using containerd
			MountPath: "/run/containerd/containerd.sock",
		},
		{
			Name:      "varlog",
			MountPath: "/var/log",
		},
	}
	return volumes, volumeMounts
}

func (n *node) daemonSetForWutongNode() client.Object {
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "wtdata",
			MountPath: "/wtdata",
		},
		{
			Name:      "sys",
			MountPath: "/sys",
		},
		{
			Name:      "etc",
			MountPath: "/newetc",
		},
		{
			Name:      "wtlocaldata",
			MountPath: "/wtlocaldata",
		},
	}
	volumes := []corev1.Volume{
		{
			Name: "wtdata",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: constants.WTDataPVC,
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
			Name: "etc",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/etc",
					Type: k8sutil.HostPath(corev1.HostPathDirectory),
				},
			},
		},
		{
			Name: "wtlocaldata",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/wtlocaldata",
					Type: k8sutil.HostPathDirectoryOrCreate(),
				},
			},
		},
	}

	args := []string{
		"--etcd=" + strings.Join(etcdEndpoints(n.cluster), ","),
		"--hostIP=$(POD_IP)",
		"--run-mode master",
		"--noderule manage,compute", // TODO: Let wt-node recognize itself
		"--nodeid=$(NODE_NAME)",
		"--image-repo-host=" + wtutil.GetImageRepository(n.cluster),
		"--hostsfile=/newetc/hosts",
		"--wt-ns=" + n.component.Namespace,
	}
	var (
		runtimeVolumes      []corev1.Volume
		runtimeVolumeMounts []corev1.VolumeMount
	)
	runtimeVolumes, runtimeVolumeMounts = n.getContainerdVolumes()
	if n.containerRuntime == containerutil.ContainerRuntimeDocker {
		runtimeVolumes, runtimeVolumeMounts = n.getDockerVolumes()
		args = append(args, "--container-runtime=docker")
	}
	volumes = append(volumes, runtimeVolumes...)
	volumeMounts = append(volumeMounts, runtimeVolumeMounts...)

	if n.etcdSecret != nil {
		volume, mount := volumeByEtcd(n.etcdSecret)
		volumeMounts = append(volumeMounts, mount)
		volumes = append(volumes, volume)
		args = append(args, etcdSSLArgs()...)
	}
	volumeMounts = mergeVolumeMounts(volumeMounts, n.component.Spec.VolumeMounts)
	volumes = mergeVolumes(volumes, n.component.Spec.Volumes)

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
			Name:  "WT_NAMESPACE",
			Value: n.component.Namespace,
		},
	}
	if n.cluster.Spec.ImageHub == nil || n.cluster.Spec.ImageHub.Domain == constants.DefImageRepository {
		envs = append(envs, corev1.EnvVar{
			Name:  "WT_DOCKER_SECRET",
			Value: hubImageRepository,
		})
	}
	envs = mergeEnvs(envs, n.component.Spec.Env)

	// prepare probe
	readinessProbe := probeutil.MakeReadinessProbeHTTP("", "/v2/ping", 6100)
	args = mergeArgs(args, n.component.Spec.Args)
	tolerations := []corev1.Toleration{
		{
			Operator: corev1.TolerationOpExists, // tolerate everything.
		},
	}

	if n.component.Spec.Tolerations != nil && len(n.component.Spec.Tolerations) > 0 {
		tolerations = n.component.Spec.Tolerations
	}
	affinity := &corev1.Affinity{}
	if n.component.Spec.Affinity != nil {
		affinity = n.component.Spec.Affinity
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
					ImagePullSecrets:              imagePullSecrets(n.component, n.cluster),
					TerminationGracePeriodSeconds: commonutil.Int64(0),
					ServiceAccountName:            "wutong-operator",
					HostAliases:                   hostsAliases(n.cluster),
					HostPID:                       true,
					DNSPolicy:                     corev1.DNSClusterFirstWithHostNet,
					HostNetwork:                   true,
					Tolerations:                   tolerations,
					Affinity:                      affinity,
					Containers: []corev1.Container{
						{
							Name:            NodeName,
							Image:           n.component.Spec.Image,
							ImagePullPolicy: n.component.ImagePullPolicy(),
							Env:             envs,
							Args:            args,
							VolumeMounts:    volumeMounts,
							ReadinessProbe:  readinessProbe,
						},
					},
					Volumes: volumes,
				},
			},
		},
	}

	return ds
}
