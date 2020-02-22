package handler

import (
	"context"
	"errors"
	"fmt"
	"github.com/goodrain/rainbond-operator/pkg/util/commonutil"
	"github.com/goodrain/rainbond-operator/pkg/util/constants"
	rbdutil "github.com/goodrain/rainbond-operator/pkg/util/rbduitl"
	"k8s.io/apimachinery/pkg/api/resource"
	"path"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var ErrNoDBEndpoints = errors.New("no ready endpoints for DB were found")

const (
	EtcdSSLPath = "/run/ssl/etcd"
)

// LabelsForRainbondComponent returns the labels for the sub resources of rbdcomponent.
func LabelsForRainbondComponent(cpt *rainbondv1alpha1.RbdComponent) map[string]string {
	labels := rbdutil.LabelsForRainbond(nil)
	labels["name"] = cpt.Name
	return labels
}

func isUIDBReady(ctx context.Context, cli client.Client, cluster *rainbondv1alpha1.RainbondCluster) error {
	if cluster.Spec.UIDatabase != nil {
		return nil
	}
	eps := &corev1.EndpointsList{}
	listOpts := []client.ListOption{
		client.MatchingLabels(map[string]string{
			"name":     DBName,
			"belongTo": "RainbondOperator", // TODO: DO NOT HARD CODE
		}),
	}
	if err := cli.List(ctx, eps, listOpts...); err != nil {
		return err
	}
	for _, ep := range eps.Items {
		for _, subset := range ep.Subsets {
			if len(subset.Addresses) > 0 {
				return nil
			}
		}
	}
	return ErrNoDBEndpoints
}

func getDefaultDBInfo(ctx context.Context, cli client.Client, in *rainbondv1alpha1.Database, namespace, name string) (*rainbondv1alpha1.Database, error) {
	if in != nil {
		// use custom db
		return in, nil
	}

	secret := &corev1.Secret{}
	if err := cli.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, secret); err != nil {
		if !k8sErrors.IsNotFound(err) {
			return nil, fmt.Errorf("get secret %s/%s: %v", name, namespace, err)
		}
		return nil, NewIgnoreError(fmt.Sprintf("secret %s/%s not fount: %v", name, namespace, err))
	}
	user := string(secret.Data[mysqlUserKey])
	pass := string(secret.Data[mysqlPasswordKey])

	return &rainbondv1alpha1.Database{
		Host:     DBName,
		Port:     3306,
		Username: user,
		Password: pass,
	}, nil
}

func etcdSecret(ctx context.Context, cli client.Client, cluster *rainbondv1alpha1.RainbondCluster) (*corev1.Secret, error) {
	if cluster.Spec.EtcdConfig == nil || cluster.Spec.EtcdConfig.SecretName == "" {
		// SecretName is empty, not using TLS.
		return nil, nil
	}
	secret := &corev1.Secret{}
	if err := cli.Get(ctx, types.NamespacedName{Namespace: cluster.Namespace, Name: cluster.Spec.EtcdConfig.SecretName}, secret); err != nil {
		return nil, err
	}
	return secret, nil
}
func getSecret(ctx context.Context, client client.Client, namespace, name string) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	if err := client.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, secret); err != nil {
		return nil, err
	}
	return secret, nil
}

func etcdEndpoints(cluster *rainbondv1alpha1.RainbondCluster) []string {
	if cluster.Spec.EtcdConfig == nil {
		return []string{"http://rbd-etcd:2379"}
	}
	return cluster.Spec.EtcdConfig.Endpoints
}

func volumeByEtcd(etcdSecret *corev1.Secret) (corev1.Volume, corev1.VolumeMount) {
	volume := corev1.Volume{
		Name: "etcdssl",
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: etcdSecret.Name,
			},
		}}
	mount := corev1.VolumeMount{
		Name:      "etcdssl",
		MountPath: "/run/ssl/etcd",
	}
	return volume, mount
}

func volumeByAPISecret(apiServerSecret *corev1.Secret) (corev1.Volume, corev1.VolumeMount) {
	volume := corev1.Volume{
		Name: "region-api-ssl",
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: apiServerSecret.Name,
			},
		}}
	mount := corev1.VolumeMount{
		Name:      "region-api-ssl",
		MountPath: "/etc/goodrain/region.goodrain.me/ssl/",
	}
	return volume, mount
}

func etcdSSLArgs() []string {
	return []string{
		"--etcd-ca=" + path.Join(EtcdSSLPath, "ca-file"),
		"--etcd-cert=" + path.Join(EtcdSSLPath, "cert-file"),
		"--etcd-key=" + path.Join(EtcdSSLPath, "key-file"),
	}
}

func storageClassNameFromRainbondVolumeRWX(ctx context.Context, cli client.Client, ns string) (string, error) {
	return storageClassNameFromRainbondVolume(ctx, cli, ns, rbdutil.LabelsForAccessModeRWX())
}

func storageClassNameFromRainbondVolumeRWO(ctx context.Context, cli client.Client, ns string) (string, error) {
	storageClassName, err := storageClassNameFromRainbondVolume(ctx, cli, ns, rbdutil.LabelsForAccessModeRWO())
	if err != nil {
		if !IsRainbondVolumeNotFound(err) {
			return "", err
		}
		return storageClassNameFromRainbondVolumeRWX(ctx, cli, ns)
	}
	return storageClassName, nil
}

func storageClassNameFromRainbondVolume(ctx context.Context, cli client.Client, ns string, labels map[string]string) (string, error) {
	volumeList := &rainbondv1alpha1.RainbondVolumeList{}
	var opts []client.ListOption
	opts = append(opts, client.InNamespace(ns))
	opts = append(opts, client.MatchingLabels(labels))
	if err := cli.List(ctx, volumeList, opts...); err != nil {
		return "", err
	}

	if len(volumeList.Items) == 0 {
		return "", NewIgnoreError(rainbondVolumeNotFound)
	}

	volume := volumeList.Items[0]
	if volume.Spec.StorageClassName == "" {
		return "", NewIgnoreError("storage class not ready")
	}
	return volume.Spec.StorageClassName, nil
}

func setStorageCassName(ctx context.Context, cli client.Client, ns string, obj interface{}) error {
	storageClassRWXer, ok := obj.(StorageClassRWXer)
	if ok {
		sc, err := storageClassNameFromRainbondVolumeRWX(ctx, cli, ns)
		if err != nil {
			return err
		}
		storageClassRWXer.SetStorageClassNameRWX(sc)
	}

	storageClassRWOer, ok := obj.(StorageClassRWOer)
	if ok {
		sc, err := storageClassNameFromRainbondVolumeRWO(ctx, cli, ns)
		if err != nil {
			return err
		}
		storageClassRWOer.SetStorageClassNameRWO(sc)
	}

	return nil
}

func createPersistentVolumeClaimRWX(ns, className, claimName string) *corev1.PersistentVolumeClaim {
	accessModes := []corev1.PersistentVolumeAccessMode{
		corev1.ReadWriteMany,
	}
	return createPersistentVolumeClaim(ns, className, claimName, accessModes)
}

func createPersistentVolumeClaimRWO(ns, className, claimName string) *corev1.PersistentVolumeClaim {
	accessModes := []corev1.PersistentVolumeAccessMode{
		corev1.ReadWriteOnce,
	}
	return createPersistentVolumeClaim(ns, className, claimName, accessModes)
}

func createPersistentVolumeClaim(ns, className, claimName string, accessModes []corev1.PersistentVolumeAccessMode) *corev1.PersistentVolumeClaim {
	storageRequest := resource.NewQuantity(21*1024*1024*1024, resource.BinarySI) // TODO: customer specified
	if className == constants.DefStorageClass || className == "nfs" {
		storageRequest = resource.NewQuantity(1*1024*1024, resource.BinarySI)
	}
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      claimName,
			Namespace: ns,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: accessModes,
			Resources: corev1.ResourceRequirements{
				Requests: map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceStorage: *storageRequest,
				},
			},
			StorageClassName: commonutil.String(className),
		},
	}

	return pvc
}
