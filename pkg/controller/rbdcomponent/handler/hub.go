package handler

import (
	"context"
	"github.com/goodrain/rainbond-operator/pkg/util/constants"

	"github.com/goodrain/rainbond-operator/pkg/util/rbdutil"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/goodrain/rainbond-operator/pkg/util/commonutil"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//HubName name
var HubName = "rbd-hub"
var hubDataPvcName = "rbd-hub"
var hubImageRepository = "hub-image-repository"

type hub struct {
	ctx       context.Context
	client    client.Client
	component *rainbondv1alpha1.RbdComponent
	cluster   *rainbondv1alpha1.RainbondCluster
	pkg       *rainbondv1alpha1.RainbondPackage
	labels    map[string]string

	storageClassNameRWX string
}

var _ ComponentHandler = &hub{}
var _ StorageClassRWXer = &hub{}

//NewHub nw hub
func NewHub(ctx context.Context, client client.Client, component *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster, pkg *rainbondv1alpha1.RainbondPackage) ComponentHandler {
	return &hub{
		component: component,
		cluster:   cluster,
		client:    client,
		ctx:       ctx,
		pkg:       pkg,
		labels:    LabelsForRainbondComponent(component),
	}
}

func (h *hub) Before() error {
	if h.cluster.Spec.ImageHub != nil {
		return NewIgnoreError("use custom image repository")
	}

	if err := setStorageCassName(h.ctx, h.client, h.component.Namespace, h); err != nil {
		return err
	}

	return nil
}

func (h *hub) Resources() []interface{} {
	return []interface{}{
		h.secretForHub(), // important! create secret before ingress.
		h.deployment(),
		h.serviceForHub(),
		h.persistentVolumeClaimForHub(),
		h.ingressForHub(),
	}
}

func (h *hub) After() error {
	return nil
}

func (h *hub) SetStorageClassNameRWX(sc string) {
	h.storageClassNameRWX = sc
}

func (h *hub) deployment() interface{} {
	ds := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      HubName,
			Namespace: h.component.Namespace,
			Labels:    h.labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: h.component.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: h.labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   HubName,
					Labels: h.labels,
				},
				Spec: corev1.PodSpec{
					TerminationGracePeriodSeconds: commonutil.Int64(0),
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
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      HubName,
			Namespace: h.component.Namespace,
			Labels:    h.labels,
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
			Selector: h.labels,
		},
	}

	return svc
}

func (h *hub) persistentVolumeClaimForHub() interface{} {
	storageClassName := rbdutil.GetStorageClass(h.cluster)
	storageRequest := resource.NewQuantity(21*1024*1024*1024, resource.BinarySI) // TODO: customer specified
	if storageClassName == constants.DefStorageClass {
		storageRequest = resource.NewQuantity(1*1024*1024, resource.BinarySI)
	}
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
			StorageClassName: commonutil.String(h.storageClassNameRWX),
		},
	}

	return pvc
}

func (h *hub) ingressForHub() interface{} {
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
			Labels: h.labels,
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
					Hosts:      []string{rbdutil.GetImageRepository(h.cluster)},
					SecretName: hubImageRepository,
				},
			},
		},
	}

	return ing
}

func (h *hub) secretForHub() interface{} {
	secret, _ := h.getSecret(hubImageRepository)
	if secret != nil {
		return nil
	}
	labels := h.labels
	labels["name"] = hubImageRepository
	_, pem, key, _ := commonutil.DomainSign(nil, rbdutil.GetImageRepository(h.cluster))
	secret = &corev1.Secret{
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

func (h *hub) getSecret(name string) (*corev1.Secret, error) {
	return getSecret(h.ctx, h.client, h.component.Namespace, name)
}
