package handler

import (
	"context"
	"fmt"
	checksqllite "github.com/goodrain/rainbond-operator/util/check-sqllite"
	"strings"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
	"github.com/goodrain/rainbond-operator/util/commonutil"
	"github.com/goodrain/rainbond-operator/util/constants"
	"github.com/goodrain/rainbond-operator/util/probeutil"

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
	storageRequest   int64
}

var _ ComponentHandler = &eventlog{}
var _ StorageClassRWXer = &eventlog{}
var _ ResourcesCreator = &eventlog{}
var _ ResourcesDeleter = &eventlog{}

// NewEventLog creates a new rbd-eventlog handler.
func NewEventLog(ctx context.Context, client client.Client, component *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster) ComponentHandler {
	return &eventlog{
		ctx:            ctx,
		client:         client,
		component:      component,
		cluster:        cluster,
		labels:         LabelsForRainbondComponent(component),
		storageRequest: getStorageRequest("GRDATA_STORAGE_REQUEST", 20),
	}
}

func (e *eventlog) Before() error {
	if !checksqllite.IsSQLLite() {
		db, err := getDefaultDBInfo(e.ctx, e.client, e.cluster.Spec.RegionDatabase, e.component.Namespace, DBName)
		if err != nil {
			return fmt.Errorf("get db info: %v", err)
		}
		if db.Name == "" {
			db.Name = RegionDatabaseName
		}
		e.db = db
	}

	secret, err := etcdSecret(e.ctx, e.client, e.cluster)
	if err != nil {
		return fmt.Errorf("failed to get etcd secret: %v", err)
	}
	e.etcdSecret = secret

	if e.component.Labels["persistentVolumeClaimAccessModes"] == string(corev1.ReadWriteOnce) {
		sc, err := storageClassNameFromRainbondVolumeRWO(e.ctx, e.client, e.component.Namespace)
		if err != nil {
			return err
		}
		e.SetStorageClassNameRWX(sc)
		return nil
	}
	return setStorageCassName(e.ctx, e.client, e.component.Namespace, e)
}

func (e *eventlog) Resources() []client.Object {
	return []client.Object{
		e.statefulset(),
		e.service(),
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

func (e *eventlog) ResourcesCreateIfNotExists() []client.Object {
	if e.component.Labels["persistentVolumeClaimAccessModes"] == string(corev1.ReadWriteOnce) {
		return []client.Object{
			createPersistentVolumeClaimRWO(e.component.Namespace, constants.GrDataPVC, e.pvcParametersRWX, e.labels, e.storageRequest),
		}
	}
	return []client.Object{
		// pvc is immutable after creation except resources.requests for bound claims
		createPersistentVolumeClaimRWX(e.component.Namespace, constants.GrDataPVC, e.pvcParametersRWX, e.labels, e.storageRequest),
	}
}

func (e *eventlog) ResourcesNeedDelete() []client.Object {
	// delete deploy which created in rainbond 5.2.0
	sts := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      EventLogName,
			Namespace: e.component.Namespace,
			Labels:    e.labels,
		},
	}
	return []client.Object{
		sts,
	}
}

func (e *eventlog) service() client.Object {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      EventLogName,
			Namespace: e.component.Namespace,
			Labels:    e.labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name: "websocket",
					Port: 6363,
				},
				{
					Name: "dockerlog",
					Port: 6362,
				},
				{
					Name: "monitorlog",
					Port: 6366,
				},
				{
					Name: "monitorlog-udp",
					Port: 6166,
				},
			},
			Selector: e.labels,
		},
	}
}

func (e *eventlog) statefulset() client.Object {
	args := []string{
		"--cluster.bind.ip=$(POD_IP)",
		"--cluster.instance.ip=$(POD_IP)",
		"--eventlog.bind.ip=$(POD_IP)",
		"--websocket.bind.ip=$(POD_IP)",
	}
	if !checksqllite.IsSQLLite() {
		args = append(args, "--db.url="+strings.Replace(e.db.RegionDataSource(), "--mysql=", "", 1))
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
	}

	env := []corev1.EnvVar{
		{
			Name:  "RBD_NAMESPACE",
			Value: e.component.Namespace,
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
			Name: "NODE_ID",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
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
	}

	env = mergeEnvs(env, e.component.Spec.Env)
	volumeMounts = mergeVolumeMounts(volumeMounts, e.component.Spec.VolumeMounts)
	volumes = mergeVolumes(volumes, e.component.Spec.Volumes)
	args = mergeArgs(args, e.component.Spec.Args)

	// prepare probe
	readinessProbe := probeutil.MakeReadinessProbeTCP("", 6363)
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      EventLogName,
			Namespace: e.component.Namespace,
			Labels:    e.labels,
		},
		Spec: appsv1.StatefulSetSpec{
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
					ImagePullSecrets:              imagePullSecrets(e.component, e.cluster),
					TerminationGracePeriodSeconds: commonutil.Int64(0),
					Containers: []corev1.Container{
						{
							Name:            EventLogName,
							Image:           e.component.Spec.Image,
							ImagePullPolicy: e.component.ImagePullPolicy(),
							Env:             env,
							Args:            args,
							VolumeMounts:    volumeMounts,
							ReadinessProbe:  readinessProbe,
							Resources:       e.component.Spec.Resources,
						},
					},
					Volumes: volumes,
				},
			},
		},
	}

	return sts
}
