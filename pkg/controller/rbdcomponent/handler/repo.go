package handler

import (
	"context"
	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/goodrain/rainbond-operator/pkg/util/commonutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var RepoName = "rbd-repo"

type repo struct {
	ctx       context.Context
	client    client.Client
	component *rainbondv1alpha1.RbdComponent
	cluster   *rainbondv1alpha1.RainbondCluster
	pkg       *rainbondv1alpha1.RainbondPackage
	labels    map[string]string

	storageClassNameRWO string
}

var _ ComponentHandler = &repo{}
var _ StorageClassRWOer = &repo{}

func NewRepo(ctx context.Context, client client.Client, component *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster, pkg *rainbondv1alpha1.RainbondPackage) ComponentHandler {
	return &repo{
		ctx:       ctx,
		client:    client,
		component: component,
		cluster:   cluster,
		labels:    component.GetLabels(),
		pkg:       pkg,
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
		r.daemonSetForRepo(),
		r.serviceForRepo(),
	}
}

func (r *repo) After() error {
	return nil
}

func (r *repo) SetStorageClassNameRWO(storageClassName string) {
	r.storageClassNameRWO = storageClassName
}

func (r *repo) daemonSetForRepo() interface{} {
	claimName := "rbd-repo-data"
	repoDataPVC := createPersistentVolumeClaimRWO(r.component.Namespace, r.storageClassNameRWO, claimName)

	ds := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      RepoName,
			Namespace: r.component.Namespace,
			Labels:    r.labels,
		},
		Spec: appsv1.StatefulSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: r.labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   RepoName,
					Labels: r.labels,
				},
				Spec: corev1.PodSpec{
					TerminationGracePeriodSeconds: commonutil.Int64(0),
					Tolerations: []corev1.Toleration{
						{
							Key:    r.cluster.Status.MasterRoleLabel,
							Effect: corev1.TaintEffectNoSchedule,
						},
					},
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
