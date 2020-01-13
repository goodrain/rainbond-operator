package handler

import (
	rainbondv1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/GLYASAI/rainbond-operator/pkg/util/k8sutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var RepoName = "rbd-repo"

type repo struct {
	component *rainbondv1alpha1.RbdComponent
	cluster   *rainbondv1alpha1.RainbondCluster
	labels    map[string]string
}

func NewRepo(ctx context.Context, client client.Client, component *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster) ComponentHandler {
	return &repo{
		component: component,
		cluster:   cluster,
		labels:    component.Labels(),
	}
}

func (r *repo) Before() error {
	// TODO: check prerequisites
	return nil
}

func (r *repo) Resources() []interface{} {
	return []interface{}{
		r.daemonSetForRepo(),
	}
}

func (r *repo) After() error {
	return nil
}

func (r *repo) daemonSetForRepo() interface{} {
	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      RepoName,
			Namespace: r.component.Namespace,
			Labels:    r.labels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: r.labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   RepoName,
					Labels: r.labels,
				},
				Spec: corev1.PodSpec{
					Tolerations: []corev1.Toleration{
						{
							Key:    "node-role.kubernetes.io/master",
							Effect: corev1.TaintEffectNoSchedule,
						},
					},
					NodeSelector: map[string]string{
						"node-role.kubernetes.io/master": "",
					},
					Containers: []corev1.Container{
						{
							Name:            RepoName,
							Image:           r.component.Spec.Image,
							ImagePullPolicy: r.component.ImagePullPolicy(),
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "repo-data",
									MountPath: "/var/opt/jfrog/artifactory",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "repo-data",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/opt/rainbond/data/repo",
									Type: k8sutil.HostPath(corev1.HostPathDirectoryOrCreate),
								},
							},
						},
					},
				},
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
