package handler

import (
	"context"
	"fmt"
	"strings"

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
	ctx        context.Context
	client     client.Client
	component  *rainbondv1alpha1.RbdComponent
	cluster    *rainbondv1alpha1.RainbondCluster
	db         *rainbondv1alpha1.Database
	labels     map[string]string
	etcdSecret *corev1.Secret
}

func NewAPI(ctx context.Context, client client.Client, component *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster) ComponentHandler {
	return &api{
		ctx:       ctx,
		client:    client,
		component: component,
		cluster:   cluster,
		labels:    component.Labels(),
	}
}

func (a *api) Before() error {
	a.db = getDefaultDBInfo(a.cluster.Spec.RegionDatabase)

	secret, err := etcdSecret(a.ctx, a.client, a.cluster)
	if err != nil {
		return fmt.Errorf("failed to get etcd secret: %v", err)
	}
	a.etcdSecret = secret

	return isPhaseOK(a.cluster)
}

func (a *api) Resources() []interface{} {
	return []interface{}{
		a.secretForAPI(),
		a.daemonSetForAPI(),
		a.serviceForAPI(),
		a.ingressForAPI(),
		a.ingressForWebsocket(),
	}
}

func (a *api) After() error {
	return nil
}

func (a *api) daemonSetForAPI() interface{} {
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "grdata",
			MountPath: "/grdata",
		},
	}
	volumes := []corev1.Volume{
		{
			Name: "grdata",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: constants.GrDataPVC,
				},
			},
		},
	}
	args := []string{
		"--api-addr=$(POD_IP):8888",
		"--api-ssl-enable=false",
		fmt.Sprintf("--log-level=%s", a.component.LogLevel()),
		a.db.RegionDataSource(),
		"--etcd=" + strings.Join(etcdEndpoints(a.cluster), ","),
	}
	if a.etcdSecret != nil {
		volume, mount := volumeByEtcd(a.etcdSecret)
		volumeMounts = append(volumeMounts, mount)
		volumes = append(volumes, volume)
		args = append(args, etcdSSLArgs()...)
	}

	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      APIName,
			Namespace: a.component.Namespace,
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
				{
					Name: "ws",
					Port: 6060,
					TargetPort: intstr.IntOrString{
						IntVal: 6060,
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

	caPem, pem, key, _ := commonutil.DomainSign("rbd-api") // sign all gateway ip

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

func (a *api) ingressForWebsocket() interface{} {
	ing := &extensions.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      APIName + "-webcli",
			Namespace: a.component.Namespace,
			Annotations: map[string]string{
				"nginx.ingress.kubernetes.io/l4-enable": "true",
				"nginx.ingress.kubernetes.io/l4-host":   "0.0.0.0",
				"nginx.ingress.kubernetes.io/l4-port":   "6060",
			},
			Labels: a.labels,
		},
		Spec: extensions.IngressSpec{
			Backend: &extensions.IngressBackend{
				ServiceName: APIName,
				ServicePort: intstr.FromString("ws"),
			},
		},
	}
	return ing
}
