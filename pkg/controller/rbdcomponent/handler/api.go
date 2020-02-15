package handler

import (
	"context"
	"fmt"
	"strings"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/goodrain/rainbond-operator/pkg/util/commonutil"
	"github.com/goodrain/rainbond-operator/pkg/util/constants"
	"github.com/goodrain/rainbond-operator/pkg/util/k8sutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("rbdcomponent_handler")

//APIName name
var APIName = "rbd-api"
var apiServerSecretName = "rbd-api-server-cert"
var apiCASecretName = "rbd-api-ca-cert"
var apiClientSecretName = "rbd-api-client-cert"

type api struct {
	ctx                      context.Context
	client                   client.Client
	db                       *rainbondv1alpha1.Database
	labels                   map[string]string
	etcdSecret, serverSecret *corev1.Secret
	component                *rainbondv1alpha1.RbdComponent
	cluster                  *rainbondv1alpha1.RainbondCluster
	pkg                      *rainbondv1alpha1.RainbondPackage
}

//NewAPI new api handle
func NewAPI(ctx context.Context, client client.Client, component *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster, pkg *rainbondv1alpha1.RainbondPackage) ComponentHandler {
	return &api{
		ctx:       ctx,
		client:    client,
		component: component,
		cluster:   cluster,
		labels:    component.GetLabels(),
		pkg:       pkg,
	}
}

func (a *api) Before() error {
	db, err := getDefaultDBInfo(a.ctx, a.client, a.cluster.Spec.RegionDatabase, a.component.Namespace, DBName)
	if err != nil {
		return fmt.Errorf("get db info: %v", err)
	}
	a.db = db

	secret, err := etcdSecret(a.ctx, a.client, a.cluster)
	if err != nil {
		return fmt.Errorf("failed to get etcd secret: %v", err)
	}
	a.etcdSecret = secret

	return nil
}

func (a *api) Resources() []interface{} {
	resources := a.secretForAPI()
	resources = append(resources, a.daemonSetForAPI())
	resources = append(resources, a.createService()...)
	resources = append(resources, a.ingressForAPI())
	resources = append(resources, a.ingressForWebsocket())
	return resources
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
		{
			Name:      "accessLog",
			MountPath: "/logs",
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
		{
			Name: "accessLog",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/opt/rainbond/logs/rbd-api/",
					Type: k8sutil.HostPath(corev1.HostPathDirectoryOrCreate),
				},
			},
		},
	}
	args := []string{
		"--api-addr=127.0.0.1:8888",
		"--enable-feature=privileged",
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
	if a.serverSecret != nil {
		volume, mount := volumeByAPISecret(a.serverSecret)
		volumeMounts = append(volumeMounts, mount)
		volumes = append(volumes, volume)
		args = append(args, "--api-ssl-enable=true",
			"--api-addr-ssl=0.0.0.0:8443",
			"--api-ssl-certfile=/etc/goodrain/region.goodrain.me/ssl/server.pem",
			"--api-ssl-keyfile=/etc/goodrain/region.goodrain.me/ssl/server.key.pem",
			"--client-ca-file=/etc/goodrain/region.goodrain.me/ssl/ca.pem",
		)
	}
	a.labels["name"] = APIName
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
					TerminationGracePeriodSeconds: commonutil.Int64(0),
					NodeSelector:                  a.cluster.Status.FirstMasterNodeLabel(),
					Tolerations: []corev1.Toleration{
						{
							Key:    a.cluster.Status.MasterRoleLabel,
							Effect: corev1.TaintEffectNoSchedule,
						},
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

func (a *api) createService() []interface{} {
	svcAPI := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      APIName + "-api",
			Namespace: a.component.Namespace,
			Labels:    a.labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name: "https",
					Port: 8443,
					TargetPort: intstr.IntOrString{
						IntVal: 8443,
					},
				},
			},
			Selector: a.labels,
		},
	}

	svcWebsocket := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      APIName + "-websocket",
			Namespace: a.component.Namespace,
			Labels:    a.labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
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

	return []interface{}{svcAPI, svcWebsocket}
}

func (a *api) getSecret(name string) (*corev1.Secret, error) {
	return getSecret(a.ctx, a.client, a.component.Namespace, name)
}
func (a *api) secretForAPI() []interface{} {
	var ips = strings.ReplaceAll(strings.Join(a.cluster.GatewayIngressIPs(), "-"), ".", "_")
	serverSecret, _ := a.getSecret(apiServerSecretName)
	var ca *commonutil.CA
	var err error
	if serverSecret != nil {
		a.serverSecret = serverSecret
		//no change,do nothing
		if availableips, ok := serverSecret.Labels["availableips"]; ok && availableips == ips {
			return nil
		}
		caSecret, _ := a.getSecret(apiCASecretName)
		if caSecret != nil {
			ca, err = commonutil.ParseCA(caSecret.Data["ca.pem"], caSecret.Data["ca.key.pem"])
			if err != nil {
				log.Error(err, "parse ca for api")
				return nil
			}
		}
	}
	if ca == nil {
		ca, err = commonutil.CreateCA()
		if err != nil {
			log.Error(err, "create ca for api")
			return nil
		}
	}
	//rbd-api-api domain support in cluster
	serverPem, serverKey, err := ca.CreateCert(a.cluster.GatewayIngressIPs(), "rbd-api-api")
	if err != nil {
		log.Error(err, "create serverSecret cert for api")
		return nil
	}
	clientPem, clientKey, err := ca.CreateCert(a.cluster.GatewayIngressIPs(), "rbd-api-api")
	if err != nil {
		log.Error(err, "create client cert for api")
		return nil
	}
	caPem, err := ca.GetCAPem()
	if err != nil {
		log.Error(err, "create ca pem for api")
		return nil
	}
	var re []interface{}
	labels := a.component.GetLabels()
	labels["availableips"] = ips
	server := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      apiServerSecretName,
			Namespace: a.component.Namespace,
			Labels:    labels,
		},
		Data: map[string][]byte{
			"server.pem":     serverPem,
			"server.key.pem": serverKey,
			"ca.pem":         caPem,
		},
	}
	a.serverSecret = server
	re = append(re, server)
	re = append(re, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      apiClientSecretName,
			Namespace: a.component.Namespace,
			Labels:    labels,
		},
		Data: map[string][]byte{
			"client.pem":     clientPem,
			"client.key.pem": clientKey,
			"ca.pem":         caPem,
		},
	})
	return re
}

func (a *api) ingressForAPI() interface{} {
	ing := &extensions.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      APIName,
			Namespace: a.component.Namespace,
			Annotations: map[string]string{
				"nginx.ingress.kubernetes.io/l4-enable": "true",
				"nginx.ingress.kubernetes.io/l4-host":   "0.0.0.0",
				"nginx.ingress.kubernetes.io/l4-port":   "8443",
			},
			Labels: a.labels,
		},
		Spec: extensions.IngressSpec{
			Backend: &extensions.IngressBackend{
				ServiceName: APIName + "-api",
				ServicePort: intstr.FromString("https"),
			},
		},
	}

	return ing
}

func (a *api) ingressForWebsocket() interface{} {
	ing := &extensions.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      APIName + "-websocket",
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
				ServiceName: APIName + "-websocket",
				ServicePort: intstr.FromString("ws"),
			},
		},
	}
	return ing
}
