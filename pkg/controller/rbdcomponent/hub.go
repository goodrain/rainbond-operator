package rbdcomponent

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	rainbondv1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/GLYASAI/rainbond-operator/pkg/util/commonutil"
	"github.com/GLYASAI/rainbond-operator/pkg/util/rbduitl"
)

var defaultPVCName = "hubdata"
var rbdHubServiceName = "rbd-hub"

func resourcesForHub(r *rainbondv1alpha1.RbdComponent) []interface{}{
	return []interface{}{
		secretForHub(r),
		daemonSetForHub(r),
		serviceForHub(r),
		persistentVolumeClaimForHub(r),
		ingressForHub(r),
	}
}

// daemonSetForHub returns a privateregistry Daemonset object
func daemonSetForHub(r *rainbondv1alpha1.RbdComponent) interface{} {
	labels := r.Labels()
	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rbd-hub",
			Namespace: r.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "rbd-hub",
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            "rbd-hub",
							Image:           "rainbond/rbd-registry:2.6.2",
							ImagePullPolicy: corev1.PullAlways, // TODO: custom
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
									ClaimName: defaultPVCName,
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

func serviceForHub(rc *rainbondv1alpha1.RbdComponent) interface{} {
	labels := rbdutil.LabelsForRainbondResource()
	labels["name"] = rbdHubServiceName

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rbdHubServiceName,
			Namespace: rc.Namespace,
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
			Selector: labelsForPrivateRegistry("rbd-hub"), // TODO
		},
	}

	return svc
}

func persistentVolumeClaimForHub(p *rainbondv1alpha1.RbdComponent) interface{} {
	storageRequest := resource.NewQuantity(10, resource.DecimalSI)
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      defaultPVCName,
			Namespace: p.Namespace,
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
			StorageClassName: commonutil.String("rbd-nfs"), // TODO: do not hard code
		},
	}

	return pvc
}

func ingressForHub(rc *rainbondv1alpha1.RbdComponent) interface{} {
	labels := rbdutil.LabelsForRainbondResource()
	labels["name"] = "rbd-hub"

	ing := &extensions.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rbd-hub",
			Namespace: rc.Namespace,
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
										ServiceName: rbdHubServiceName,
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
						"goodrain.me",
					},
					SecretName: "rbd-hub-goodrain.me",
				},
			},
		},
	}

	return ing
}

func secretForHub(rc *rainbondv1alpha1.RbdComponent) interface{} {
	labels := rbdutil.LabelsForRainbondResource()
	labels["name"] = "rbd-hub-goodrain.me"

	_, pem, key, _ := commonutil.DomainSign("goodrain.me")

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rbd-hub-goodrain.me",
			Namespace: rc.Namespace,
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

// labelsForPrivateRegistry returns the labels for selecting the resources
// belonging to the given PrivateRegistry CR name.
func labelsForPrivateRegistry(name string) map[string]string {
	return map[string]string{"name": "rbd-hub"} // TODO: only one rainbond?
}
