package handler

import (
	"context"

	rainbondv1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/GLYASAI/rainbond-operator/pkg/util/constants"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var EventLogName = "rbd-eventlog"

type eventlog struct {
	component *rainbondv1alpha1.RbdComponent
	cluster   *rainbondv1alpha1.RainbondCluster
	labels    map[string]string
}

func NewEventLog(ctx context.Context, client client.Client, component *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster) ComponentHandler {
	return &eventlog{
		component: component,
		cluster:   cluster,
		labels:    component.Labels(),
	}
}

func (e *eventlog) Before() error {
	// TODO: check prerequisites
	return nil
}

func (e *eventlog) Resources() []interface{} {
	return []interface{}{
		e.daemonSetForEventLog(),
	}
}

func (e *eventlog) After() error {
	return nil
}

func (e *eventlog) daemonSetForEventLog() interface{} {
	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      EventLogName,
			Namespace: e.component.Namespace,
			Labels:    e.labels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: e.labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   EventLogName,
					Labels: e.labels,
				},
				Spec: corev1.PodSpec{
					HostNetwork: true,
					DNSPolicy:   corev1.DNSClusterFirstWithHostNet,
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
							Name:            EventLogName,
							Image:           e.component.Spec.Image,
							ImagePullPolicy: e.component.ImagePullPolicy(),
							Env: []corev1.EnvVar{
								{
									Name: "POD_IP",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.podIP",
										},
									},
								},
								{
									Name:  "K8S_MASTER",
									Value: "kubernetes",
								},
								{
									Name:  "DOCKER_LOG_SAVE_DAY",
									Value: "7",
								},
							},
							Args: []string{
								"--cluster.bind.ip=$(POD_IP)",
								"--cluster.instance.ip=$(POD_IP)",
								"--db.url=root:rainbond@tcp(rbd-db:3306)/region", // TODO: DO NOT HARD CODE
								"--discover.etcd.addr=http://etcd0:2379",         // TODO: DO NOT HARD CODE
								"--eventlog.bind.ip=$(POD_IP)",
								"--websocket.bind.ip=$(POD_IP)",
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "grdata",
									MountPath: "/grdata",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "grdata",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: constants.GrDataPVC,
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
