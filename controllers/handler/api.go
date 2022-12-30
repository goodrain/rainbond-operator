package handler

import (
	"context"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/wutong-paas/wutong-operator/util/k8sutil"
	utilversion "k8s.io/apimachinery/pkg/util/version"

	wutongv1alpha1 "github.com/wutong-paas/wutong-operator/api/v1alpha1"
	"github.com/wutong-paas/wutong-operator/util/commonutil"
	"github.com/wutong-paas/wutong-operator/util/constants"
	"github.com/wutong-paas/wutong-operator/util/probeutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("wutongcomponent_handler")

// APIName name
var APIName = "wt-api"
var apiServerSecretName = "wt-api-server-cert"
var apiCASecretName = "wt-api-ca-cert"
var apiClientSecretName = "wt-api-client-cert"

type api struct {
	ctx                      context.Context
	client                   client.Client
	db                       *wutongv1alpha1.Database
	labels                   map[string]string
	etcdSecret, serverSecret *corev1.Secret
	component                *wutongv1alpha1.WutongComponent
	cluster                  *wutongv1alpha1.WutongCluster

	pvcParametersRWX     *pvcParameters
	pvcName              string
	dataStorageRequest   int64
	wtdataStorageRequest int64
}

var _ ComponentHandler = &api{}
var _ StorageClassRWXer = &api{}

// NewAPI new api handle
func NewAPI(ctx context.Context, client client.Client, component *wutongv1alpha1.WutongComponent, cluster *wutongv1alpha1.WutongCluster) ComponentHandler {
	return &api{
		ctx:                  ctx,
		client:               client,
		component:            component,
		cluster:              cluster,
		labels:               LabelsForWutongComponent(component),
		pvcName:              "wt-api",
		dataStorageRequest:   getStorageRequest("API_DATA_STORAGE_REQUEST", 1),
		wtdataStorageRequest: getStorageRequest("WTDATA_STORAGE_REQUEST", 40),
	}
}

func (a *api) Before() error {
	db, err := getDefaultDBInfo(a.ctx, a.client, a.cluster.Spec.RegionDatabase, a.component.Namespace, DBName)
	if err != nil {
		return fmt.Errorf("get db info: %v", err)
	}
	if db.Name == "" {
		db.Name = RegionDatabaseName
	}
	a.db = db

	secret, err := etcdSecret(a.ctx, a.client, a.cluster)
	if err != nil {
		return fmt.Errorf("failed to get etcd secret: %v", err)
	}
	a.etcdSecret = secret

	if a.component.Labels["persistentVolumeClaimAccessModes"] == string(corev1.ReadWriteOnce) {
		sc, err := storageClassNameFromWutongVolumeRWO(a.ctx, a.client, a.component.Namespace)
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
	resources = append(resources, a.ingressForAPIHealthz())
	return resources
}

func (a *api) After() error {
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
			createPersistentVolumeClaimRWO(a.component.Namespace, constants.WTDataPVC, a.pvcParametersRWX, a.labels, a.wtdataStorageRequest),
			createPersistentVolumeClaimRWO(a.component.Namespace, a.pvcName, a.pvcParametersRWX, a.labels, a.dataStorageRequest),
		}
	}
	return []client.Object{
		// pvc is immutable after creation except resources.requests for bound claims
		createPersistentVolumeClaimRWX(a.component.Namespace, constants.WTDataPVC, a.pvcParametersRWX, a.labels, a.wtdataStorageRequest),
		createPersistentVolumeClaimRWX(a.component.Namespace, a.pvcName, a.pvcParametersRWX, a.labels, a.dataStorageRequest),
	}
}

func (a *api) deployment() client.Object {
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "wtdata",
			MountPath: "/wtdata",
		},
		{
			Name:      "accesslog",
			MountPath: "/logs",
		},
	}
	volumes := []corev1.Volume{
		{
			Name: "wtdata",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: constants.WTDataPVC,
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
			"--builder-api="+ChaosName+":3228",
			"--api-addr-ssl=0.0.0.0:8443",
			"--api-ssl-certfile=/etc/wutong/region.wutong.me/ssl/server.pem",
			"--api-ssl-keyfile=/etc/wutong/region.wutong.me/ssl/server.key.pem",
			"--client-ca-file=/etc/wutong/region.wutong.me/ssl/ca.pem",
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
					ServiceAccountName: "wutong-operator",
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
	//wt-api-api domain support in cluster
	serverPem, serverKey, err := ca.CreateCert(a.cluster.GatewayIngressIPs(), "wt-api-api")
	if err != nil {
		log.Error(err, "create serverSecret cert for api")
		return nil
	}
	clientPem, clientKey, err := ca.CreateCert(a.cluster.GatewayIngressIPs(), "wt-api-api")
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

func (a *api) ingressForAPIHealthz() client.Object {
	annotations := map[string]string{
		"nginx.ingress.kubernetes.io/l4-enable": "true",
		"nginx.ingress.kubernetes.io/l4-host":   "0.0.0.0",
		"nginx.ingress.kubernetes.io/l4-port":   "8889",
	}
	if k8sutil.GetKubeVersion().AtLeast(utilversion.MustParseSemantic("v1.19.0")) {
		logrus.Info("create networking v1 ingress for healthz")
		return createIngress(APIName+"-healthz", a.component.Namespace, annotations, a.labels, APIName+"-healthz", "healthz")
	}
	logrus.Info("create networking beta v1 ingress for healthz")
	return createLegacyIngress(APIName+"-healthz", a.component.Namespace, annotations, a.labels, APIName+"-api-inner", intstr.FromString("healthz"))
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
	logrus.Info("create networking beta v1 ingress for websocket")
	return createLegacyIngress(APIName+"-websocket", a.component.Namespace, annotations, a.labels, APIName+"-websocket", intstr.FromString("ws"))
}
