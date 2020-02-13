package handler

import (
	"context"
	"fmt"
	"strings"

	"github.com/goodrain/rainbond-operator/pkg/util/commonutil"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var WebCliName = "rbd-webcli"

type webcli struct {
	ctx        context.Context
	client     client.Client
	component  *rainbondv1alpha1.RbdComponent
	cluster    *rainbondv1alpha1.RainbondCluster
	pkg        *rainbondv1alpha1.RainbondPackage
	labels     map[string]string
	etcdSecret *corev1.Secret
}

func NewWebCli(ctx context.Context, client client.Client, component *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster, pkg *rainbondv1alpha1.RainbondPackage) ComponentHandler {
	return &webcli{
		ctx:       ctx,
		client:    client,
		component: component,
		cluster:   cluster,
		labels:    component.GetLabels(),
		pkg:       pkg,
	}
}

func (w *webcli) Before() error {
	secret, err := etcdSecret(w.ctx, w.client, w.cluster)
	if err != nil {
		return fmt.Errorf("failed to get etcd secret: %v", err)
	}
	w.etcdSecret = secret

	return nil
}

func (w *webcli) Resources() []interface{} {
	return []interface{}{
		w.daemonSetForAPI(),
	}
}

func (w *webcli) After() error {
	return nil
}

func (w *webcli) daemonSetForAPI() interface{} {
	volumeMounts := []corev1.VolumeMount{}
	volumes := []corev1.Volume{}
	args := []string{
		"--hostIP=$(POD_IP)",
		fmt.Sprintf("--log-level=%s", w.component.LogLevel()),
		"--etcd-endpoints=" + strings.Join(etcdEndpoints(w.cluster), ","),
	}
	if w.etcdSecret != nil {
		volume, mount := volumeByEtcd(w.etcdSecret)
		volumeMounts = append(volumeMounts, mount)
		volumes = append(volumes, volume)
		args = append(args, etcdSSLArgs()...)
	}

	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      WebCliName,
			Namespace: w.component.Namespace,
			Labels:    w.labels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: w.labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   WebCliName,
					Labels: w.labels,
				},
				Spec: corev1.PodSpec{
					TerminationGracePeriodSeconds: commonutil.Int64(0),
					Tolerations: []corev1.Toleration{
						{
							Key:    w.cluster.Status.MasterRoleLabel,
							Effect: corev1.TaintEffectNoSchedule,
						},
					},
					NodeSelector: w.cluster.Status.FirstMasterNodeLabel(),
					Containers: []corev1.Container{
						{
							Name:            WebCliName,
							Image:           w.component.Spec.Image,
							ImagePullPolicy: w.component.ImagePullPolicy(),
							Env: []corev1.EnvVar{
								{
									Name: "POD_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.podIP",
										},
									},
								},
							},
							Args:         args,
							VolumeMounts: volumeMounts,
						},
					},
					Volumes: volumes,
				},
			},
		},
	}

	return ds
}
