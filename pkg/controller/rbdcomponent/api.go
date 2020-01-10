package rbdcomponent

import (
	rainbondv1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/GLYASAI/rainbond-operator/pkg/util/commonutil"
	"github.com/GLYASAI/rainbond-operator/pkg/util/rbduitl"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var rbdAPIName = "rbd-api" // TODO: use r.Name

func resourcesForAPI(r *rainbondv1alpha1.RbdComponent) []interface{} {
	return []interface{}{
		secretForAPI(r),
		daemonSetForAPI(r),
		serviceForAPI(r),
		ingressForAPI(r),
	}
}

func daemonSetForAPI(r *rainbondv1alpha1.RbdComponent) interface{} {
	labels := labelsForRbdComponent(map[string]string{
		"name": r.Name,
	})

	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rbdAPIName,
			Namespace: r.Namespace, // TODO: can use custom namespace?
			Labels:    labels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   rbdAPIName,
					Labels: labels,
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
							Name:            rbdAPIName,
							Image:           r.Spec.Image,
							ImagePullPolicy: corev1.PullIfNotPresent,
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
									Name:  "EX_DOMAIN",
									Value: "foobar.grapps.cn", // TODO: huangrh
								},
							},
							Args: []string{
								"--api-addr-ssl=0.0.0.0:8443",
								"--api-addr=$(POD_IP):8888",
								"--log-level=debug",
								"--mysql=root:rainbond@tcp(rbd-db:3306)/region",
								"--api-ssl-enable=false",
								"--etcd=http://etcd0:2379",
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "ssl",
									MountPath: "/etc/goodrain/region.goodrain.me/ssl",
								},
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
									ClaimName: "grdata",
								},
							},
						},
						{
							Name: "ssl",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "rbd-api-ssl", // TODO: check this secret before create rbd-api
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

func serviceForAPI(rc *rainbondv1alpha1.RbdComponent) interface{} {
	l := map[string]string{
		"name": rbdAPIName,
	}
	labels := rbdutil.Labels(l).WithRainbondLabels()

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rbdAPIName,
			Namespace: rc.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name: "http",
					Port: 8888,
					TargetPort: intstr.IntOrString{
						IntVal: 8888,
					},
				},
			},
			Selector: labels,
		},
	}

	return svc
}

func secretForAPI(rc *rainbondv1alpha1.RbdComponent) interface{} {
	labels := rbdutil.LabelsForRainbondResource()
	labels["name"] = "rbd-api-ssl"

	caPem, pem, key, _ := commonutil.DomainSign("region.goodrain.me") // sign all gateway ip

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rbd-api-ssl",
			Namespace: rc.Namespace,
			Labels:    labels,
		},
		Data: map[string][]byte{
			"server.pem":     pem,
			"server.key.pem": key,
			"ca.pem":         caPem,
			"cert_file":      pem,
			"key_file":       key,
			"ssl_ca_cert":    caPem,
			"tls.crt":        pem,
			"tls.key":        key,
		},
	}

	return secret
}

func ingressForAPI(rc *rainbondv1alpha1.RbdComponent) interface{} {
	l := map[string]string{
		"name": rbdAPIName,
	}
	labels := rbdutil.Labels(l).WithRainbondLabels()

	ing := &extensions.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rbdAPIName,
			Namespace: rc.Namespace,
			Annotations: map[string]string{
				"nginx.ingress.kubernetes.io/weight":          "100",
				"nginx.ingress.kubernetes.io/proxy-body-size": "0",
			},
			Labels: labels,
		},
		Spec: extensions.IngressSpec{
			Rules: []extensions.IngressRule{
				{
					Host: "region.goodrain.me",
					IngressRuleValue: extensions.IngressRuleValue{
						HTTP: &extensions.HTTPIngressRuleValue{
							Paths: []extensions.HTTPIngressPath{
								{
									Path: "/",
									Backend: extensions.IngressBackend{
										ServiceName: rbdAPIName,
										ServicePort: intstr.FromString("http"),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	return ing
}
