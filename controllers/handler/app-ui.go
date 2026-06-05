package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
	checksqllite "github.com/goodrain/rainbond-operator/util/check-sqllite"
	"github.com/goodrain/rainbond-operator/util/commonutil"
	"github.com/goodrain/rainbond-operator/util/rbdutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// AppUIName name for rbd-app-ui resources.
var AppUIName = "rbd-app-ui"

// AppUIDBMigrationsName -
var AppUIDBMigrationsName = "rbd-app-ui-migrations"

const (
	// appUISecretName is the Secret holding rbd-app-ui's Django SECRET_KEY.
	// The key is generated once and persisted, so it survives pod restarts,
	// node reschedules and platform upgrades — keeping issued JWTs valid.
	appUISecretName = "rbd-app-ui-secret"
	// appUISecretKey is the data key inside appUISecretName.
	appUISecretKey = "SECRET_KEY"
)

type appui struct {
	ctx              context.Context
	client           client.Client
	labels           map[string]string
	db               *rainbondv1alpha1.Database
	component        *rainbondv1alpha1.RbdComponent
	cluster          *rainbondv1alpha1.RainbondCluster
	pvcParametersRWO *pvcParameters
	storageRequest   int64
}

var _ ComponentHandler = &appui{}

// NewAppUI creates a new rbd-app-ui handler.
func NewAppUI(ctx context.Context, client client.Client, component *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster) ComponentHandler {
	return &appui{
		ctx:            ctx,
		client:         client,
		component:      component,
		labels:         LabelsForRainbondComponent(component),
		cluster:        cluster,
		storageRequest: getStorageRequest("APP_UI_DATA_STORAGE_REQUEST", 5),
	}
}

func (a *appui) Before() error {
	if !checksqllite.IsSQLLite() {
		db, err := getDefaultDBInfo(a.ctx, a.client, a.cluster.Spec.UIDatabase, a.component.Namespace, DBName)
		if err != nil {
			return fmt.Errorf("get db info: %v", err)
		}
		if db.Name == "" {
			db.Name = ConsoleDatabaseName
		}
		a.db = db
		if err := isUIDBReady(a.ctx, a.client, a.component, a.cluster); err != nil {
			return err
		}
	}
	if a.cluster.Spec.SuffixHTTPHost == "" {
		return fmt.Errorf("wait suffixHTTPHost")
	}
	if a.cluster.Spec.ImageHub == nil {
		return NewIgnoreError("image repository not ready")
	}

	return setStorageCassName(a.ctx, a.client, a.component.Namespace, a)
}

func (a *appui) SetStorageClassNameRWO(pvcParameters *pvcParameters) {
	a.pvcParametersRWO = pvcParameters
}

func (a *appui) Resources() []client.Object {
	port, ok := a.component.Labels["port"]
	if !ok {
		port = "7070"
	}
	p, err := strconv.Atoi(port)
	if err != nil {
		p = 7070
		port = "7070"
		log.Error(err, "strconv.Atoi(port)")
	}
	var res []client.Object

	// Create PVC
	appUIPVC := createPersistentVolumeClaimRWO(a.component.Namespace, "rbd-app-ui-data", a.pvcParametersRWO, a.labels, a.storageRequest)
	res = append(res, appUIPVC)

	// SECRET_KEY Secret must precede the Deployment that references it.
	res = append(res, a.secretForAppUI())

	res = append(res, a.deploymentForAppUI())
	res = append(res, a.serviceForAppUI(int32(p)))

	return res
}

func (a *appui) After() error {
	return nil
}

func (a *appui) ListPods() ([]corev1.Pod, error) {
	return listPods(a.ctx, a.client, a.component.Namespace, a.labels)
}

func (a *appui) ResourcesCreateIfNotExists() []client.Object {
	return []client.Object{
		rbdDefaultRouteTemplateForTCP("rbd-app-ui", 7070),
	}
}

func (a *appui) deploymentForAppUI() client.Object {
	cpt := a.component
	envs := []corev1.EnvVar{
		{
			Name:  "RBD_NAMESPACE",
			Value: a.component.Namespace,
		},
		{
			Name:  "CRYPTOGRAPHY_ALLOW_OPENSSL_102",
			Value: "true",
		},
		{
			Name:  "REGION_URL",
			Value: fmt.Sprintf("https://rbd-api-api:%s", rbdutil.GetenvDefault("API_PORT", "8443")),
		},
		{
			Name:  "REGION_WS_URL",
			Value: fmt.Sprintf("ws://%s:%s", a.cluster.GatewayIngressIP(), rbdutil.GetenvDefault("API_WS_PORT", "6060")),
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
		{
			Name: "SECRET_KEY",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: appUISecretName},
					Key:                  appUISecretKey,
				},
			},
		},
	}
	if !checksqllite.IsSQLLite() {
		mysqlEnvs := []corev1.EnvVar{
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
				Value: a.db.Name,
			},
		}
		envs = append(envs, mysqlEnvs...)
	}
	volumes := []corev1.Volume{
		{
			Name: "ssl",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: apiClientSecretName,
				},
			},
		},
		{
			Name: "app",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: "rbd-app-ui-data",
				},
			},
		},
	}
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "app",
			MountPath: "/app/data",
			SubPath:   "data",
		},
		{
			Name:      "ssl",
			MountPath: "/app/region/ssl",
		},
		{
			Name:      "app",
			MountPath: "/app/ui/console/migrations",
			SubPath:   "console/migrations",
		},
		{
			Name:      "app",
			MountPath: "/app/ui/www/migrations",
			SubPath:   "www/migrations",
		},
	}

	envs = mergeEnvs(envs, a.component.Spec.Env)
	volumeMounts = mergeVolumeMounts(volumeMounts, a.component.Spec.VolumeMounts)
	volumes = mergeVolumes(volumes, a.component.Spec.Volumes)
	resources := setDefaultResources(a.component.Spec.Resources)

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
					ImagePullSecrets:              imagePullSecrets(a.component, a.cluster),
					TerminationGracePeriodSeconds: commonutil.Int64(0),
					Affinity:                      a.component.Spec.Affinity,
					Containers: []corev1.Container{
						{
							Name:            AppUIName,
							Image:           cpt.Spec.Image,
							ImagePullPolicy: cpt.ImagePullPolicy(),
							Env:             envs,
							VolumeMounts:    volumeMounts,
							ReadinessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/console/config/info",
										Port: intstr.FromInt(7070),
									},
								},
								InitialDelaySeconds: 5,
								TimeoutSeconds:      3,
								PeriodSeconds:       3,
								SuccessThreshold:    1,
								FailureThreshold:    6,
							},
							Resources: resources,
							LivenessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/console/config/info",
										Port: intstr.FromInt(7070),
									},
								},
								InitialDelaySeconds: 120,
								TimeoutSeconds:      10,
								PeriodSeconds:       30,
								SuccessThreshold:    1,
								FailureThreshold:    5,
							},
						},
					},
					Volumes: volumes,
				},
			},
		},
	}

	if a.cluster.Annotations != nil {
		if enterpriseID, ok := a.cluster.Annotations["enterprise_id"]; ok {
			deploy.Spec.Template.Spec.Containers[0].Env = append(deploy.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{Name: "ENTERPRISE_ID", Value: enterpriseID})
		}
		if os.Getenv("ENTERPRISE_ID") != "" {
			deploy.Spec.Template.Spec.Containers[0].Env = append(deploy.Spec.Template.Spec.Containers[0].Env, corev1.EnvVar{Name: "ENTERPRISE_ID", Value: os.Getenv("ENTERPRISE_ID")})
		}
	}

	return deploy
}

func (a *appui) serviceForAppUI(port int32) client.Object {
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
					Port: port,
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

// secretForAppUI returns the Secret carrying rbd-app-ui's Django SECRET_KEY.
// An existing key is always reused so the value stays stable across reconciles
// and upgrades; a new cryptographically-random key is generated only when the
// Secret is absent. Stability is what keeps already-issued JWTs valid.
func (a *appui) secretForAppUI() client.Object {
	secretKey := randomSecretKey()
	if existing, err := getSecret(a.ctx, a.client, a.component.Namespace, appUISecretName); err == nil && existing != nil {
		if current := existing.Data[appUISecretKey]; len(current) > 0 {
			secretKey = string(current)
		}
	}
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      appUISecretName,
			Namespace: a.component.Namespace,
			Labels:    a.labels,
		},
		Data: map[string][]byte{
			appUISecretKey: []byte(secretKey),
		},
	}
}

// randomSecretKey returns a 256-bit cryptographically-random key, hex-encoded.
func randomSecretKey() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand failure means the host CSPRNG is unavailable; surface it
		// rather than emitting a weak all-zero key.
		log.Error(err, "generate rbd-app-ui SECRET_KEY")
		return ""
	}
	return hex.EncodeToString(b)
}
