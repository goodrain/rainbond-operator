package rbdcomponent

import (
	rainbondv1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var rbdEventLogName = "rbd-eventlog"

func resourcesForEventLog(r *rainbondv1alpha1.RbdComponent) []interface{} {
	return []interface{}{
		daemonSetForEventLog(r),
	}
}

func daemonSetForEventLog(r *rainbondv1alpha1.RbdComponent) interface{} {
	labels := r.Labels()
	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rbdEventLogName,
			Namespace: r.Namespace, // TODO: can use custom namespace?
			Labels:    labels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   rbdEventLogName,
					Labels: labels,
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
							Name:            rbdEventLogName,
							Image:           r.Spec.Image,
							ImagePullPolicy: corev1.PullIfNotPresent, // TODO: custom
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
									Value: "https://172.20.0.11:6443",
								},
								{
									Name:  "DOCKER_LOG_SAVE_DAY",
									Value: "7",
								},
							},
							Args: []string{ // TODO: huangrh
								"--cluster.bind.ip=$(POD_IP)",
								"--cluster.instance.ip=$(POD_IP)",
								"--db.url=root:rainbond@tcp(rbd-db:3306)/region",
								"--discover.etcd.addr=http://etcd0:2379",
								"--eventlog.bind.ip=$(POD_IP)",
								"--websocket.bind.ip=$(POD_IP)",
							},
						},
					},
				},
			},
		},
	}

	return ds
}
