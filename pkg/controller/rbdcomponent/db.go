package rbdcomponent

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	rainbondv1alpha1 "github.com/GLYASAI/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/GLYASAI/rainbond-operator/pkg/util/commonutil"
)

var rbdDBName = "rbd-db"

func deploymentForRainbondDB(r *rainbondv1alpha1.RbdComponent) interface{} {
	labels := labelsForRbdComponent(rbdDBName) // TODO: only on rainbond
	sts := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rbdDBName,
			Namespace: r.Namespace, // TODO: can use custom namespace?
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: commonutil.Int32(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   rbdDBName,
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            rbdDBName,
							Image:           "mysql:5.7.14",
							ImagePullPolicy: corev1.PullIfNotPresent, // TODO: custom
							Env: []corev1.EnvVar{
								{
									Name:  "MYSQL_ROOT_PASSWORD",
									Value: "rainbond",
								},
								{
									Name:  "MYSQL_DATABASE",
									Value: "region",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "data",
									MountPath: "/var/lib/mysql",
								},
								{
									Name:      "mysqlcnf",
									MountPath: "/etc/mysql/conf.d/mysql.cnf",
									SubPath:   "mysql.cnf",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "data",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: "rbd-db-data",
								},
							},
						},
						{
							Name: "mysqlcnf",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "rbd-db-conf",
									},
									Items: []corev1.KeyToPath{
										{
											Key:  "mysql.cnf",
											Path: "mysql.cnf",
										},
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

func serviceForDB(rc *rainbondv1alpha1.RbdComponent) interface{} {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rbd-db",
			Namespace: rc.Namespace,
			Labels: map[string]string{
				"name": "rbd-db",
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name: "main",
					Port: 3306,
				},
			},
			Selector: map[string]string{
				"name": "rbd-db",
			},
		},
	}

	return svc
}

func persistentVolumeClaimForDB(p *rainbondv1alpha1.RbdComponent) interface{} {
	storageRequest := resource.NewQuantity(10, resource.BinarySI)
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rbd-db-data",
			Namespace: p.Namespace,
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
			StorageClassName: commonutil.String("nfs"),
		},
	}

	return pvc
}

func configMapForDB(r *rainbondv1alpha1.RbdComponent) interface{} {
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rbd-db-conf",
			Namespace: r.Namespace,
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
