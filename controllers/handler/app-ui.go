package handler

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"runtime"
	"strconv"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
	checksqllite "github.com/goodrain/rainbond-operator/util/check-sqllite"
	"github.com/goodrain/rainbond-operator/util/commonutil"
	"github.com/goodrain/rainbond-operator/util/rbdutil"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
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
		storageRequest: getStorageRequest("APP_UI_DATA_STORAGE_REQUEST", 1),
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
			Name:  "SECRET_KEY",
			Value: getHashMac(),
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
							Resources: a.component.Spec.Resources,
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

func getSystemInfo() string {
	// 获取操作系统信息
	os := runtime.GOOS

	// 获取系统架构
	arch := runtime.GOARCH

	// 获取 CPU 核心数
	cpuCount := runtime.NumCPU()

	// 获取 CPU 相关信息
	cpus, err := cpu.Info()
	if err != nil {
		fmt.Println("Error getting CPU info:", err)
		return ""
	}

	// 获取内存信息
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		fmt.Println("Error getting memory info:", err)
		return ""
	}

	// 获取总内存（GB）
	totalMemory := vmStat.Total / (1024 * 1024 * 1024) // 转换为 GB

	// 将系统信息拼接成一个字符串
	systemInfo := fmt.Sprintf("OS:%s-Arch:%s-CPUs:%d-Mem:%dGB-CPU:%s",
		os,
		arch,
		cpuCount,
		totalMemory,
		cpus[0].ModelName)

	return systemInfo
}

// 计算MD5值
func calculateMD5(input string) string {
	hash := md5.New()
	hash.Write([]byte(input))
	hashInBytes := hash.Sum(nil)
	return hex.EncodeToString(hashInBytes)
}

// 获取系统信息并计算 MD5 值
func getHashMac() string {
	// 获取系统信息
	systemInfo := getSystemInfo()
	// 计算系统信息的 MD5 值
	md5Value := calculateMD5(systemInfo)
	// 返回一个包含 "SECRET_KEY" 和 MD5 值的映射
	return md5Value
}
