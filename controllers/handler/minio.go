package handler

import (
	"context"
	"fmt"
	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
	"github.com/goodrain/rainbond-operator/util/commonutil"
	"github.com/goodrain/rainbond-operator/util/rbdutil"
	minio "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MinIOName name for minIO
var MinIOName = "minio"

type minIO struct {
	ctx              context.Context
	client           client.Client
	component        *rainbondv1alpha1.RbdComponent
	pvcParametersRWO *pvcParameters
	labels           map[string]string
	storageRequest   int64
	cluster          *rainbondv1alpha1.RainbondCluster
}

var _ ComponentHandler = &minIO{}

// NewMinIO creates a new minIO handler.
func NewMinIO(ctx context.Context, client client.Client, component *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster) ComponentHandler {
	return &minIO{
		ctx:            ctx,
		client:         client,
		component:      component,
		labels:         LabelsForRainbondComponent(component),
		storageRequest: getStorageRequest("MINIO_DATA_STORAGE_REQUEST", 20),
		cluster:        cluster,
	}
}

func (m *minIO) Before() error {
	return setStorageCassName(m.ctx, m.client, m.component.Namespace, m)
}

func (m *minIO) Resources() []client.Object {
	return []client.Object{
		m.statefulSet(),
		m.service(),
	}
}

func (m *minIO) After() error {
	ctx := context.Background()
	minioClient, err := minio.New("http://minio-service:9000", &minio.Options{
		Creds:  credentials.NewStaticV4("admin", "admin1234", ""),
		Secure: false, // 如果使用 HTTP，设置为 false
	})
	if err != nil {
		return fmt.Errorf("failed to create minio client: %w", err)
	}

	// 检查桶是否存在
	exists, err := minioClient.BucketExists(ctx, "rbd-hub")
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}

	// 如果桶不存在，则创建桶
	if !exists {
		err = minioClient.MakeBucket(ctx, "rbd-hub", minio.MakeBucketOptions{})
		if err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}
	}
	return nil
}

func (m *minIO) ListPods() ([]corev1.Pod, error) {
	return listPods(m.ctx, m.client, m.component.Namespace, m.labels)
}

func (m *minIO) SetStorageClassNameRWO(pvcParameters *pvcParameters) {
	m.pvcParametersRWO = pvcParameters
}

func (m *minIO) statefulSet() client.Object {
	claimName := "minio-data" // PersistentVolumeClaim 名称
	minioPVC := createPersistentVolumeClaimRWO(m.component.Namespace, claimName, m.pvcParametersRWO, m.labels, m.storageRequest)

	vms := append(m.component.Spec.VolumeMounts, corev1.VolumeMount{
		Name:      claimName,
		MountPath: "/data", // MinIO 数据存储路径
	})

	ds := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "minio",
			Namespace: m.component.Namespace,
			Labels:    m.labels,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: m.component.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: m.labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: m.labels,
				},
				Spec: corev1.PodSpec{
					ImagePullSecrets:              imagePullSecrets(m.component, m.cluster),
					TerminationGracePeriodSeconds: commonutil.Int64(0),
					ServiceAccountName:            rbdutil.GetenvDefault("SERVICE_ACCOUNT_NAME", "rainbond-operator"),
					Containers: []corev1.Container{
						{
							Image:           m.component.Spec.Image,
							ImagePullPolicy: m.component.ImagePullPolicy(),
							Name:            "minio",
							Command: []string{
								"bin/bash",
								"-c",
							},
							Args: []string{
								"minio server /data --console-address :9001",
							},
							VolumeMounts: vms,
							Resources:    m.component.Spec.Resources,
							Env: []corev1.EnvVar{
								{
									Name:  "MINIO_BUCKETS",
									Value: "rbd-hub",
								}, {
									Name:  "MINIO_ROOT_USER",
									Value: "admin",
								}, {
									Name:  "MINIO_ROOT_PASSWORD",
									Value: rbdutil.GetenvDefault("RBD_MINIO_ROOT_PASSWORD", "admin1234"),
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: claimName,
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: claimName,
								},
							},
						},
					},
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{*minioPVC},
		},
	}

	return ds
}

func (m *minIO) service() client.Object {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "minio-service",
			Namespace: m.component.Namespace,
			Labels:    m.labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "minio-9000",
					Port:       9000,
					TargetPort: intstr.FromInt(9000),
				},
				{
					Name:       "minio-9001",
					Port:       9001,
					TargetPort: intstr.FromInt(9001),
				},
			},
			Selector: m.labels,
			Type:     corev1.ServiceTypeNodePort, // 或者改为 LoadBalancer
		},
	}

	return svc
}
