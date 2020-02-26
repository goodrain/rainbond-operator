package handler

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	"sigs.k8s.io/controller-runtime/pkg/client"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/goodrain/rainbond-operator/pkg/util/commonutil"
	"github.com/goodrain/rainbond-operator/pkg/util/k8sutil"
)

// DBName name
var DBName = "rbd-db"
var mysqlUser = "write"
var mysqlUserKey = "mysql-user"
var mysqlPasswordKey = "mysql-password"

type db struct {
	ctx                      context.Context
	client                   client.Client
	component                *rainbondv1alpha1.RbdComponent
	cluster                  *rainbondv1alpha1.RainbondCluster
	labels                   map[string]string
	secret                   *corev1.Secret
	mysqlUser, mysqlPassword string
}

//NewDB new db
func NewDB(ctx context.Context, client client.Client, component *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster) ComponentHandler {
	return &db{
		ctx:           ctx,
		client:        client,
		component:     component,
		cluster:       cluster,
		labels:        LabelsForRainbondComponent(component),
		mysqlUser:     mysqlUser,
		mysqlPassword: string(uuid.NewUUID())[0:8],
	}
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

	return nil
}

func (d *db) Resources() []interface{} {
	return []interface{}{
		d.secretForDB(),
		d.statefulsetForDB(),
		d.serviceForDB(),
		d.serviceForExporter(),
		d.configMapForDB(),
		d.initdbCMForDB(),
	}
}

func (d *db) After() error {
	return nil
}

func (d *db) statefulsetForDB() interface{} {
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
					NodeSelector: d.cluster.Status.FirstMasterNodeLabel(),
					Tolerations: []corev1.Toleration{
						{
							Key:    d.cluster.Status.MasterRoleLabel,
							Effect: corev1.TaintEffectNoSchedule,
						},
					},
					TerminationGracePeriodSeconds: commonutil.Int64(0),
					Containers: []corev1.Container{
						{
							Name:            DBName,
							Image:           d.component.Spec.Image,
							ImagePullPolicy: d.component.ImagePullPolicy(),
							Env: []corev1.EnvVar{
								{
									Name:  "MYSQL_ALLOW_EMPTY_PASSWORD",
									Value: "yes",
								},
								{
									Name: "MYSQL_USER",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: DBName,
											},
											Key:      mysqlUserKey,
											Optional: commonutil.Bool(true),
										},
									},
								},
								{
									Name: "MYSQL_PASSWORD",
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
									Value: "region",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "rbd-db-data",
									MountPath: "/data",
								},
								{
									Name:      "initdb",
									MountPath: "/docker-entrypoint-initdb.d",
								},
							},
							LivenessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									Exec: &corev1.ExecAction{Command: []string{"mysqladmin", "ping"}},
								},
								InitialDelaySeconds: 30,
								PeriodSeconds:       10,
								TimeoutSeconds:      5,
							},
							ReadinessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									Exec: &corev1.ExecAction{Command: []string{"mysql", "-e", "SELECT 1"}},
								},
								InitialDelaySeconds: 5,
								PeriodSeconds:       2,
								TimeoutSeconds:      1,
							},
						},
						{
							Name:            DBName + "-exporter",
							Image:           "goodrain.me/mysqld-exporter",
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
							Name: "rbd-db-data",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/opt/rainbond/data/db",
									Type: k8sutil.HostPath(corev1.HostPathDirectoryOrCreate),
								},
							},
						},
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
					},
				},
			},
		},
	}

	return sts
}

func (d *db) serviceForDB() interface{} {
	mysqlSvc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      DBName,
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

func (d *db) configMapForDB() interface{} {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rbd-db-conf",
			Namespace: d.component.Namespace,
		},
		Data: map[string]string{
			"mysql.cnf": `
[client]
# Default is Latin1, if you need UTF-8 set this (also in server section)
default-character-set = utf8

[mysqld]
#
# * Character sets
#
# Default is Latin1, if you need UTF-8 set all this (also in client section)
#
character-set-server  = utf8
collation-server      = utf8_general_ci
character_set_server   = utf8
collation_server       = utf8_general_ci`,
		},
	}

	return cm
}

func (d *db) initdbCMForDB() interface{} {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rbd-db-initdb",
			Namespace: d.component.Namespace,
		},
		Data: map[string]string{
			"initdb.sql": `
CREATE DATABASE console;
GRANT ALL ON *.* TO '` + d.mysqlUser + `'@'%';
FLUSH PRIVILEGES;
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
			mysqlPasswordKey: d.mysqlPassword,
			mysqlUserKey:     d.mysqlUser,
		},
	}
}
