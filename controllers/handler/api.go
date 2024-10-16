package handler

import (
	"context"
	"fmt"
	v2 "github.com/goodrain/rainbond-operator/api/v2"
	"github.com/goodrain/rainbond-operator/util/constants"
	"os"
	"strconv"
	"strings"

	checksqllite "github.com/goodrain/rainbond-operator/util/check-sqllite"
	"github.com/goodrain/rainbond-operator/util/rbdutil"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
	"github.com/goodrain/rainbond-operator/util/commonutil"
	"github.com/goodrain/rainbond-operator/util/probeutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("rbdcomponent_handler")

// APIName name
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

	dataStorageRequest   int64
	grdataStorageRequest int64
}

var _ ComponentHandler = &api{}

// NewAPI new api handle
func NewAPI(ctx context.Context, client client.Client, component *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster) ComponentHandler {
	return &api{
		ctx:                  ctx,
		client:               client,
		component:            component,
		cluster:              cluster,
		labels:               LabelsForRainbondComponent(component),
		dataStorageRequest:   getStorageRequest("API_DATA_STORAGE_REQUEST", 1),
		grdataStorageRequest: getStorageRequest("GRDATA_STORAGE_REQUEST", 20),
	}
}

func (a *api) Before() error {
	if !checksqllite.IsSQLLite() {
		db, err := getDefaultDBInfo(a.ctx, a.client, a.cluster.Spec.RegionDatabase, a.component.Namespace, DBName)
		if err != nil {
			return fmt.Errorf("get db info: %v", err)
		}
		if db.Name == "" {
			db.Name = RegionDatabaseName
		}
		a.db = db
	}

	secret, err := etcdSecret(a.ctx, a.client, a.cluster)
	if err != nil {
		return fmt.Errorf("failed to get etcd secret: %v", err)
	}
	a.etcdSecret = secret
	return nil
}

func (a *api) Resources() []client.Object {
	resources := a.secretAndConfigMapForAPI()
	resources = append(resources, a.deployment())
	resources = append(resources, a.createService()...)
	resources = append(resources, rbdDefaultRouteTemplateForTCP("rbd-api-api", 8443))
	resources = append(resources, rbdDefaultRouteTemplateForTCP("rbd-api-healthz", 8889))
	resources = append(resources, rbdDefaultRouteTemplateForTCP("rbd-api-websocket", 6060))
	resources = append(resources, a.ingressForLangProxy())
	resources = append(resources, a.upstreamForExternalDomain())
	return resources
}

func (a *api) After() error {
	return nil
}

func (a *api) ListPods() ([]corev1.Pod, error) {
	return listPods(a.ctx, a.client, a.component.Namespace, a.labels)
}

func (a *api) ResourcesCreateIfNotExists() []client.Object {
	return []client.Object{}
}

func (a *api) deployment() client.Object {
	var volumeMounts []corev1.VolumeMount
	var volumes []corev1.Volume
	args := []string{
		"--api-addr=0.0.0.0:8888",
		"--enable-feature=privileged",
	}
	if !checksqllite.IsSQLLite() {
		args = append(args, a.db.RegionDataSource())
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
			"--builder-api="+ChaosName+":3228",
			"--api-addr-ssl=0.0.0.0:8443",
			"--api-ssl-certfile=/etc/goodrain/region.goodrain.me/ssl/server.pem",
			"--api-ssl-keyfile=/etc/goodrain/region.goodrain.me/ssl/server.key.pem",
			"--client-ca-file=/etc/goodrain/region.goodrain.me/ssl/ca.pem",
		)
	}
	a.labels["name"] = APIName
	envs := []corev1.EnvVar{
		{
			Name:  "RBD_NAMESPACE",
			Value: a.component.Namespace,
		},
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
		{
			Name:  "SERVICE_ID",
			Value: "rbd-api",
		},
		{
			Name:  "LOGGER_DRIVER_NAME",
			Value: "streamlog",
		},
		{
			Name:  "HELM_TOKEN",
			Value: os.Getenv("HELM_TOKEN"),
		},
	}

	args = mergeArgs(args, a.component.Spec.Args)
	envs = mergeEnvs(envs, a.component.Spec.Env)
	volumeMounts = mergeVolumeMounts(volumeMounts, a.component.Spec.VolumeMounts)
	volumes = mergeVolumes(volumes, a.component.Spec.Volumes)

	// prepare probe
	readinessProbe := probeutil.MakeReadinessProbeHTTP("", "/v2/health", 8888)
	ds := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      APIName,
			Namespace: a.component.Namespace,
			Labels:    a.labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: a.component.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: a.labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   APIName,
					Labels: a.labels,
				},
				Spec: corev1.PodSpec{
					ImagePullSecrets:              imagePullSecrets(a.component, a.cluster),
					TerminationGracePeriodSeconds: commonutil.Int64(0),
					Containers: []corev1.Container{
						{
							Name:            APIName,
							Image:           a.component.Spec.Image,
							ImagePullPolicy: a.component.ImagePullPolicy(),
							Env:             envs,
							Args:            args,
							VolumeMounts:    volumeMounts,
							ReadinessProbe:  readinessProbe,
							Resources:       a.component.Spec.Resources,
						},
					},
					ServiceAccountName: rbdutil.GetenvDefault("SERVICE_ACCOUNT_NAME", "rainbond-operator"),
					Volumes:            volumes,
				},
			},
		},
	}

	return ds
}

func (a *api) createService() []client.Object {
	APIPort, _ := strconv.ParseInt(rbdutil.GetenvDefault("API_PORT", "8443"), 10, 32)
	APIWebsocketPort, _ := strconv.ParseInt(rbdutil.GetenvDefault("API_WS_PORT", "6060"), 10, 32)

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
					Port: int32(APIPort),
					TargetPort: intstr.IntOrString{
						IntVal: int32(APIPort),
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
					Port: int32(APIWebsocketPort),
					TargetPort: intstr.IntOrString{
						IntVal: int32(APIWebsocketPort),
					},
				},
			},
			Selector: a.labels,
		},
	}

	inner := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      APIName + "-api-inner",
			Namespace: a.component.Namespace,
			Labels:    a.labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name: "https",
					Port: 8888,
					TargetPort: intstr.IntOrString{
						IntVal: 8888,
					},
				},
				{
					Name: "eventlog-grpc-server",
					Port: 6366,
				},
			},
			Selector: a.labels,
		},
	}

	healthz := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      APIName + "-healthz",
			Namespace: a.component.Namespace,
			Labels:    a.labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name: "healthz",
					Port: 8889,
					TargetPort: intstr.IntOrString{
						IntVal: 8889,
					},
				},
			},
			Selector: a.labels,
		},
	}

	return []client.Object{svcAPI, svcWebsocket, inner, healthz}
}

func (a *api) getSecret(name string) (*corev1.Secret, error) {
	return getSecret(a.ctx, a.client, a.component.Namespace, name)
}
func (a *api) secretAndConfigMapForAPI() []client.Object {
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
	var re []client.Object
	labels := copyLabels(a.labels)
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

	APIPort, _ := strconv.ParseInt(rbdutil.GetenvDefault("API_PORT", "8443"), 10, 64)
	APIWebsocketPort, _ := strconv.ParseInt(rbdutil.GetenvDefault("API_WS_PORT", "6060"), 10, 64)
	re = append(re, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "region-config",
			Namespace: a.component.Namespace,
		},
		Data: map[string]string{
			"apiAddress":          fmt.Sprintf("https://%s:%d", a.cluster.GatewayIngressIP(), APIPort),
			"websocketAddress":    fmt.Sprintf("ws://%s:%d", a.cluster.GatewayIngressIP(), APIWebsocketPort),
			"defaultDomainSuffix": a.cluster.Spec.SuffixHTTPHost,
			"defaultTCPHost":      a.cluster.GatewayIngressIP(),
		},
		BinaryData: map[string][]byte{
			"client.pem":     clientPem,
			"client.key.pem": clientKey,
			"ca.pem":         caPem,
		},
	})
	return re
}

func (a *api) ingressForLangProxy() client.Object {
	weight := 1
	return &v2.ApisixRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "lang-proxy",
			Namespace: a.component.Namespace,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       constants.ApisixRoute,
			APIVersion: constants.APISixAPIVersion,
		},
		Spec: v2.ApisixRouteSpec{
			HTTP: []v2.ApisixRouteHTTP{
				{
					Name: "lang-proxy",
					Match: v2.ApisixRouteHTTPMatch{
						Hosts: []string{
							"lang.goodrain.me",
							"maven.goodrain.me",
						},
						Paths: []string{
							"/*",
						},
					},
					Upstreams: []v2.ApisixRouteUpstreamReference{
						{
							Name:   "buildpack-upstream",
							Weight: &weight,
						},
					},
					Plugins: []v2.ApisixRoutePlugin{
						{
							Name:   "proxy-rewrite",
							Enable: true,
							Config: v2.ApisixRoutePluginConfig{
								"scheme": "https",
								"host":   "buildpack.oss-cn-shanghai.aliyuncs.com",
							},
						},
					},
				},
			},
		},
	}
}

func (a *api) upstreamForExternalDomain() *v2.ApisixUpstream {
	port := 443 // HTTPS 默认端口
	return &v2.ApisixUpstream{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "buildpack-upstream",
			Namespace: a.component.Namespace,
		},
		Spec: &v2.ApisixUpstreamSpec{
			ExternalNodes: []v2.ApisixUpstreamExternalNode{
				{
					Name: "buildpack.oss-cn-shanghai.aliyuncs.com", // 外部服务地址
					Type: "Domain",
					Port: &port, // 端口
				},
			},
			ApisixUpstreamConfig: v2.ApisixUpstreamConfig{
				Scheme: "https",
			},
		},
	}
}
