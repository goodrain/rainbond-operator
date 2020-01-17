package k8sutil

import (
	"context"
	"fmt"
	"net"
	"os"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func UpdateOrCreateResource(ctx context.Context, cli client.Client, reqLogger logr.Logger, obj runtime.Object, meta metav1.Object) error {
	err := cli.Get(ctx, types.NamespacedName{Name: meta.GetName(), Namespace: meta.GetNamespace()}, obj)
	if err != nil {
		if !errors.IsNotFound(err) {
			reqLogger.Error(err, fmt.Sprintf("Failed to get %s", obj.GetObjectKind()))
			return err
		}
		reqLogger.Info("Creating a new", obj.GetObjectKind().GroupVersionKind().Kind, "Namespace", meta.GetNamespace(), "Name", meta.GetName())
		err = cli.Create(ctx, obj)
		if err != nil {
			reqLogger.Error(err, fmt.Sprintf("Failed to create new %s", obj.GetObjectKind()), "Namespace", meta.GetNamespace(), "Name", meta.GetName())
			return err
		}
		return nil
	}

	// obj exsits, update
	reqLogger.Info(fmt.Sprintf("Update %s", obj.GetObjectKind().GroupVersionKind().Kind), "Namespace", meta.GetNamespace(), "Name", meta.GetName())
	if err := cli.Update(ctx, obj); err != nil {
		reqLogger.Error(err, "Failed to update ", obj.GetObjectKind())
		return err
	}

	return nil
}

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

func IsKubernetesResourceNotFoundError(err error) bool {
	return errors.IsNotFound(err)
}
