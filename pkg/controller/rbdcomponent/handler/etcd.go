package handler

import (
	"context"
	"fmt"
	"os"
	"strconv"

	etcdv1beta2 "github.com/coreos/etcd-operator/pkg/apis/etcd/v1beta2"
	"github.com/docker/distribution/reference"
	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/goodrain/rainbond-operator/pkg/util/commonutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// EtcdName name for rbd-etcd.
var EtcdName = "rbd-etcd"

type etcd struct {
	ctx       context.Context
	client    client.Client
	component *rainbondv1alpha1.RbdComponent
	cluster   *rainbondv1alpha1.RainbondCluster
	labels    map[string]string

	pvcParametersRWO *pvcParameters
	storageRequest   int64

	enableEtcdOperator bool
}

var _ ComponentHandler = &etcd{}
var _ StorageClassRWOer = &etcd{}

// NewETCD creates a new rbd-etcd handler.
func NewETCD(ctx context.Context, client client.Client, component *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster) ComponentHandler {
	labels := LabelsForRainbondComponent(component)
	labels["etcd_node"] = EtcdName
	return &etcd{
		ctx:            ctx,
		client:         client,
		component:      component,
		cluster:        cluster,
		labels:         labels,
		storageRequest: getStorageRequest("ETCD_DATA_STORAGE_REQUEST", 21),
	}
}

func (e *etcd) Before() error {
	if e.cluster.Spec.EtcdConfig != nil {
		return NewIgnoreError(fmt.Sprintf("specified etcd configuration"))
	}
	if enableEtcdOperator, _ := strconv.ParseBool(os.Getenv("ENABLE_ETCD_OPERATOR")); enableEtcdOperator {
		e.enableEtcdOperator = true
	}
	if err := setStorageCassName(e.ctx, e.client, e.component.Namespace, e); err != nil {
		return err
	}

	return nil
}

func (e *etcd) Resources() []interface{} {
	if e.enableEtcdOperator {
		return []interface{}{
			e.etcdCluster(),
		}
	}

	return []interface{}{
		e.statefulsetForEtcd(),
		e.serviceForEtcd(),
	}
}

func (e *etcd) After() error {
	return nil
}

func (e *etcd) ListPods() ([]corev1.Pod, error) {
	labels := e.labels
	if e.enableEtcdOperator {
		labels = map[string]string{
			// app=etcd,etcd_cluster=example-etcd-cluster
			"app":          "etcd",
			"etcd_cluster": EtcdName,
		}
	}
	return listPods(e.ctx, e.client, e.component.Namespace, labels)
}

func (e *etcd) SetStorageClassNameRWO(pvcParameters *pvcParameters) {
	e.pvcParametersRWO = pvcParameters
}

func (e *etcd) Replicas() *int32 {
	if !e.enableEtcdOperator {
		commonutil.Int32(1)
	}
	return nil
}

func (e *etcd) statefulsetForEtcd() interface{} {
	claimName := "data"
	pvc := createPersistentVolumeClaimRWO(e.component.Namespace, claimName, e.pvcParametersRWO, e.labels, e.storageRequest)
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      EtcdName,
			Namespace: e.component.Namespace,
			Labels:    e.labels,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:    commonutil.Int32(1),
			ServiceName: EtcdName,
			Selector: &metav1.LabelSelector{
				MatchLabels: e.labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      EtcdName,
					Namespace: e.component.Namespace,
					Labels:    e.labels,
				},
				Spec: corev1.PodSpec{
					TerminationGracePeriodSeconds: commonutil.Int64(0),
					Containers: []corev1.Container{
						{
							Name:            EtcdName,
							Image:           e.component.Spec.Image,
							ImagePullPolicy: e.component.ImagePullPolicy(),
							Command: []string{
								"/usr/local/bin/etcd",
								"--name",
								EtcdName,
								"--initial-advertise-peer-urls",
								fmt.Sprintf("http://%s:2380", EtcdName),
								"--listen-peer-urls",
								"http://0.0.0.0:2380",
								"--listen-client-urls",
								"http://0.0.0.0:2379",
								"--advertise-client-urls",
								fmt.Sprintf("http://%s:2379", EtcdName),
								"--initial-cluster",
								fmt.Sprintf("%s=http://%s:2380", EtcdName, EtcdName),
								"--initial-cluster-state",
								"new",
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "client",
									ContainerPort: 2379,
								},
								{
									Name:          "server",
									ContainerPort: 2380,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      claimName,
									MountPath: "/var/lib/etcd",
								},
							},
						},
					},
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{*pvc},
		},
	}
	return sts
}

func (e *etcd) serviceForEtcd() interface{} {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      EtcdName,
			Namespace: e.component.Namespace,
			Labels:    e.labels,
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "None",
			Ports: []corev1.ServicePort{
				{
					Name: "client",
					Port: 2379,
				},
				{
					Name: "server",
					Port: 2380,
				},
			},
			Selector: e.labels,
		},
	}

	return svc
}

func (e *etcd) etcdCluster() *etcdv1beta2.EtcdCluster {
	claimName := "data"
	pvc := createPersistentVolumeClaimRWO(e.component.Namespace, claimName, e.pvcParametersRWO, e.labels, e.storageRequest)

	// make sure the image name is right
	repo, _ := reference.Parse(e.component.Spec.Image)
	named := repo.(reference.Named)
	tag := "latest"
	if t, ok := repo.(reference.Tagged); ok {
		tag = t.Tag()
	}

	defaultSize := 1
	if e.component.Spec.Replicas != nil {
		defaultSize = int(*e.component.Spec.Replicas)
	}

	affinity := &corev1.Affinity{
		PodAntiAffinity: &corev1.PodAntiAffinity{
			PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
				{
					Weight: 100,
					PodAffinityTerm: corev1.PodAffinityTerm{
						LabelSelector: &metav1.LabelSelector{
							MatchExpressions: []metav1.LabelSelectorRequirement{
								{
									Key:      "app",
									Operator: metav1.LabelSelectorOpIn,
									Values: []string{
										"etcd",
									},
								},
							},
						},
						TopologyKey: "kubernetes.io/hostname",
					},
				},
			},
		},
	}

	ec := &etcdv1beta2.EtcdCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      EtcdName,
			Namespace: e.component.Namespace,
			Labels:    e.labels,
		},
		Spec: etcdv1beta2.ClusterSpec{
			Size:       defaultSize,
			Repository: named.Name(),
			Version:    tag,
			Pod: &etcdv1beta2.PodPolicy{
				Affinity:                  affinity,
				PersistentVolumeClaimSpec: &pvc.Spec,
			},
		},
	}

	return ec
}
