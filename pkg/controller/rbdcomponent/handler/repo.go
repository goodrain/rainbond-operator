package handler

import (
	"context"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/goodrain/rainbond-operator/pkg/util/commonutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// RepoName name for rbd-repo.
var RepoName = "rbd-repo"

type repo struct {
	ctx       context.Context
	client    client.Client
	component *rainbondv1alpha1.RbdComponent
	cluster   *rainbondv1alpha1.RainbondCluster
	labels    map[string]string

	pvcParametersRWO *pvcParameters
	storageRequest   int64
}

var _ ComponentHandler = &repo{}
var _ StorageClassRWOer = &repo{}

// NewRepo creates a new rbd-repo hanlder.
func NewRepo(ctx context.Context, client client.Client, component *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster) ComponentHandler {
	return &repo{
		ctx:            ctx,
		client:         client,
		component:      component,
		cluster:        cluster,
		labels:         LabelsForRainbondComponent(component),
		storageRequest: getStorageRequest("REPO_DATA_STORAGE_REQUEST", 21),
	}
}

func (r *repo) Before() error {
	if err := setStorageCassName(r.ctx, r.client, r.component.Namespace, r); err != nil {
		return err
	}
	return nil
}

func (r *repo) Resources() []interface{} {
	return []interface{}{
		r.statefulset(),
		r.serviceForRepo(),
	}
}

func (r *repo) After() error {
	return nil
}

func (r *repo) ListPods() ([]corev1.Pod, error) {
	return listPods(r.ctx, r.client, r.component.Namespace, r.labels)
}

func (r *repo) SetStorageClassNameRWO(pvcParameters *pvcParameters) {
	r.pvcParametersRWO = pvcParameters
}

func (r *repo) statefulset() interface{} {
	claimName := "data"
	repoDataPVC := createPersistentVolumeClaimRWO(r.component.Namespace, claimName, r.pvcParametersRWO, r.labels, r.storageRequest)

	ds := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      RepoName,
			Namespace: r.component.Namespace,
			Labels:    r.labels,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: r.component.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: r.labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   RepoName,
					Labels: r.labels,
				},
				Spec: corev1.PodSpec{
					ImagePullSecrets:              imagePullSecrets(r.component, r.cluster),
					TerminationGracePeriodSeconds: commonutil.Int64(0),
					Containers: []corev1.Container{
						{
							Name:            RepoName,
							Image:           r.component.Spec.Image,
							ImagePullPolicy: r.component.ImagePullPolicy(),
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      claimName,
									MountPath: "/var/opt/jfrog/artifactory",
								},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse("512Mi"),
									corev1.ResourceCPU:    resource.MustParse("0m"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse("2048Mi"),
									corev1.ResourceCPU:    resource.MustParse("1000m"),
								},
							},
						},
					},
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				*repoDataPVC,
			},
		},
	}

	return ds
}

func (r *repo) serviceForRepo() interface{} {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      RepoName,
			Namespace: r.component.Namespace,
			Labels:    r.labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name: "http",
					Port: 80,
				},
			},
			Selector: r.labels,
		},
	}
	return svc
}
