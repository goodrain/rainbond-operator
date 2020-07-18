package k8sutil

import (
	"context"
	"fmt"
	"k8s.io/kubectl/pkg/describe"
	"net"
	"os"
	"time"

	"github.com/goodrain/rainbond-operator/pkg/util/commonutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func MustNewKubeConfig(kubeconfigPath string) *rest.Config {
	if kubeconfigPath != "" {
		cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			panic(err)
		}
		return cfg
	}

	cfg, err := InClusterConfig()
	if err != nil {
		panic(err)
	}
	return cfg
}

func NewKubeConfig() (*rest.Config, error) {
	cfg, err := InClusterConfig()
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func InClusterConfig() (*rest.Config, error) {
	// Work around https://github.com/kubernetes/kubernetes/issues/40973
	// See https://github.com/coreos/etcd-operator/issues/731#issuecomment-283804819
	if len(os.Getenv("KUBERNETES_SERVICE_HOST")) == 0 {
		addrs, err := net.LookupHost("kubernetes.default.svc")
		if err != nil {
			panic(err)
		}
		os.Setenv("KUBERNETES_SERVICE_HOST", addrs[0])
	}
	if len(os.Getenv("KUBERNETES_SERVICE_PORT")) == 0 {
		os.Setenv("KUBERNETES_SERVICE_PORT", "443")
	}
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

// HostPath returns a pointer to the HostPathType value passed in.
func HostPath(hostpath corev1.HostPathType) *corev1.HostPathType {
	return &hostpath
}

// MountPropagationMode returns a pointer to the MountPropagationMode value passed in.
func MountPropagationMode(moundPropagationMode corev1.MountPropagationMode) *corev1.MountPropagationMode {
	return &moundPropagationMode
}

// PersistentVolumeReclaimPolicy returns a pointer to the PersistentVolumeReclaimPolicy value passed in.
func PersistentVolumeReclaimPolicy(persistentVolumeReclaimPolicy corev1.PersistentVolumeReclaimPolicy) *corev1.PersistentVolumeReclaimPolicy {
	return &persistentVolumeReclaimPolicy
}

func UpdateCRStatus(client client.Client, obj runtime.Object) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	err := client.Status().Update(ctx, obj)
	if err != nil {
		err = client.Status().Update(ctx, obj)
		if err != nil {
			return fmt.Errorf("update custom resource status: %v", err)
		}
	}
	return nil
}

func MaterRoleLabel(key string) map[string]string {
	var labels map[string]string
	switch key {
	case describe.LabelNodeRolePrefix + "master":
		labels = map[string]string{
			key: "",
		}
	case describe.NodeLabelRole:
		labels = map[string]string{
			key: "master",
		}
	}
	return labels
}

func CreatePersistentVolumeClaim(ns, claimName string, accessModes []corev1.PersistentVolumeAccessMode, labels map[string]string, storageClassName string, storageRequest int64) *corev1.PersistentVolumeClaim {
	size := resource.NewQuantity(storageRequest*1024*1024*1024, resource.BinarySI)
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      claimName,
			Namespace: ns,
			Labels:    labels,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: accessModes,
			Resources: corev1.ResourceRequirements{
				Requests: map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceStorage: *size,
				},
			},
			StorageClassName: commonutil.String(storageClassName),
		},
	}

	return pvc
}
