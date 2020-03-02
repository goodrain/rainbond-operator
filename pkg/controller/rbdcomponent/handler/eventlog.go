package handler

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/goodrain/rainbond-operator/pkg/util/commonutil"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/goodrain/rainbond-operator/pkg/util/constants"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// EventLogName name for rbd-eventlog.
var EventLogName = "rbd-eventlog"

type eventlog struct {
	ctx        context.Context
	client     client.Client
	component  *rainbondv1alpha1.RbdComponent
	cluster    *rainbondv1alpha1.RainbondCluster
	labels     map[string]string
	db         *rainbondv1alpha1.Database
	etcdSecret *corev1.Secret

	pvcParametersRWX *pvcParameters
}

var _ ComponentHandler = &eventlog{}
var _ StorageClassRWXer = &eventlog{}

// NewEventLog creates a new rbd-eventlog handler.
func NewEventLog(ctx context.Context, client client.Client, component *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster) ComponentHandler {
	return &eventlog{
		ctx:       ctx,
		client:    client,
		component: component,
		cluster:   cluster,
		labels:    LabelsForRainbondComponent(component),
	}
}

func (e *eventlog) Before() error {
	db, err := getDefaultDBInfo(e.ctx, e.client, e.cluster.Spec.RegionDatabase, e.component.Namespace, DBName)
	if err != nil {
		return fmt.Errorf("get db info: %v", err)
	}
	e.db = db

	secret, err := etcdSecret(e.ctx, e.client, e.cluster)
	if err != nil {
		return fmt.Errorf("failed to get etcd secret: %v", err)
	}
	e.etcdSecret = secret

	if err := setStorageCassName(e.ctx, e.client, e.component.Namespace, e); err != nil {
		return err
	}

	return nil
}

func (e *eventlog) Resources() []interface{} {
	return []interface{}{
		e.deployment(),
	}
}

func (e *eventlog) After() error {
	return nil
}

func (e *eventlog) ListPods() ([]corev1.Pod, error) {
	return listPods(e.ctx, e.client, e.component.Namespace, e.labels)
}

func (e *eventlog) SetStorageClassNameRWX(pvcParameters *pvcParameters) {
	e.pvcParametersRWX = pvcParameters
}

func (e *eventlog) ResourcesCreateIfNotExists() []interface{} {
	return []interface{}{
		// pvc is immutable after creation except resources.requests for bound claims
		createPersistentVolumeClaimRWX(e.component.Namespace, constants.GrDataPVC, e.pvcParametersRWX, e.labels),
	}
}

func (e *eventlog) deployment() interface{} {
	args := []string{
		"--cluster.bind.ip=$(POD_IP)",
		"--cluster.instance.ip=$(POD_IP)",
		"--eventlog.bind.ip=$(POD_IP)",
		"--websocket.bind.ip=$(POD_IP)",
		"--db.url=" + strings.Replace(e.db.RegionDataSource(), "--mysql=", "", 1),
		"--discover.etcd.addr=" + strings.Join(etcdEndpoints(e.cluster), ","),
	}
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "grdata",
			MountPath: "/grdata",
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
	}
	if e.etcdSecret != nil {
		volume, mount := volumeByEtcd(e.etcdSecret)
		volumeMounts = append(volumeMounts, mount)
		volumes = append(volumes, volume)
		args = append(args, eventLogEtcdArgs()...)
	}

	ds := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      EventLogName,
			Namespace: e.component.Namespace,
			Labels:    e.labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: e.component.Spec.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: e.labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   EventLogName,
					Labels: e.labels,
				},
				Spec: corev1.PodSpec{
					TerminationGracePeriodSeconds: commonutil.Int64(0),
					Containers: []corev1.Container{
						{
							Name:            EventLogName,
							Image:           e.component.Spec.Image,
							ImagePullPolicy: e.component.ImagePullPolicy(),
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
									Name:  "K8S_MASTER",
									Value: "kubernetes",
								},
								{
									Name:  "DOCKER_LOG_SAVE_DAY",
									Value: "7",
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

func eventLogEtcdArgs() []string {
	return []string{
		"--discover.etcd.ca=" + path.Join(EtcdSSLPath, "ca-file"),
		"--discover.etcd.cert=" + path.Join(EtcdSSLPath, "cert-file"),
		"--discover.etcd.key=" + path.Join(EtcdSSLPath, "key-file"),
	}
}
