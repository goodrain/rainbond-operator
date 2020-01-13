package handler

import (
	rainbondv1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/GLYASAI/rainbond-operator/pkg/util/commonutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var HubName = "rbd-hub"
var hubDataPvcName = "hub-data"
var hubImageRepository = "hub-image-repository"

type hub struct {
	component *rainbondv1alpha1.RbdComponent
	cluster   *rainbondv1alpha1.RainbondCluster
}

func NewHub(component *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster) ComponentHandler {
	return &hub{
		component: component,
		cluster:   cluster,
	}
}

func (h *hub) Before() error {
	return nil
}

func (h *hub) Resources() []interface{} {
	return []interface{}{
		h.secretForHub(), // important! create secret before ingress.
		h.daemonSetForHub(),
		h.secretForHub(),
		h.persistentVolumeClaimForHub(),
		h.ingressForHub(),
	}
}

func (h *hub) After() error {
	return nil
}

func (h *hub) daemonSetForHub() interface{} {
	labels := h.component.Labels()
	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      HubName,
			Namespace: h.component.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   HubName,
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
							Name:            "rbd-hub",
							Image:           h.component.Spec.Image,
							ImagePullPolicy: h.component.ImagePullPolicy(),
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "hubdata",
									MountPath: "/var/lib/registry",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "hubdata",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: hubDataPvcName,
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

func (h *hub) serviceForHub() interface{} {
	labels := h.component.Labels()
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      HubName,
			Namespace: h.component.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name: "main",
					Port: 5000,
					TargetPort: intstr.IntOrString{
						IntVal: 5000,
					},
				},
			},
			Selector: labels,
		},
	}

	return svc
}

func (h *hub) persistentVolumeClaimForHub() interface{} {
	storageRequest := resource.NewQuantity(10, resource.DecimalSI) // TODO: DO NOT HARD CODE.
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      hubDataPvcName,
			Namespace: h.component.Namespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteMany,
			},
			Resources: corev1.ResourceRequirements{
				Requests: map[corev1.ResourceName]resource.Quantity{
					corev1.ResourceStorage: *storageRequest,
				},
			},
			StorageClassName: commonutil.String(h.cluster.StorageClass()),
		},
	}

	return pvc
}

func (h *hub) ingressForHub() interface{} {
	labels := h.component.Labels()
	ing := &extensions.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      HubName,
			Namespace: h.component.Namespace,
			Annotations: map[string]string{
				"nginx.ingress.kubernetes.io/weight":                       "100",
				"nginx.ingress.kubernetes.io/proxy-body-size":              "0",
				"nginx.ingress.kubernetes.io/set-header-Host":              "$http_host",
				"nginx.ingress.kubernetes.io/set-header-X-Forwarded-Host":  "$http_host",
				"nginx.ingress.kubernetes.io/set-header-X-Forwarded-Proto": "https",
				"nginx.ingress.kubernetes.io/set-header-X-Scheme":          "$scheme",
			},
			Labels: labels,
		},
		Spec: extensions.IngressSpec{
			Rules: []extensions.IngressRule{
				{
					Host: "goodrain.me",
					IngressRuleValue: extensions.IngressRuleValue{
						HTTP: &extensions.HTTPIngressRuleValue{
							Paths: []extensions.HTTPIngressPath{
								{
									Path: "/v2/",
									Backend: extensions.IngressBackend{
										ServiceName: HubName,
										ServicePort: intstr.FromInt(5000),
									},
								},
							},
						},
					},
				},
			},
			TLS: []extensions.IngressTLS{
				{
					Hosts: []string{
						h.cluster.ImageRepository(),
					},
					SecretName: hubImageRepository,
				},
			},
		},
	}

	return ing
}

func (h *hub) secretForHub() interface{} {
	labels := h.component.Labels()
	labels["name"] = hubImageRepository

	_, pem, key, _ := commonutil.DomainSign(h.cluster.ImageRepository())

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      hubImageRepository,
			Namespace: h.component.Namespace,
			Labels:    labels,
		},
		Data: map[string][]byte{
			"tls.crt": pem,
			"tls.key": key,
			"cert":    pem,
		},
	}

	return secret
}
