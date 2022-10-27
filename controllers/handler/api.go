package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	check_sqllite "github.com/goodrain/rainbond-operator/util/check-sqllite"
	"github.com/goodrain/rainbond-operator/util/k8sutil"
	"github.com/sirupsen/logrus"
	utilversion "k8s.io/apimachinery/pkg/util/version"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
	"github.com/goodrain/rainbond-operator/util/commonutil"
	"github.com/goodrain/rainbond-operator/util/constants"
	"github.com/goodrain/rainbond-operator/util/probeutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
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

	pvcParametersRWX     *pvcParameters
	pvcName              string
	dataStorageRequest   int64
	grdataStorageRequest int64
}

var _ ComponentHandler = &api{}
var _ StorageClassRWXer = &api{}

//NewAPI new api handle
func NewAPI(ctx context.Context, client client.Client, component *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster) ComponentHandler {
	return &api{
		ctx:                  ctx,
		client:               client,
		component:            component,
		cluster:              cluster,
		labels:               LabelsForRainbondComponent(component),
		pvcName:              "rbd-api",
		dataStorageRequest:   getStorageRequest("API_DATA_STORAGE_REQUEST", 1),
		grdataStorageRequest: getStorageRequest("GRDATA_STORAGE_REQUEST", 40),
	}
}

func (a *api) Before() error {
	if !check_sqllite.IsSQLLite() {
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

	if a.component.Labels["persistentVolumeClaimAccessModes"] == string(corev1.ReadWriteOnce) {
		sc, err := storageClassNameFromRainbondVolumeRWO(a.ctx, a.client, a.component.Namespace)
		if err != nil {
			return err
		}
		a.SetStorageClassNameRWX(sc)
		return nil
	}
	return setStorageCassName(a.ctx, a.client, a.component.Namespace, a)
}

func (a *api) Resources() []client.Object {
	resources := a.secretAndConfigMapForAPI()
	resources = append(resources, a.deployment())
	resources = append(resources, a.createService()...)
	resources = append(resources, a.ingressForAPI())
	resources = append(resources, a.ingressForWebsocket())
	return resources
}

func (a *api) After() error {
	var url string
	url = os.Getenv("CONSOLE_DOMAIN")
	if url == "" {
		logrus.Error("CONSOLE_DOMAIN not find in env")
		return nil
	}
	var ips = strings.ReplaceAll(strings.Join(a.cluster.GatewayIngressIPs(), "-"), ".", "_")
	serverSecret, _ := a.getSecret(apiServerSecretName)
	if serverSecret != nil {
		a.serverSecret = serverSecret
		if availableips, ok := serverSecret.Labels["availableips"]; ok && availableips == ips {
			caPem := serverSecret.Data["ca.pem"]
			clientPem := serverSecret.Data["server.pem"]
			clientKey := serverSecret.Data["server.key.pem"]
			regionInfo := make(map[string]interface{})
			regionInfo["regionName"] = time.Now().Unix()
			regionInfo["regionType"] = []string{"custom"}
			regionInfo["sslCaCert"] = string(caPem)
			regionInfo["keyFile"] = string(clientKey)
			regionInfo["certFile"] = string(clientPem)
			regionInfo["url"] = fmt.Sprintf("https://%s:%s", a.cluster.GatewayIngressIP(), "8443")
			regionInfo["wsUrl"] = fmt.Sprintf("ws://%s:%s", a.cluster.GatewayIngressIP(), "6060")
			regionInfo["httpDomain"] = a.cluster.Spec.SuffixHTTPHost
			regionInfo["tcpDomain"] = a.cluster.GatewayIngressIP()
			regionInfo["desc"] = "Helm"
			regionInfo["regionAlias"] = "对接集群"
			regionInfo["provider"] = "helm"
			regionInfo["providerClusterId"] = ""

			if os.Getenv("HELM_TOKEN") != "" {
				regionInfo["token"] = os.Getenv("HELM_TOKEN")
			}
			if os.Getenv("ENTERPRISE_ID") != "" {
				regionInfo["enterpriseId"] = os.Getenv("ENTERPRISE_ID")
			}
			if os.Getenv("CLOUD_SERVER") != "" {
				cloud := os.Getenv("CLOUD_SERVER")
				switch cloud {
				case "aliyun":
					regionInfo["regionType"] = []string{"aliyun"}
				case "huawei":
					regionInfo["regionType"] = []string{"huawei"}
				case "tencent":
					regionInfo["regionType"] = []string{"tencent"}
				}
			}
			reqParam, err := json.Marshal(regionInfo)
			if err != nil {
				logrus.Error("Marshal RequestParam fail", err)
				return nil
			}
			resp, err := http.Post(url,
				"application/json",
				strings.NewReader(string(reqParam)))
			if err != nil {
				logrus.Error("request reader fail", err)
				return nil
			}
			defer resp.Body.Close()
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				logrus.Error("ReadAll error", err)
				return nil
			}
			logrus.Debug("Response body:", string(body))
		}
	}
	return nil
}

func (a *api) ListPods() ([]corev1.Pod, error) {
	return listPods(a.ctx, a.client, a.component.Namespace, a.labels)
}

func (a *api) SetStorageClassNameRWX(pvcParameters *pvcParameters) {
	a.pvcParametersRWX = pvcParameters
}

func (a *api) ResourcesCreateIfNotExists() []client.Object {
	if a.component.Labels["persistentVolumeClaimAccessModes"] == string(corev1.ReadWriteOnce) {
		return []client.Object{
			createPersistentVolumeClaimRWO(a.component.Namespace, constants.GrDataPVC, a.pvcParametersRWX, a.labels, a.grdataStorageRequest),
			createPersistentVolumeClaimRWO(a.component.Namespace, a.pvcName, a.pvcParametersRWX, a.labels, a.dataStorageRequest),
		}
	}
	return []client.Object{
		// pvc is immutable after creation except resources.requests for bound claims
		createPersistentVolumeClaimRWX(a.component.Namespace, constants.GrDataPVC, a.pvcParametersRWX, a.labels),
		createPersistentVolumeClaimRWX(a.component.Namespace, a.pvcName, a.pvcParametersRWX, a.labels),
	}
}

func (a *api) deployment() client.Object {
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "grdata",
			MountPath: "/grdata",
		},
		{
			Name:      "accesslog",
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
			Name: "accesslog",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: a.pvcName,
				},
			},
		},
	}
	args := []string{
		"--api-addr=0.0.0.0:8888",
		"--enable-feature=privileged",
		"--etcd=" + strings.Join(etcdEndpoints(a.cluster), ","),
	}
	if !check_sqllite.IsSQLLite() {
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
					ServiceAccountName: "rainbond-operator",
					Volumes:            volumes,
				},
			},
		},
	}

	return ds
}

func (a *api) createService() []client.Object {
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
			},
			Selector: a.labels,
		},
	}

	return []client.Object{svcAPI, svcWebsocket, inner}
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

	re = append(re, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "region-config",
			Namespace: a.component.Namespace,
		},
		Data: map[string]string{
			"apiAddress":          fmt.Sprintf("https://%s:%d", a.cluster.GatewayIngressIP(), 8443),
			"websocketAddress":    fmt.Sprintf("ws://%s:%d", a.cluster.GatewayIngressIP(), 6060),
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

func (a *api) ingressForAPI() client.Object {
	annotations := map[string]string{
		"nginx.ingress.kubernetes.io/l4-enable": "true",
		"nginx.ingress.kubernetes.io/l4-host":   "0.0.0.0",
		"nginx.ingress.kubernetes.io/l4-port":   "8443",
	}
	if k8sutil.GetKubeVersion().AtLeast(utilversion.MustParseSemantic("v1.19.0")) {
		logrus.Info("create networking v1 ingress for api")
		return createIngress(APIName, a.component.Namespace, annotations, a.labels, APIName+"-api", "https")
	}
	logrus.Info("create networking beta v1 ingress for api")
	return createLegacyIngress(APIName, a.component.Namespace, annotations, a.labels, APIName+"-api", intstr.FromString("https"))
}

func (a *api) ingressForWebsocket() client.Object {
	annotations := map[string]string{
		"nginx.ingress.kubernetes.io/l4-enable": "true",
		"nginx.ingress.kubernetes.io/l4-host":   "0.0.0.0",
		"nginx.ingress.kubernetes.io/l4-port":   "6060",
	}
	if k8sutil.GetKubeVersion().AtLeast(utilversion.MustParseSemantic("v1.19.0")) {
		logrus.Info("create networking v1 ingress for websocket")
		return createIngress(APIName+"-websocket", a.component.Namespace, annotations, a.labels, APIName+"-websocket", "ws")
	}
	logrus.Info("create networking beta v1 ingress for api")
	return createLegacyIngress(APIName+"-websocket", a.component.Namespace, annotations, a.labels, APIName+"-websocket", intstr.FromString("ws"))
}
