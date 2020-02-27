package handler

import (
	"context"
	"errors"
	"fmt"
	"path"

	"github.com/goodrain/rainbond-operator/pkg/util/commonutil"
	"github.com/goodrain/rainbond-operator/pkg/util/rbdutil"
	"k8s.io/apimachinery/pkg/api/resource"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ErrNoDBEndpoints -
var ErrNoDBEndpoints = errors.New("no ready endpoints for DB were found")

const (
	// EtcdSSLPath ssl file path for etcd
	EtcdSSLPath = "/run/ssl/etcd"
)

// pvcParameters holds parameters to create pvc.
type pvcParameters struct {
	storageClassName string
	storageRequest   *int32
}

// LabelsForRainbondComponent returns the labels for the sub resources of rbdcomponent.
func LabelsForRainbondComponent(cpt *rainbondv1alpha1.RbdComponent) map[string]string {
	labels := rbdutil.LabelsForRainbond(nil)
	labels["name"] = cpt.Name
	return labels
}

func isUIDBReady(ctx context.Context, cli client.Client, cpt *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster) error {
	if cluster.Spec.UIDatabase != nil {
		return nil
	}
	labels := rbdutil.LabelsForRainbond(map[string]string{
		"name": DBName,
	})
	eps := &corev1.EndpointsList{}
	listOpts := []client.ListOption{
		client.MatchingLabels(labels),
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

func storageClassNameFromRainbondVolumeRWX(ctx context.Context, cli client.Client, ns string) (*pvcParameters, error) {
	return storageClassNameFromRainbondVolume(ctx, cli, ns, rbdutil.LabelsForAccessModeRWX())
}

func storageClassNameFromRainbondVolumeRWO(ctx context.Context, cli client.Client, ns string) (*pvcParameters, error) {
	pvcParameters, err := storageClassNameFromRainbondVolume(ctx, cli, ns, rbdutil.LabelsForAccessModeRWO())
	if err != nil {
		if !IsRainbondVolumeNotFound(err) {
			return nil, err
		}
		return storageClassNameFromRainbondVolumeRWX(ctx, cli, ns)
	}
	return pvcParameters, nil
}

func storageClassNameFromRainbondVolume(ctx context.Context, cli client.Client, ns string, labels map[string]string) (*pvcParameters, error) {
	volumeList := &rainbondv1alpha1.RainbondVolumeList{}
	var opts []client.ListOption
	opts = append(opts, client.InNamespace(ns))
	opts = append(opts, client.MatchingLabels(labels))
	if err := cli.List(ctx, volumeList, opts...); err != nil {
		return nil, err
	}

	if len(volumeList.Items) == 0 {
		return nil, NewIgnoreError(rainbondVolumeNotFound)
	}

	volume := volumeList.Items[0]
	if volume.Spec.StorageClassName == "" {
		return nil, NewIgnoreError("storage class not ready")
	}

	pvcParameters := &pvcParameters{
		storageClassName: volume.Spec.StorageClassName,
		storageRequest:   volume.Spec.StorageRequest,
	}
	return pvcParameters, nil
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

func createPersistentVolumeClaimRWX(ns, claimName string, pvcParameters *pvcParameters) *corev1.PersistentVolumeClaim {
	accessModes := []corev1.PersistentVolumeAccessMode{
		corev1.ReadWriteMany,
	}
	return createPersistentVolumeClaim(ns, claimName, accessModes, pvcParameters)
}

func createPersistentVolumeClaimRWO(ns, claimName string, pvcParameters *pvcParameters) *corev1.PersistentVolumeClaim {
	accessModes := []corev1.PersistentVolumeAccessMode{
		corev1.ReadWriteOnce,
	}
	return createPersistentVolumeClaim(ns, claimName, accessModes, pvcParameters)
}

func createPersistentVolumeClaim(ns, claimName string, accessModes []corev1.PersistentVolumeAccessMode, pvcParameters *pvcParameters) *corev1.PersistentVolumeClaim {
	var size int64 = 1
	if pvcParameters.storageRequest != nil && *pvcParameters.storageRequest > 0 {
		size = int64(*pvcParameters.storageRequest)
	}
	storageRequest := resource.NewQuantity(size*1024*1024*1024, resource.BinarySI)
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
			StorageClassName: commonutil.String(pvcParameters.storageClassName),
		},
	}

	return pvc
}

func affinityForRequiredNodes(nodeNames []string) *corev1.Affinity {
	return &corev1.Affinity{
		NodeAffinity: &corev1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
				NodeSelectorTerms: []corev1.NodeSelectorTerm{
					{
						MatchFields: []corev1.NodeSelectorRequirement{
							{
								Key:      "metadata.name",
								Operator: corev1.NodeSelectorOpIn,
								Values:   nodeNames,
							},
						},
					},
				},
			},
		},
	}
}
