package handler

import (
	"context"
	"fmt"
	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/goodrain/rainbond-operator/pkg/util/commonutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	kubeaggregatorv1beta1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//APIName name
var MetricsServerName = "metrics-server"
var metricsGroupAPI = "v1beta1.metrics.k8s.io"

type metricsServer struct {
	ctx       context.Context
	client    client.Client
	db        *rainbondv1alpha1.Database
	labels    map[string]string
	component *rainbondv1alpha1.RbdComponent
	cluster   *rainbondv1alpha1.RainbondCluster
	pkg       *rainbondv1alpha1.RainbondPackage
}

// NewMetricsServer creates a new metrics-server handler
func NewMetricsServer(ctx context.Context, client client.Client, component *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster, pkg *rainbondv1alpha1.RainbondPackage) ComponentHandler {
	return &metricsServer{
		ctx:       ctx,
		client:    client,
		component: component,
		cluster:   cluster,
		labels:    component.GetLabels(),
		pkg:       pkg,
	}
}

func (m *metricsServer) Before() error {
	apiservce := &kubeaggregatorv1beta1.APIService{}
	if err := m.client.Get(m.ctx, types.NamespacedName{Name: metricsGroupAPI}, apiservce); err != nil {
		if !k8sErrors.IsNotFound(err) {
			return fmt.Errorf("get apiservice(%s/%s): %v", MetricsServerName, m.cluster.Namespace, err)
		}
		return nil
	}
	createdByRainbond := apiservce.Spec.Service.Namespace == m.component.Namespace && apiservce.Spec.Service.Name == MetricsServerName
	if !createdByRainbond {
		return NewIgnoreError(fmt.Sprintf("%s already exists", metricsGroupAPI))
	}
	return nil
}

func (m *metricsServer) Resources() []interface{} {
	return []interface{}{
		m.deploySetForMetricsServer(),
		m.serviceForMetricsServer(),
	}
}

func (m *metricsServer) After() error {
	newApiService := m.apiserviceForMetricsServer()
	apiservce := &kubeaggregatorv1beta1.APIService{}
	if err := m.client.Get(m.ctx, types.NamespacedName{Name: metricsGroupAPI}, apiservce); err != nil {
		if !k8sErrors.IsNotFound(err) {
			return fmt.Errorf("get apiservice(%s/%s): %v", MetricsServerName, m.cluster.Namespace, err)
		}
		if err := m.client.Create(m.ctx, newApiService); err != nil {
			return fmt.Errorf("create new api service: %v", err)
		}
		return nil
	}

	log.Info(fmt.Sprintf("an old api service(%s) has been found, update it.", newApiService.GetName()))
	newApiService.ResourceVersion = apiservce.ResourceVersion
	if err := m.client.Update(m.ctx, newApiService); err != nil {
		return fmt.Errorf("update api service: %v", err)
	}
	return nil
}

func (m *metricsServer) deploySetForMetricsServer() interface{} {
	ds := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      MetricsServerName,
			Namespace: m.component.Namespace,
			Labels:    m.labels,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: m.labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   MetricsServerName,
					Labels: m.labels,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName:            "rainbond-operator",
					TerminationGracePeriodSeconds: commonutil.Int64(0),
					NodeSelector: map[string]string{
						"beta.kubernetes.io/os": "linux",
						"kubernetes.io/arch":    "amd64",
					},
					Containers: []corev1.Container{
						{
							Name:            MetricsServerName,
							Image:           m.component.Spec.Image,
							ImagePullPolicy: m.component.ImagePullPolicy(),
							Args: []string{
								"--cert-dir=/tmp",
								"--secure-port=4443",
								"--kubelet-insecure-tls",
								"--kubelet-preferred-address-types=InternalIP",
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "main-port",
									ContainerPort: 4443,
								},
							},
							SecurityContext: &corev1.SecurityContext{
								ReadOnlyRootFilesystem: commonutil.Bool(true),
								RunAsNonRoot:           commonutil.Bool(true),
								RunAsUser:              commonutil.Int64(1000),
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "tmp-dir",
									MountPath: "/tmp",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "tmp-dir",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
		},
	}

	return ds
}

func (m *metricsServer) serviceForMetricsServer() interface{} {
	labels := m.labels
	labels["kubernetes.io/name"] = "Metrics-server"
	labels["kubernetes.io/cluster-service"] = "true"
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      MetricsServerName,
			Namespace: m.component.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port: 443,
					TargetPort: intstr.IntOrString{
						IntVal: 4443,
					},
				},
			},
			Selector: m.labels,
		},
	}

	return svc
}

func (m *metricsServer) apiserviceForMetricsServer() *kubeaggregatorv1beta1.APIService {
	return &kubeaggregatorv1beta1.APIService{
		ObjectMeta: metav1.ObjectMeta{
			Name: metricsGroupAPI,
		},
		Spec: kubeaggregatorv1beta1.APIServiceSpec{
			Service: &kubeaggregatorv1beta1.ServiceReference{
				Name:      MetricsServerName,
				Namespace: m.cluster.Namespace,
			},
			Group:                 "metrics.k8s.io",
			Version:               "v1beta1",
			InsecureSkipTLSVerify: true,
			GroupPriorityMinimum:  100,
			VersionPriority:       30,
		},
	}
}
