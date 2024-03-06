package handler

import (
	"context"
	"errors"
	"fmt"
	"k8s.io/apimachinery/pkg/util/intstr"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/goodrain/rainbond-operator/util/commonutil"
	"github.com/goodrain/rainbond-operator/util/constants"
	"github.com/goodrain/rainbond-operator/util/rbdutil"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ErrNoDBEndpoints -
var ErrNoDBEndpoints = errors.New("no ready endpoints for DB were found")

const (
	// EtcdSSLPath ssl file path for etcd
	EtcdSSLPath = "/run/ssl/etcd"
	// RegionDatabaseName -
	RegionDatabaseName = "region"
	// ConsoleDatabaseName -
	ConsoleDatabaseName = "console"
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

	dbcpt := &rainbondv1alpha1.RbdComponent{}
	if err := cli.Get(ctx, types.NamespacedName{Namespace: cpt.Namespace, Name: DBName}, dbcpt); err != nil {
		return err
	}

	if dbcpt.Status.ReadyReplicas == 0 {
		return errors.New("no ready replicas for rbdcomponent rbd-db")
	}

	return nil
}

func isUIDBMigrateOK(ctx context.Context, cli client.Client, cpt *rainbondv1alpha1.RbdComponent) error {
	job := &batchv1.Job{}
	if err := cli.Get(ctx, types.NamespacedName{Namespace: cpt.Namespace, Name: AppUIDBMigrationsName}, job); err != nil {
		if k8sErrors.IsNotFound(err) {
			return NewIgnoreError(fmt.Sprintf("job %s not found", AppUIDBMigrationsName))
		}
		return err
	}

	var complete bool
	for _, cond := range job.Status.Conditions {
		if cond.Type == batchv1.JobComplete && cond.Status == corev1.ConditionTrue {
			complete = true
			break
		}
	}

	if !complete {
		return NewIgnoreError(fmt.Sprintf("job %s not complete", AppUIDBMigrationsName))
	}

	return nil
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
		Host:     dbhost,
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
		MountPath: EtcdSSLPath,
	}
	return volume, mount
}

func volumeByContainerdSocket() (corev1.Volume, corev1.VolumeMount) {
	volume := corev1.Volume{
		Name: "containerdsock",
		VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: "/run/containerd/containerd.sock",
			},
		},
	}
	mount := corev1.VolumeMount{
		Name:      "containerdsock",
		MountPath: "/run/containerd/containerd.sock",
	}
	return volume, mount
}

func volumeByDockerSocket() (corev1.Volume, corev1.VolumeMount) {
	volume := corev1.Volume{
		Name: "dockersock",
		VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: "/var/run/docker.sock",
			},
		},
	}
	mount := corev1.VolumeMount{
		Name:      "dockersock",
		MountPath: "/var/run/docker.sock",
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
	return storageClassNameFromRainbondVolume(ctx, cli, ns, false)
}

func storageClassNameFromRainbondVolumeRWO(ctx context.Context, cli client.Client, ns string) (*pvcParameters, error) {
	pvcParameters, err := storageClassNameFromRainbondVolume(ctx, cli, ns, true)
	if err != nil {
		if !IsRainbondVolumeNotFound(err) {
			return nil, err
		}
		return storageClassNameFromRainbondVolumeRWX(ctx, cli, ns)
	}
	return pvcParameters, nil
}

func storageClassNameFromRainbondVolume(ctx context.Context, cli client.Client, ns string, rwo bool) (*pvcParameters, error) {
	var labels map[string]string
	if rwo {
		labels = rbdutil.LabelsForAccessModeRWO()
	} else {
		labels = rbdutil.LabelsForAccessModeRWX()
	}
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
	}
	if !rwo {
		pvcParameters.storageRequest = commonutil.Int32(1)
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

func createPersistentVolumeClaimRWX(ns, claimName string, pvcParameters *pvcParameters, labels map[string]string, storageRequest int64) *corev1.PersistentVolumeClaim {
	accessModes := []corev1.PersistentVolumeAccessMode{
		corev1.ReadWriteMany,
	}
	return createPersistentVolumeClaim(ns, claimName, accessModes, pvcParameters, labels, storageRequest)
}

func createPersistentVolumeClaimRWO(ns, claimName string, pvcParameters *pvcParameters, labels map[string]string, storageRequest int64) *corev1.PersistentVolumeClaim {
	accessModes := []corev1.PersistentVolumeAccessMode{
		corev1.ReadWriteOnce,
	}
	return createPersistentVolumeClaim(ns, claimName, accessModes, pvcParameters, labels, storageRequest)
}

func createPersistentVolumeClaim(ns, claimName string, accessModes []corev1.PersistentVolumeAccessMode, pvcParameters *pvcParameters, labels map[string]string, storageRequest int64) *corev1.PersistentVolumeClaim {
	if pvcParameters.storageRequest != nil && storageRequest == 0 {
		storageRequest = int64(*pvcParameters.storageRequest)
	}
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
			StorageClassName: commonutil.String(pvcParameters.storageClassName),
		},
	}

	return pvc
}

func getNodeNames(cli client.Client) ([]string, error) {
	nodeList := &corev1.NodeList{}
	if err := cli.List(context.Background(), nodeList); err != nil {
		return nil, err
	}

	var nodeNames []string
	for _, n := range nodeList.Items {
		nodeNames = append(nodeNames, n.Name)
	}
	return nodeNames, nil
}

func affinityForRequiredNodes(nodeNames []string) *corev1.Affinity {
	return &corev1.Affinity{
		NodeAffinity: &corev1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
				NodeSelectorTerms: []corev1.NodeSelectorTerm{
					{
						// You cannot use matchFields directly to select multiple nodes.
						// When nodes have no labels, there will be problems.
						// More info: https://github.com/kubernetes/kubernetes/issues/78238#issuecomment-495373236
						MatchExpressions: []corev1.NodeSelectorRequirement{
							{
								Key:      "kubernetes.io/hostname",
								Operator: corev1.NodeSelectorOpIn,
								Values:   nodeNames,
							},
						},
					},
					{
						MatchExpressions: []corev1.NodeSelectorRequirement{
							{
								Key:      "k3s.io/hostname",
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

func copyLabels(m map[string]string) map[string]string {
	cp := make(map[string]string)
	for k, v := range m {
		cp[k] = v
	}

	return cp
}

func hostsAliases(cluster *rainbondv1alpha1.RainbondCluster) []corev1.HostAlias {
	var hostAliases []corev1.HostAlias
	if rbdutil.GetImageRepository(cluster) == constants.DefImageRepository {
		hostAliases = append(hostAliases, corev1.HostAlias{
			IP:        cluster.InnerGatewayIngressIP(),
			Hostnames: []string{rbdutil.GetImageRepository(cluster)},
		})
	}
	return hostAliases
}

func listPods(ctx context.Context, cli client.Client, namespace string, labels map[string]string) ([]corev1.Pod, error) {
	podList := &corev1.PodList{}
	var opts []client.ListOption
	opts = append(opts, client.InNamespace(namespace))
	opts = append(opts, client.MatchingLabels(labels))
	if err := cli.List(ctx, podList, opts...); err != nil {
		return nil, err
	}
	if len(podList.Items) == 0 {
		log.V(6).Info("pod list is empty", "labels", labels)
	}
	return podList.Items, nil
}

func isEtcdAvailable(ctx context.Context, cli client.Client, cpt *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster) error {
	if cluster.Spec.EtcdConfig != nil {
		return nil
	}

	dbcpt := &rainbondv1alpha1.RbdComponent{}
	if err := cli.Get(ctx, types.NamespacedName{Namespace: cpt.Namespace, Name: EtcdName}, dbcpt); err != nil {
		return err
	}

	if dbcpt.Status.ReadyReplicas == 0 {
		return errors.New("no ready replicas for rbdcomponent rbd-etcd")
	}

	return nil
}

func getStorageRequest(env string, defSize int64) int64 {
	storageRequest, _ := strconv.ParseInt(os.Getenv(env), 10, 64)
	if storageRequest == 0 {
		storageRequest = defSize
	}
	return storageRequest
}

func imagePullSecrets(cpt *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster) []corev1.LocalObjectReference {
	// pirority component does not support pulling images with credentials
	if cpt.Spec.PriorityComponent {
		return nil
	}
	if cluster.Status.ImagePullSecret == nil {
		return nil
	}
	return []corev1.LocalObjectReference{
		*cluster.Status.ImagePullSecret,
	}
}

func mergeArgs(commonArgs, priorityArgs []string) []string {
	prioritySet := make(map[string]struct{})
	for _, arg := range priorityArgs {
		prioritySet[strings.Split(arg, "=")[0]] = struct{}{}
	}
	for _, arg := range commonArgs {
		key := strings.Split(arg, "=")[0]
		if _, ok := prioritySet[key]; ok {
			continue
		}
		priorityArgs = append(priorityArgs, arg)
	}
	return priorityArgs
}

func mergeEnvs(commonEnvs, priorityEnvs []corev1.EnvVar) []corev1.EnvVar {
	envSet := make(map[string]struct{})
	for _, env := range priorityEnvs {
		envSet[env.Name] = struct{}{}
	}
	for _, env := range commonEnvs {
		if _, ok := envSet[env.Name]; ok {
			continue
		}
		priorityEnvs = append(priorityEnvs, env)
	}
	return priorityEnvs
}

func mergeResources(commonResources, priorityResources corev1.ResourceRequirements) corev1.ResourceRequirements {
	if priorityResources.Requests != nil {
		commonResources.Requests = priorityResources.Requests
	}
	if priorityResources.Limits != nil {
		commonResources.Limits = priorityResources.Limits
	}
	return commonResources
}

func mergeVolumes(commonVolumes, priorityVolumes []corev1.Volume) []corev1.Volume {
	set := make(map[string]struct{})
	for _, vol := range priorityVolumes {
		set[vol.Name] = struct{}{}
	}
	for _, vol := range commonVolumes {
		if _, ok := set[vol.Name]; ok {
			continue
		}
		priorityVolumes = append(priorityVolumes, vol)
	}
	return priorityVolumes
}

func mergeVolumeMounts(commonMountVolumes, priorityMountVolumes []corev1.VolumeMount) []corev1.VolumeMount {
	set := make(map[string]struct{})
	for _, vol := range priorityMountVolumes {
		set[vol.Name] = struct{}{}
	}
	for _, vol := range commonMountVolumes {
		if _, ok := set[vol.Name]; ok {
			continue
		}
		priorityMountVolumes = append(priorityMountVolumes, vol)
	}
	return priorityMountVolumes
}

func createIngress(name, namespace string, annotations, labels map[string]string, serviceName, servicePortName string) *networkingv1.Ingress {
	return &networkingv1.Ingress{
		ObjectMeta: createIngressMeta(name, namespace, annotations, labels),
		Spec: networkingv1.IngressSpec{
			DefaultBackend: &networkingv1.IngressBackend{
				Service: &networkingv1.IngressServiceBackend{
					Name: serviceName,
					Port: networkingv1.ServiceBackendPort{
						Name: servicePortName,
					},
				},
			},
		},
	}

}

func createLegacyIngress(name, namespace string, annotations, labels map[string]string, serviceName string, servicePort intstr.IntOrString) *networkingv1beta1.Ingress {
	return &networkingv1beta1.Ingress{
		ObjectMeta: createIngressMeta(name, namespace, annotations, labels),
		Spec: networkingv1beta1.IngressSpec{
			Backend: &networkingv1beta1.IngressBackend{
				ServiceName: serviceName,
				ServicePort: servicePort,
			},
		},
	}
}

func createIngressMeta(name, namespace string, annotations, labels map[string]string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:        name,
		Namespace:   namespace,
		Annotations: annotations,
		Labels:      labels,
	}
}
