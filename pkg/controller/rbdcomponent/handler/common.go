package handler

import (
	"context"
	"errors"
	"path"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var ErrNoDBEndpoints = errors.New("no ready endpoints for DB were found")

const (
	EtcdSSLPath = "/run/ssl/etcd"
)

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

func getDefaultDBInfo(in *rainbondv1alpha1.Database) *rainbondv1alpha1.Database {
	if in != nil {
		return in
	}
	return &rainbondv1alpha1.Database{
		Host:     DBName,
		Port:     3306,
		Username: "root",
		Password: "rainbond",
	}
}

func checkPackageStatus(pkg *rainbondv1alpha1.RainbondPackage) error {
	var packageCompleted bool
	if pkg.Status != nil {
		for _, cond := range pkg.Status.Conditions {
			if cond.Type == rainbondv1alpha1.Ready && cond.Status == rainbondv1alpha1.Completed {
				packageCompleted = true
				break
			}
		}
	}
	if !packageCompleted {
		return NewIgnoreError("rainbond package is not completed in InstallationModeWithoutPackage mode")
	}
	return nil
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
		return []string{"http://etcd0:2379"}
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
