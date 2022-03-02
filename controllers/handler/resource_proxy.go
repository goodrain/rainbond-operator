package handler

import (
	"context"

	"k8s.io/apimachinery/pkg/api/resource"

	wutongv1alpha1 "github.com/wutong/wutong-operator/api/v1alpha1"
	"github.com/wutong/wutong-operator/util/commonutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ResourceProxyName name for wt-resource-proxy.
var ResourceProxyName = "wt-resource-proxy"

type resourceProxy struct {
	ctx       context.Context
	client    client.Client
	component *wutongv1alpha1.WutongComponent
	cluster   *wutongv1alpha1.WutongCluster
	labels    map[string]string

	pvcParametersRWO *pvcParameters
	storageRequest   int64
}

var _ ComponentHandler = &resourceProxy{}
var _ StorageClassRWOer = &resourceProxy{}

// NewResourceProxy creates a new wt-resourceProxy hanlder.
func NewResourceProxy(ctx context.Context, client client.Client, component *wutongv1alpha1.WutongComponent, cluster *wutongv1alpha1.WutongCluster) ComponentHandler {
	return &resourceProxy{
		ctx:            ctx,
		client:         client,
		component:      component,
		cluster:        cluster,
		labels:         LabelsForWutongComponent(component),
		storageRequest: getStorageRequest("RESOURCE_PROXY_DATA_STORAGE_REQUEST", 21),
	}
}

func (r *resourceProxy) Before() error {
	if err := setStorageCassName(r.ctx, r.client, r.component.Namespace, r); err != nil {
		return err
	}
	return nil
}

func (r *resourceProxy) Resources() []client.Object {
	return r.resource()
}

func (r *resourceProxy) After() error {
	return nil
}

func (r *resourceProxy) ListPods() ([]corev1.Pod, error) {
	return listPods(r.ctx, r.client, r.component.Namespace, r.labels)
}

func (r *resourceProxy) SetStorageClassNameRWO(pvcParameters *pvcParameters) {
	r.pvcParametersRWO = pvcParameters
}

func (r *resourceProxy) resource() []client.Object {
	claimName := "data"
	resourceProxyDataPVC := createPersistentVolumeClaimRWO(r.component.Namespace, claimName, r.pvcParametersRWO, r.labels, r.storageRequest)

	volumeMounts := []corev1.VolumeMount{
		{
			Name:      claimName,
			MountPath: "/data/nginx/cache",
		},
	}

	volumeMounts = mergeVolumeMounts(volumeMounts, r.component.Spec.VolumeMounts)

	resources := corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("512Mi"),
			corev1.ResourceCPU:    resource.MustParse("1000m"),
		},
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("64Mi"),
			corev1.ResourceCPU:    resource.MustParse("100m"),
		},
	}

	resources = mergeResources(resources, r.component.Spec.Resources)

	ds := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ResourceProxyName,
			Namespace: r.component.Namespace,
			Labels:    r.labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: r.component.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: r.labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   ResourceProxyName,
					Labels: r.labels,
				},
				Spec: corev1.PodSpec{
					ImagePullSecrets:              imagePullSecrets(r.component, r.cluster),
					TerminationGracePeriodSeconds: commonutil.Int64(0),
					Containers: []corev1.Container{
						{
							Name:            ResourceProxyName,
							Image:           r.component.Spec.Image,
							ImagePullPolicy: r.component.ImagePullPolicy(),
							VolumeMounts:    volumeMounts,
							Resources:       resources,
							Args:            r.component.Spec.Args,
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
		},
	}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ResourceProxyName,
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
	return []client.Object{ds, resourceProxyDataPVC, svc}
}
