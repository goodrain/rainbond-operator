package handler

import (
	"context"
	"fmt"
	"os"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// LocalPathName name for local-path-provisioner
var LocalPathName = "local-path-provisioner"
var LocalPathSAName = "local-path-provisioner-service-account"

type localPath struct {
	ctx       context.Context
	client    client.Client
	component *rainbondv1alpha1.RbdComponent
	labels    map[string]string
}

var _ ComponentHandler = &localPath{}

// NewLocalPath creates a new rbd-local-path handler.
func NewLocalPath(ctx context.Context, client client.Client, component *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster) ComponentHandler {
	return &localPath{
		ctx:       ctx,
		client:    client,
		component: component,
		labels:    LabelsForRainbondComponent(component),
	}
}

func (l *localPath) Before() error {
	return nil
}

func (l *localPath) Resources() []client.Object {
	return []client.Object{
		l.serviceAccount(),
		l.role(),
		l.roleBinding(),
		l.clusterRole(),
		l.clusterRoleBinding(),
		l.configMap(),
		l.deployment(),
		l.storageClass(),
	}
}

func (l *localPath) After() error {
	return nil
}

func (l *localPath) ListPods() ([]corev1.Pod, error) {
	return listPods(l.ctx, l.client, l.component.Namespace, l.labels)
}

func (l *localPath) configMap() client.Object {
	helperPodYaml := fmt.Sprintf(`apiVersion: v1
kind: Pod
metadata:
  name: helper-pod
spec:
  priorityClassName: system-node-critical
  tolerations:
    - key: node.kubernetes.io/disk-pressure
      operator: Exists
      effect: NoSchedule
  containers:
    - name: helper-pod
      image: %s/alpine:latest
      imagePullPolicy: IfNotPresent`, os.Getenv("RAINBOND_IMAGE_REPOSITORY"))
	data := map[string]string{
		"config.json": `{
        "nodePathMap": [
          {
            "node": "DEFAULT_PATH_FOR_NON_LISTED_NODES",
            "paths": ["/opt/rainbond/local-path-provisioner"]
          }
        ]
      }`,
		"setup": `#!/bin/sh
set -eu
mkdir -m 0777 -p "$VOL_DIR"`,
		"teardown": `#!/bin/sh
set -eu
rm -rf "$VOL_DIR"`,
		"helperPod.yaml": helperPodYaml,
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "local-path-config",
			Namespace: l.component.Namespace,
		},
		Data: data,
	}

	return cm
}

func (l *localPath) storageClass() client.Object {
	sc := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "local-path",
			Labels: map[string]string{"accessModes": "rwo"},
		},
		Provisioner: "rancher.io/local-path",
		VolumeBindingMode: func() *storagev1.VolumeBindingMode {
			mode := storagev1.VolumeBindingWaitForFirstConsumer
			return &mode
		}(),
		ReclaimPolicy: func() *corev1.PersistentVolumeReclaimPolicy {
			rp := corev1.PersistentVolumeReclaimDelete
			return &rp
		}(),
	}

	return sc
}

func (l *localPath) deployment() client.Object {
	// 定义 configVolume
	configVolume := corev1.Volume{
		Name: "config-volume",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "local-path-config",
				},
			},
		},
	}

	// 定义容器
	container := corev1.Container{
		Name:            "local-path-provisioner",
		Image:           l.component.Spec.Image,
		ImagePullPolicy: l.component.ImagePullPolicy(),
		Resources:       l.component.Spec.Resources,
		Command: []string{
			"local-path-provisioner",
			"--debug",
			"start",
			"--config",
			"/etc/config/config.json",
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "config-volume",
				MountPath: "/etc/config/",
			},
		},
		Env: []corev1.EnvVar{
			{
				Name: "POD_NAMESPACE",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.namespace",
					},
				},
			},
			{
				Name:  "CONFIG_MOUNT_PATH",
				Value: "/etc/config/",
			},
		},
	}

	// 创建 Deployment
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "local-path-provisioner",
			Namespace: l.component.Namespace,
			Labels:    l.labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: l.component.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: l.labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: l.labels,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: LocalPathSAName,
					Containers: []corev1.Container{
						container,
					},
					Volumes: []corev1.Volume{
						configVolume,
					},
				},
			},
		},
	}

	return deploy
}

func (l *localPath) serviceAccount() client.Object {
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      LocalPathSAName,
			Namespace: l.component.Namespace,
		},
	}
	return sa
}

func (l *localPath) role() client.Object {
	role := &v1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "local-path-provisioner-role",
			Namespace: l.component.Namespace,
		},
		Rules: []v1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"pods"},
				Verbs:     []string{"get", "list", "watch", "create", "patch", "update", "delete"},
			},
		},
	}
	return role
}

func (l *localPath) roleBinding() client.Object {
	rb := &v1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "local-path-provisioner-bind",
			Namespace: l.component.Namespace,
		},
		RoleRef: v1.RoleRef{
			APIGroup: v1.SchemeGroupVersion.Group,
			Kind:     "Role",
			Name:     "local-path-provisioner-role",
		},
		Subjects: []v1.Subject{
			{
				Kind:      v1.ServiceAccountKind,
				Name:      LocalPathSAName,
				Namespace: l.component.Namespace,
			},
		},
	}
	return rb
}

func (l *localPath) clusterRole() client.Object {
	cr := &v1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: "local-path-provisioner-role",
		},
		Rules: []v1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"nodes", "persistentvolumeclaims", "configmaps", "pods", "pods/log"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"persistentvolumes"},
				Verbs:     []string{"get", "list", "watch", "create", "patch", "update", "delete"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"events"},
				Verbs:     []string{"create", "patch"},
			},
			{
				APIGroups: []string{"storage.k8s.io"},
				Resources: []string{"storageclasses"},
				Verbs:     []string{"get", "list", "watch"},
			},
		},
	}
	return cr
}

func (l *localPath) clusterRoleBinding() client.Object {
	crb := &v1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "local-path-provisioner-bind",
		},
		RoleRef: v1.RoleRef{
			APIGroup: v1.SchemeGroupVersion.Group,
			Kind:     "ClusterRole",
			Name:     "local-path-provisioner-role",
		},
		Subjects: []v1.Subject{
			{
				Kind:      v1.ServiceAccountKind,
				Name:      LocalPathSAName,
				Namespace: l.component.Namespace,
			},
		},
	}
	return crb
}
