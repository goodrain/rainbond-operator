package handler

import (
	"context"
	"errors"
	"fmt"
	"time"

	"k8s.io/client-go/kubernetes"

	wutongv1alpha1 "github.com/wutong-paas/wutong-operator/api/v1alpha1"
	"github.com/wutong-paas/wutong-operator/util/commonutil"
	"github.com/wutong-paas/wutong-operator/util/k8sutil"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	plabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	kubeaggregatorv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ErrV1beta1MetricsExists -
var ErrV1beta1MetricsExists = errors.New("v1beta1.metrics.k8s.io already exists")

// MetricsServerName name for metrics-server
var MetricsServerName = "metrics-server"
var metricsGroupAPI = "v1beta1.metrics.k8s.io"

type metricsServer struct {
	ctx        context.Context
	client     client.Client
	db         *wutongv1alpha1.Database
	labels     map[string]string
	component  *wutongv1alpha1.WutongComponent
	cluster    *wutongv1alpha1.WutongCluster
	apiservice *kubeaggregatorv1.APIService

	pods []corev1.Pod
}

var _ ComponentHandler = &metricsServer{}
var _ Replicaser = &metricsServer{}

// NewMetricsServer creates a new metrics-server handler
func NewMetricsServer(ctx context.Context, client client.Client, component *wutongv1alpha1.WutongComponent, cluster *wutongv1alpha1.WutongCluster) ComponentHandler {
	return &metricsServer{
		ctx:       ctx,
		client:    client,
		component: component,
		cluster:   cluster,
		labels:    LabelsForWutongComponent(component),
	}
}

func (m *metricsServer) Before() error {
	apiservice := &kubeaggregatorv1.APIService{}
	if err := m.client.Get(m.ctx, types.NamespacedName{Name: metricsGroupAPI}, apiservice); err != nil {
		if !k8sErrors.IsNotFound(err) {
			return fmt.Errorf("get apiservice(%s/%s): %v", MetricsServerName, m.cluster.Namespace, err)
		}

		if apiservice.Spec.Service.Namespace != MetricsServerName || apiservice.Spec.Service.Name != m.cluster.Namespace {
			// delete the wrong apiservice and return error
			m.client.Delete(m.ctx, apiservice)
			return fmt.Errorf("get apiservice(%s/%s): %v", MetricsServerName, m.cluster.Namespace, err)
		}
		return nil
	}
	m.apiservice = apiservice
	return nil
}

func (m *metricsServer) apiServiceCreatedByWutong() bool {
	apiservice := m.apiservice
	if apiservice == nil {
		return true
	}
	return apiservice.Spec.Service.Namespace == m.component.Namespace && apiservice.Spec.Service.Name == MetricsServerName
}

func (m *metricsServer) Resources() []client.Object {
	if !m.apiServiceCreatedByWutong() {
		return nil
	}
	return []client.Object{
		m.deployment(),
		m.serviceForMetricsServer(),
	}
}

func (m *metricsServer) After() error {
	if !m.apiServiceCreatedByWutong() {
		return nil
	}

	newAPIService := m.apiserviceForMetricsServer()
	apiservice := &kubeaggregatorv1.APIService{}
	if err := m.client.Get(m.ctx, types.NamespacedName{Name: metricsGroupAPI}, apiservice); err != nil {
		if !k8sErrors.IsNotFound(err) {
			return fmt.Errorf("get apiservice(%s/%s): %v", MetricsServerName, m.cluster.Namespace, err)
		}
		if err := m.client.Create(m.ctx, newAPIService); err != nil {
			return fmt.Errorf("create new api service: %v", err)
		}
		return nil
	}

	if !apiServiceNeedUpgrade(apiservice, newAPIService) {
		return nil
	}

	log.Info(fmt.Sprintf("an old api service(%s) has been found, update it.", newAPIService.GetName()))
	newAPIService.ResourceVersion = apiservice.ResourceVersion
	if err := m.client.Update(m.ctx, newAPIService); err != nil {
		return fmt.Errorf("update api service: %v", err)
	}
	return nil
}

func apiServiceNeedUpgrade(old, new *kubeaggregatorv1.APIService) bool {
	if old.Spec.Service == nil {
		return true
	}
	oldService, newService := old.Spec.Service, new.Spec.Service
	if oldService.Name != newService.Name || oldService.Namespace != newService.Namespace {
		return true
	}
	return false
}

func (m *metricsServer) ListPods() ([]corev1.Pod, error) {

	labels := m.labels
	if !m.apiServiceCreatedByWutong() {
		restConfig := k8sutil.MustNewKubeConfig("")
		clientset := kubernetes.NewForConfigOrDie(restConfig)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		defer cancel()
		svcRef := m.apiservice.Spec.Service
		svc, err := clientset.CoreV1().Services(svcRef.Namespace).Get(ctx, svcRef.Name, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("get svc(%s/%s) based on apiservice %s: %v", svcRef.Namespace, svcRef.Name, m.apiservice.Name, err)
		}

		labels = svc.Spec.Selector
		selector := plabels.SelectorFromSet(labels)
		opts := metav1.ListOptions{
			LabelSelector: selector.String(),
		}
		podList, err := clientset.CoreV1().Pods(svcRef.Namespace).List(ctx, opts)
		if err != nil {
			return nil, err
		}
		m.pods = podList.Items
		return podList.Items, nil
	}
	pods, err := listPods(m.ctx, m.client, m.component.Namespace, labels)
	m.pods = pods
	return pods, err
}

func (m *metricsServer) Replicas() *int32 {
	return commonutil.Int32(int32(len(m.pods)))
}

func (m *metricsServer) deployment() client.Object {
	ds := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      MetricsServerName,
			Namespace: m.component.Namespace,
			Labels:    m.labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: m.component.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: m.labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   MetricsServerName,
					Labels: m.labels,
				},
				Spec: corev1.PodSpec{
					ImagePullSecrets:              imagePullSecrets(m.component, m.cluster),
					ServiceAccountName:            "wutong-operator",
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
							Resources: m.component.Spec.Resources,
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

func (m *metricsServer) serviceForMetricsServer() client.Object {
	labels := copyLabels(m.labels)
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

func (m *metricsServer) apiserviceForMetricsServer() *kubeaggregatorv1.APIService {
	return &kubeaggregatorv1.APIService{
		ObjectMeta: metav1.ObjectMeta{
			Name: metricsGroupAPI,
		},
		Spec: kubeaggregatorv1.APIServiceSpec{
			Service: &kubeaggregatorv1.ServiceReference{
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
