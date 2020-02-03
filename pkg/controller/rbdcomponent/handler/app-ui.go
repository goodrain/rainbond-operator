package handler

import (
	"context"
	"fmt"
	"strconv"

	rainbondv1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/GLYASAI/rainbond-operator/pkg/util/commonutil"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var AppUIName = "rbd-app-ui"

type appui struct {
	ctx       context.Context
	client    client.Client
	component *rainbondv1alpha1.RbdComponent
	cluster   *rainbondv1alpha1.RainbondCluster
	labels    map[string]string
	db        *rainbondv1alpha1.Database
}

func NewAppUI(ctx context.Context, client client.Client, component *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster) ComponentHandler {
	return &appui{
		ctx:       ctx,
		client:    client,
		component: component,
		cluster:   cluster,
		labels:    component.GetLabels(),
	}
}

func (a *appui) Before() error {
	a.db = getDefaultDBInfo(a.cluster.Spec.UIDatabase)

	if err := isPhaseOK(a.cluster); err != nil {
		return err
	}

	return isUIDBReady(a.ctx, a.client, a.cluster)
}

func (a *appui) Resources() []interface{} {
	return []interface{}{
		a.deploymentForAppUI(),
		a.serviceForAppUI(),
		a.ingressForAppUI(),
	}
}

func (a *appui) After() error {
	return nil
}

func (a *appui) deploymentForAppUI() interface{} {
	cpt := a.component
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      AppUIName,
			Namespace: cpt.Namespace,
			Labels:    cpt.GetLabels(),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: commonutil.Int32(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: cpt.GetLabels(),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   AppUIName,
					Labels: cpt.GetLabels(),
				},
				Spec: corev1.PodSpec{
					TerminationGracePeriodSeconds: commonutil.Int64(0),
					Containers: []corev1.Container{
						{
							Name:            AppUIName,
							Image:           cpt.Spec.Image,
							ImagePullPolicy: cpt.ImagePullPolicy(),
							Env: []corev1.EnvVar{
								{
									Name:  "MYSQL_HOST",
									Value: a.db.Host,
								},
								{
									Name:  "MYSQL_PORT",
									Value: strconv.Itoa(a.db.Port),
								},
								{
									Name:  "MYSQL_USER",
									Value: a.db.Username,
								},
								{
									Name:  "MYSQL_PASS",
									Value: a.db.Password,
								},
								{
									Name:  "MYSQL_DB",
									Value: "console",
								},
								{
									Name:  "REGION_URL",
									Value: "http://rbd-api:8888",
								},
								{
									Name:  "REGION_WS_URL",
									Value: fmt.Sprintf("ws://%s:6060", a.cluster.GatewayIngressIP()),
								},
								{
									Name:  "REGION_HTTP_DOMAIN",
									Value: a.cluster.Spec.SuffixHTTPHost,
								},
								{
									Name:  "REGION_TCP_DOMAIN",
									Value: a.cluster.GatewayIngressIP(),
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "ssl",
									MountPath: "/app/region/ssl",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "ssl",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: apiSecretName,
								},
							},
						},
					},
				},
			},
		},
	}

	return deploy
}

func (a *appui) serviceForAppUI() interface{} {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      AppUIName,
			Namespace: a.component.Namespace,
			Labels:    a.labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name: "http",
					Port: 7070,
					TargetPort: intstr.IntOrString{
						IntVal: 7070,
					},
				},
			},
			Selector: a.labels,
		},
	}

	return svc
}

func (a *appui) ingressForAppUI() interface{} {
	ing := &extensions.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      AppUIName,
			Namespace: a.component.Namespace,
			Annotations: map[string]string{
				"nginx.ingress.kubernetes.io/l4-enable": "true",
				"nginx.ingress.kubernetes.io/l4-host":   "0.0.0.0",
				"nginx.ingress.kubernetes.io/l4-port":   "7070",
			},
			Labels: a.labels,
		},
		Spec: extensions.IngressSpec{
			Backend: &extensions.IngressBackend{
				ServiceName: AppUIName,
				ServicePort: intstr.FromString("http"),
			},
		},
	}
	return ing
}
