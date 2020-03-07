package handler

import (
	"context"
	"fmt"
	"strconv"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/goodrain/rainbond-operator/pkg/util/commonutil"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// AppUIName name for rbd-app-ui resources.
var AppUIName = "rbd-app-ui"

type appui struct {
	ctx       context.Context
	client    client.Client
	labels    map[string]string
	db        *rainbondv1alpha1.Database
	component *rainbondv1alpha1.RbdComponent
	cluster   *rainbondv1alpha1.RainbondCluster

	pvcParametersRWX *pvcParameters
	pvcName          string
	storageRequest   int64
}

var _ ComponentHandler = &appui{}
var _ StorageClassRWXer = &appui{}

// NewAppUI creates a new rbd-app-ui handler.
func NewAppUI(ctx context.Context, client client.Client, component *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster) ComponentHandler {
	return &appui{
		ctx:            ctx,
		client:         client,
		component:      component,
		cluster:        cluster,
		labels:         LabelsForRainbondComponent(component),
		pvcName:        "rbd-app-ui",
		storageRequest: getStorageRequest("APP_UI_DATA_STORAGE_REQUEST", 1),
	}
}

func (a *appui) Before() error {
	db, err := getDefaultDBInfo(a.ctx, a.client, a.cluster.Spec.UIDatabase, a.component.Namespace, DBName)
	if err != nil {
		return fmt.Errorf("get db info: %v", err)
	}
	a.db = db

	if err := setStorageCassName(a.ctx, a.client, a.component.Namespace, a); err != nil {
		return err
	}

	return isUIDBReady(a.ctx, a.client, a.component, a.cluster)
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

func (a *appui) ListPods() ([]corev1.Pod, error) {
	return listPods(a.ctx, a.client, a.component.Namespace, a.labels)
}

func (a *appui) SetStorageClassNameRWX(pvcParameters *pvcParameters) {
	a.pvcParametersRWX = pvcParameters
}

func (a *appui) ResourcesCreateIfNotExists() []interface{} {
	return []interface{}{
		// pvc is immutable after creation except resources.requests for bound claims
		createPersistentVolumeClaimRWX(a.component.Namespace, a.pvcName, a.pvcParametersRWX, a.labels),
	}
}

func (a *appui) deploymentForAppUI() interface{} {
	cpt := a.component
	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      AppUIName,
			Namespace: cpt.Namespace,
			Labels:    a.labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: a.component.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: a.labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   AppUIName,
					Labels: a.labels,
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
									Value: "https://rbd-api-api:8443",
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
								{
									Name:  "IMAGE_REPO",
									Value: a.cluster.Spec.ImageHub.Domain,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "ssl",
									MountPath: "/app/region/ssl",
								},
								{
									Name:      "logs",
									MountPath: "/app/logs/",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "ssl",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: apiClientSecretName,
								},
							},
						},
						{
							Name: "logs",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: a.pvcName,
								},
							},
						},
					},
				},
			},
		},
	}

	if a.cluster.Annotations != nil {
		if enterpriseID, ok := a.cluster.Annotations["enterprise_id"]; ok {
			deploy.Spec.Template.Spec.Containers[0].Env = append(deploy.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{Name: "ENTERPRISE_ID", Value: enterpriseID})
		}
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
