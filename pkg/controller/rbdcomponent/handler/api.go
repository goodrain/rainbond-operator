package handler

import (
	"fmt"
	"context"

	rainbondv1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/GLYASAI/rainbond-operator/pkg/util/commonutil"
	"github.com/GLYASAI/rainbond-operator/pkg/util/constants"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var APIName = "rbd-api"
var apiSecretName = "rbd-api-ssl"

type api struct {
	component *rainbondv1alpha1.RbdComponent
	cluster   *rainbondv1alpha1.RainbondCluster
	labels    map[string]string
}

func NewAPI(ctx context.Context, client client.Client, component *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster) ComponentHandler {
	return &api{
		component: component,
		cluster:   cluster,
		labels:    component.Labels(),
	}
}

func (a *api) Before() error {
	// No prerequisites, if no gateway-installed node is specified, install on all nodes that meet the conditions
	return nil
}

func (a *api) Resources() []interface{} {
	return []interface{}{
		a.secretForAPI(),
		a.daemonSetForAPI(),
		a.serviceForAPI(),
		a.ingressForAPI(),
	}
}

func (a *api) After() error {
	return nil
}

func (a *api) daemonSetForAPI() interface{} {
	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      APIName,
			Namespace: a.component.Namespace, // TODO: can use custom namespace?
			Labels:    a.labels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: a.labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   APIName,
					Labels: a.labels,
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
							Name:            APIName,
							Image:           a.component.Spec.Image,
							ImagePullPolicy: a.component.ImagePullPolicy(),
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
									Value: a.cluster.Spec.SuffixHTTPHost,
								},
							},
							Args: []string{
								"--api-addr-ssl=0.0.0.0:8443",
								"--api-addr=$(POD_IP):8888",
								fmt.Sprintf("--log-level=%s", a.component.LogLevel()),
								"--mysql=root:rainbond@tcp(rbd-db:3306)/region", // TODO: do not hard code
								"--api-ssl-enable=false",
								"--etcd=http://etcd0:2379", // TODO: do not hard code
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

func (a *api) serviceForAPI() interface{} {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      APIName,
			Namespace: a.component.Namespace,
			Labels:    a.labels,
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
			Selector: a.labels,
		},
	}

	return svc
}

func (a *api) secretForAPI() interface{} {
	labels := a.component.Labels()
	labels["name"] = apiSecretName

	caPem, pem, key, _ := commonutil.DomainSign("region.goodrain.me") // sign all gateway ip

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      apiSecretName,
			Namespace: a.component.Namespace,
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

func (a *api) ingressForAPI() interface{} {
	// TODO: tls
	ing := &extensions.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      APIName,
			Namespace: a.component.Namespace,
			Annotations: map[string]string{
				"nginx.ingress.kubernetes.io/weight":          "100",
				"nginx.ingress.kubernetes.io/proxy-body-size": "0",
			},
			Labels: a.labels,
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
										ServiceName: APIName,
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
