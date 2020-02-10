package handler

import (
	"context"
	"fmt"
	"github.com/goodrain/rainbond-operator/pkg/util/commonutil"
	"strings"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/goodrain/rainbond-operator/pkg/util/constants"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var EventLogName = "rbd-eventlog"

type eventlog struct {
	ctx        context.Context
	client     client.Client
	component  *rainbondv1alpha1.RbdComponent
	cluster    *rainbondv1alpha1.RainbondCluster
	pkg        *rainbondv1alpha1.RainbondPackage
	labels     map[string]string
	db         *rainbondv1alpha1.Database
	etcdSecret *corev1.Secret
}

func NewEventLog(ctx context.Context, client client.Client, component *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster, pkg *rainbondv1alpha1.RainbondPackage) ComponentHandler {
	return &eventlog{
		ctx:       ctx,
		client:    client,
		component: component,
		cluster:   cluster,
		labels:    component.GetLabels(),
		pkg:       pkg,
	}
}

func (e *eventlog) Before() error {
	e.db = getDefaultDBInfo(e.cluster.Spec.RegionDatabase)

	secret, err := etcdSecret(e.ctx, e.client, e.cluster)
	if err != nil {
		return fmt.Errorf("failed to get etcd secret: %v", err)
	}
	e.etcdSecret = secret

	return nil
}

func (e *eventlog) Resources() []interface{} {
	return []interface{}{
		e.daemonSetForEventLog(),
	}
}

func (e *eventlog) After() error {
	return nil
}

func (e *eventlog) daemonSetForEventLog() interface{} {
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
		args = append(args, etcdSSLArgs()...)
	}

	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      EventLogName,
			Namespace: e.component.Namespace,
			Labels:    e.labels,
		},
		Spec: appsv1.DaemonSetSpec{
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
					HostNetwork:                   true,
					DNSPolicy:                     corev1.DNSClusterFirstWithHostNet,
					NodeSelector:                  e.cluster.Status.FirstMasterNodeLabel(),
					Tolerations: []corev1.Toleration{
						{
							Key:    e.cluster.Status.MasterRoleLabel,
							Effect: corev1.TaintEffectNoSchedule,
						},
					},
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
