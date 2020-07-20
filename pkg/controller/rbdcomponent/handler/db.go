package handler

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/docker/distribution/reference"
	mysqlv1alpha1 "github.com/oracle/mysql-operator/pkg/apis/mysql/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	"sigs.k8s.io/controller-runtime/pkg/client"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/goodrain/rainbond-operator/pkg/util/commonutil"
)

var (
	// DBName name
	DBName           = "rbd-db"
	dbhost           = DBName + "-rw"
	mycnf            = DBName + "-mycnf"
	mysqlUser        = "root"
	mysqlUserKey     = "mysql-user"
	mysqlPasswordKey = "mysql-password"
)

type db struct {
	ctx       context.Context
	client    client.Client
	component *rainbondv1alpha1.RbdComponent
	cluster   *rainbondv1alpha1.RainbondCluster
	labels    map[string]string

	secret                   *corev1.Secret
	mysqlUser, mysqlPassword string
	enableMysqlOperator      bool
	databases                []string

	pvcParametersRWO *pvcParameters
	storageRequest   int64
}

var _ ComponentHandler = &db{}
var _ StorageClassRWOer = &db{}

//NewDB new db
func NewDB(ctx context.Context, client client.Client, component *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster) ComponentHandler {
	d := &db{
		ctx:            ctx,
		client:         client,
		component:      component,
		cluster:        cluster,
		labels:         LabelsForRainbondComponent(component),
		mysqlUser:      "root",
		mysqlPassword:  string(uuid.NewUUID())[0:8],
		databases:      []string{"console"},
		storageRequest: getStorageRequest("DB_DATA_STORAGE_REQUEST", 21),
	}
	regionDBName := os.Getenv("REGION_DB_NAME")
	if regionDBName == "" {
		regionDBName = "region"
	}
	d.databases = append(d.databases, regionDBName)
	return d
}

func (d *db) Before() error {
	if d.cluster.Spec.RegionDatabase != nil && d.cluster.Spec.UIDatabase != nil {
		return NewIgnoreError("use custom database")
	}

	secret := &corev1.Secret{}
	if err := d.client.Get(d.ctx, types.NamespacedName{Namespace: d.component.Namespace, Name: DBName}, secret); err != nil {
		if !k8sErrors.IsNotFound(err) {
			return fmt.Errorf("get secret %s/%s: %v", DBName, d.component.Namespace, err)
		}
		secret = nil
	}
	d.secret = secret
	if d.secret != nil {
		// use the old password
		d.mysqlUser = string(d.secret.Data[mysqlUserKey])
		d.mysqlPassword = string(d.secret.Data[mysqlPasswordKey])
	}

	if err := setStorageCassName(d.ctx, d.client, d.component.Namespace, d); err != nil {
		return err
	}

	return nil
}

func (d *db) Resources() []interface{} {
	if d.enableMysqlOperator {
		return []interface{}{
			d.secretForDB(),
			d.serviceForMysqlCluster(),
			d.configMapForMyCnf(),
			d.mysqlCluster(),
		}
	}

	return []interface{}{
		d.secretForDB(),
		d.configMapForMyCnf(),
		d.initdbCMForDB(),
		d.statefulsetForDB(),
		d.serviceForDB(),
		d.serviceForExporter(),
	}
}

func (d *db) After() error {
	return nil
}

func (d *db) ListPods() ([]corev1.Pod, error) {
	labels := d.labels
	if d.enableMysqlOperator {
		labels = map[string]string{
			"v1alpha1.mysql.oracle.com/cluster": DBName,
		}
	}
	return listPods(d.ctx, d.client, d.component.Namespace, labels)
}

func (d *db) SetStorageClassNameRWO(pvcParameters *pvcParameters) {
	d.pvcParametersRWO = pvcParameters
}

func (d *db) Replicas() *int32 {
	if !d.enableMysqlOperator {
		commonutil.Int32(1)
	}
	return nil
}

func (d *db) statefulsetForDB() interface{} {
	repo, _ := reference.Parse(d.component.Spec.Image)
	name := repo.(reference.Named).Name()
	exporterImage := strings.Replace(name, "rbd-db", "mysqld-exporter", 1)

	claimName := "data"
	pvc := createPersistentVolumeClaimRWO(d.component.Namespace, claimName, d.pvcParametersRWO, d.labels, d.storageRequest)

	regionDBName := os.Getenv("REGION_DB_NAME")
	if regionDBName == "" {
		regionDBName = "region"
	}

	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      DBName,
			Namespace: d.component.Namespace,
			Labels:    d.labels,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: commonutil.Int32(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: d.labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   DBName,
					Labels: d.labels,
				},
				Spec: corev1.PodSpec{
					ImagePullSecrets:              imagePullSecrets(d.component, d.cluster),
					TerminationGracePeriodSeconds: commonutil.Int64(0),
					Containers: []corev1.Container{
						{
							Name:            DBName,
							Image:           d.component.Spec.Image,
							ImagePullPolicy: d.component.ImagePullPolicy(),
							Env: []corev1.EnvVar{
								{
									Name:  "MYSQL_ROOT_HOST",
									Value: "%",
								},
								{
									Name:  "MYSQL_LOG_CONSOLE",
									Value: "true",
								},
								{
									Name: "MYSQL_ROOT_PASSWORD",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: DBName,
											},
											Key:      mysqlPasswordKey,
											Optional: commonutil.Bool(true),
										},
									},
								},
								{
									Name:  "MYSQL_DATABASE",
									Value: regionDBName,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      claimName,
									MountPath: "/var/lib/mysql",
									SubPath:   "mysql",
								},
								{
									Name:      "initdb",
									MountPath: "/docker-entrypoint-initdb.d",
								},
								{
									Name:      mycnf,
									MountPath: "/etc/my.cnf",
									SubPath:   "my.cnf",
								},
							},
							ReadinessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									Exec: &corev1.ExecAction{Command: []string{"mysql", "-u" + d.mysqlUser, "-p" + d.mysqlPassword, "-e", "SELECT 1"}},
								},
								InitialDelaySeconds: 5,
								PeriodSeconds:       2,
								TimeoutSeconds:      1,
							},
						},
						{
							Name:            DBName + "-exporter",
							Image:           exporterImage,
							ImagePullPolicy: d.component.ImagePullPolicy(),
							Env: []corev1.EnvVar{
								{
									Name:  "DATA_SOURCE_NAME",
									Value: fmt.Sprintf("%s:%s@tcp(127.0.0.1:3306)/", d.mysqlUser, d.mysqlPassword),
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "initdb",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "rbd-db-initdb",
									},
								},
							},
						},
						{
							Name: mycnf,
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: mycnf,
									},
								},
							},
						},
					},
				},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{*pvc},
		},
	}

	return sts
}

func (d *db) serviceForDB() interface{} {
	mysqlSvc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dbhost,
			Namespace: d.component.Namespace,
			Labels:    d.labels,
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "None",
			Ports: []corev1.ServicePort{
				{
					Name: "main",
					Port: 3306,
				},
			},
			Selector: d.labels,
		},
	}
	return mysqlSvc
}

func (d *db) serviceForExporter() interface{} {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      DBName + "-exporter",
			Namespace: d.component.Namespace,
			Labels:    d.labels,
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "None",
			Ports: []corev1.ServicePort{
				{
					Name: "exporter",
					Port: 9104,
				},
			},
			Selector: d.labels,
		},
	}
}

func (d *db) initdbCMForDB() interface{} {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rbd-db-initdb",
			Namespace: d.component.Namespace,
		},
		Data: map[string]string{
			"initdb.sql": `
CREATE DATABASE IF NOT EXISTS console;
`,
		},
	}

	return cm
}

func (d *db) secretForDB() interface{} {
	if d.secret != nil {
		return nil
	}
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      DBName,
			Namespace: d.component.Namespace,
		},
		StringData: map[string]string{
			"password":       d.mysqlPassword,
			mysqlPasswordKey: d.mysqlPassword,
			mysqlUserKey:     d.mysqlUser,
		},
	}
}

func (d *db) mysqlCluster() *mysqlv1alpha1.Cluster {
	claimName := "data"
	pvc := createPersistentVolumeClaimRWO(d.component.Namespace, claimName, d.pvcParametersRWO, d.labels, d.storageRequest)

	var defaultSize int32 = 1
	if d.component.Spec.Replicas != nil {
		defaultSize = *d.component.Spec.Replicas
	}

	// make sure the image name is right
	repo, _ := reference.Parse(d.component.Spec.Image)
	named := repo.(reference.Named)
	tag := "latest"
	if t, ok := repo.(reference.Tagged); ok {
		tag = t.Tag()
	}
	affinity := &corev1.Affinity{
		PodAntiAffinity: &corev1.PodAntiAffinity{
			PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
				{
					Weight: 100,
					PodAffinityTerm: corev1.PodAffinityTerm{
						LabelSelector: &metav1.LabelSelector{
							MatchExpressions: []metav1.LabelSelectorRequirement{
								{
									Key:      "v1alpha1.mysql.oracle.com/cluster",
									Operator: metav1.LabelSelectorOpIn,
									Values:   []string{DBName},
								},
							},
						},
						TopologyKey: "kubernetes.io/hostname",
					},
				},
			},
		},
	}

	mc := &mysqlv1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      DBName,
			Namespace: d.component.Namespace,
			Labels:    d.labels,
		},
		Spec: mysqlv1alpha1.ClusterSpec{
			MultiMaster:      true,
			Members:          defaultSize,
			ImagePullSecrets: imagePullSecrets(d.component, d.cluster),
			RootPasswordSecret: &corev1.LocalObjectReference{
				Name: DBName,
			},
			Repository:          named.Name(),
			Version:             tag,
			VolumeClaimTemplate: pvc,
			Config: &corev1.LocalObjectReference{
				Name: mycnf,
			},
			Affinity: affinity,
		},
	}
	return mc
}

func (d *db) serviceForMysqlCluster() interface{} {
	labels := copyLabels(d.labels)
	labels["v1alpha1.mysql.oracle.com/cluster"] = DBName
	selector := map[string]string{
		"v1alpha1.mysql.oracle.com/cluster": DBName,
	}
	if d.enableMysqlOperator && d.component.Spec.Replicas != nil && *d.component.Spec.Replicas > 1 {
		selector["v1alpha1.mysql.oracle.com/role"] = "primary"
	}
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dbhost,
			Namespace: d.component.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "None",
			Ports: []corev1.ServicePort{
				{
					Name: "main",
					Port: 3306,
				},
			},
			Selector: selector,
		},
	}
	return svc
}

func (d *db) configMapForMyCnf() interface{} {
	var innodbDirs []string
	for _, database := range d.databases {
		innodbDirs = append(innodbDirs, "/var/lib/mysql/"+database)
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      mycnf,
			Namespace: d.component.Namespace,
		},
		Data: map[string]string{
			"my.cnf": fmt.Sprintf(`
[client]
# Default is Latin1, if you need UTF-8 set this (also in server section)
default-character-set = utf8mb4

[mysqld]
user=mysql
innodb_directories="%s"

#
# * Character sets
#
# Default is Latin1, if you need UTF-8 set all this (also in client section)
#
character-set-server  = utf8mb4
collation-server      = utf8mb4_unicode_ci
character_set_server   = utf8mb4
collation_server       = utf8mb4_unicode_ci

# Compatible with versions before 8.0
default_authentication_plugin=mysql_native_password
skip-host-cache
skip-name-resolve
`, strings.Join(innodbDirs, ";")),
		},
	}

	return cm
}
