package handler

import (
	"context"
	"fmt"

	etcdv1beta2 "github.com/coreos/etcd-operator/pkg/apis/etcd/v1beta2"
	"github.com/docker/distribution/reference"
	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/goodrain/rainbond-operator/pkg/util/commonutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// EtcdName name for rbd-etcd.
var EtcdName = "rbd-etcd"

type etcd struct {
	ctx       context.Context
	client    client.Client
	component *rainbondv1alpha1.RbdComponent
	cluster   *rainbondv1alpha1.RainbondCluster
	labels    map[string]string

	pvcParametersRWO *pvcParameters
	storageRequest   int64
}

var _ ComponentHandler = &etcd{}
var _ StorageClassRWOer = &etcd{}
var _ Replicaser = &etcd{}

// NewETCD creates a new rbd-etcd handler.
func NewETCD(ctx context.Context, client client.Client, component *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster) ComponentHandler {
	labels := LabelsForRainbondComponent(component)
	labels["etcd_node"] = EtcdName
	return &etcd{
		ctx:            ctx,
		client:         client,
		component:      component,
		cluster:        cluster,
		labels:         labels,
		storageRequest: getStorageRequest("ETCD_DATA_STORAGE_REQUEST", 21),
	}
}

func (e *etcd) Before() error {
	if e.cluster.Spec.EtcdConfig != nil {
		return NewIgnoreError(fmt.Sprintf("specified etcd configuration"))
	}
	if err := setStorageCassName(e.ctx, e.client, e.component.Namespace, e); err != nil {
		return err
	}

	return nil
}

func (e *etcd) Resources() []interface{} {
	if e.cluster.Spec.EnableHA {
		return []interface{}{
			e.statefulsetForEtcdCluster(),
			e.serviceForEtcd(),
		}
	}

	return []interface{}{
		e.statefulsetForEtcd(),
		e.serviceForEtcd(),
	}
}

func (e *etcd) After() error {
	return nil
}

func (e *etcd) ListPods() ([]corev1.Pod, error) {
	return listPods(e.ctx, e.client, e.component.Namespace, e.labels)
}

func (e *etcd) SetStorageClassNameRWO(pvcParameters *pvcParameters) {
	e.pvcParametersRWO = pvcParameters
}

func (e *etcd) Replicas() *int32 {
	if e.cluster.Spec.EnableHA {
		return commonutil.Int32(3)
	}
	return commonutil.Int32(1)
}

func (e *etcd) statefulsetForEtcd() interface{} {
	claimName := "data"
	pvc := createPersistentVolumeClaimRWO(e.component.Namespace, claimName, e.pvcParametersRWO, e.labels, e.storageRequest)
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      EtcdName,
			Namespace: e.component.Namespace,
			Labels:    e.labels,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:    e.Replicas(),
			ServiceName: EtcdName,
			Selector: &metav1.LabelSelector{
				MatchLabels: e.labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      EtcdName,
					Namespace: e.component.Namespace,
					Labels:    e.labels,
				},
				Spec: corev1.PodSpec{
					ImagePullSecrets:              imagePullSecrets(e.component, e.cluster),
					TerminationGracePeriodSeconds: commonutil.Int64(0),
					Containers: []corev1.Container{
						{
							Name:            EtcdName,
							Image:           e.component.Spec.Image,
							ImagePullPolicy: e.component.ImagePullPolicy(),
							Command: []string{
								"/usr/local/bin/etcd",
								"--name",
								EtcdName,
								"--data-dir /var/run/etcd/default.etcd",
								"--initial-advertise-peer-urls",
								fmt.Sprintf("http://%s:2380", EtcdName),
								"--listen-peer-urls",
								"http://0.0.0.0:2380",
								"--listen-client-urls",
								"http://0.0.0.0:2379",
								"--advertise-client-urls",
								fmt.Sprintf("http://%s:2379", EtcdName),
								"--initial-cluster",
								fmt.Sprintf("%s=http://%s:2380", EtcdName, EtcdName),
								"--initial-cluster-state",
								"new",
							},
							Env: []corev1.EnvVar{
								{
									Name:  "ETCD_QUOTA_BACKEND_BYTES",
									Value: "4294967296", // 4 Gi
								},
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "client",
									ContainerPort: 2379,
								},
								{
									Name:          "server",
									ContainerPort: 2380,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      claimName,
									MountPath: "/var/run/etcd",
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

func (e *etcd) statefulsetForEtcdCluster() *appsv1.StatefulSet {
	claimName := "data"
	pvc := createPersistentVolumeClaimRWO(e.component.Namespace, claimName, e.pvcParametersRWO, e.labels, e.storageRequest)

	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      EtcdName,
			Namespace: e.component.Namespace,
			Labels:    e.labels,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:    e.Replicas(),
			ServiceName: EtcdName,
			Selector: &metav1.LabelSelector{
				MatchLabels: e.labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      EtcdName,
					Namespace: e.component.Namespace,
					Labels:    e.labels,
				},
				Spec: corev1.PodSpec{
					TerminationGracePeriodSeconds: commonutil.Int64(0),
					Containers: []corev1.Container{
						{
							Name:            EtcdName,
							Image:           e.component.Spec.Image,
							ImagePullPolicy: e.component.ImagePullPolicy(),
							Command: []string{
								"/bin/sh",
								"-ec",
								`
HOSTNAME=$(hostname)
          echo "etcd api version is ${ETCDAPI_VERSION}"

          eps() {
              EPS=""
              for i in $(seq 0 $((${INITIAL_CLUSTER_SIZE} - 1))); do
                  EPS="${EPS}${EPS:+,}http://${SET_NAME}-${i}.${SET_NAME}.${CLUSTER_NAMESPACE}:2379"
              done
              echo ${EPS}
          }

          member_hash() {
              etcdctl member list | grep http://${HOSTNAME}.${SET_NAME}.${CLUSTER_NAMESPACE}:2380 | cut -d':' -f1 | cut -d'[' -f1
          }

          initial_peers() {
                PEERS=""
                for i in $(seq 0 $((${INITIAL_CLUSTER_SIZE} - 1))); do
                PEERS="${PEERS}${PEERS:+,}${SET_NAME}-${i}=http://${SET_NAME}-${i}.${SET_NAME}.${CLUSTER_NAMESPACE}:2380"
                done
                echo ${PEERS}
          }

          # etcd-SET_ID
          SET_ID=${HOSTNAME##*-}
          # adding a new member to existing cluster (assuming all initial pods are available)
          if [ "${SET_ID}" -ge ${INITIAL_CLUSTER_SIZE} ]; then
              export ETCDCTL_ENDPOINTS=$(eps)

              # member already added?
              MEMBER_HASH=$(member_hash)
              if [ -n "${MEMBER_HASH}" ]; then
                  # the member hash exists but for some reason etcd failed
                  # as the datadir has not be created, we can remove the member
                  # and retrieve new hash
                  if [ "${ETCDAPI_VERSION}" -eq 3 ]; then
                      ETCDCTL_API=3 etcdctl --user=root:${ROOT_PASSWORD} member remove ${MEMBER_HASH}
                  else
                      etcdctl --username=root:${ROOT_PASSWORD} member remove ${MEMBER_HASH}
                  fi
              fi
              echo "Adding new member"
              rm -rf /var/run/etcd/*
              # ensure etcd dir exist
              mkdir -p /var/run/etcd/
              # sleep 60s wait endpoint become ready
              echo "sleep 60s wait endpoint become ready,sleeping..."
              sleep 60

              if [ "${ETCDAPI_VERSION}" -eq 3 ]; then
                  ETCDCTL_API=3 etcdctl --user=root:${ROOT_PASSWORD} member add ${HOSTNAME} --peer-urls=http://${HOSTNAME}.${SET_NAME}.${CLUSTER_NAMESPACE}:2380 | grep "^ETCD_" > /var/run/etcd/new_member_envs
              else
                  etcdctl --username=root:${ROOT_PASSWORD} member add ${HOSTNAME} http://${HOSTNAME}.${SET_NAME}.${CLUSTER_NAMESPACE}:2380 | grep "^ETCD_" > /var/run/etcd/new_member_envs
              fi
              
              

              if [ $? -ne 0 ]; then
                  echo "member add ${HOSTNAME} error."
                  rm -f /var/run/etcd/new_member_envs
                  exit 1
              fi

              cat /var/run/etcd/new_member_envs
              source /var/run/etcd/new_member_envs

              exec etcd --name ${HOSTNAME} \
                  --initial-advertise-peer-urls http://${HOSTNAME}.${SET_NAME}.${CLUSTER_NAMESPACE}:2380 \
                  --listen-peer-urls http://0.0.0.0:2380 \
                  --listen-client-urls http://0.0.0.0:2379 \
                  --advertise-client-urls http://${HOSTNAME}.${SET_NAME}.${CLUSTER_NAMESPACE}:2379 \
                  --data-dir /var/run/etcd/default.etcd \
                  --initial-cluster ${ETCD_INITIAL_CLUSTER} \
                  --initial-cluster-state ${ETCD_INITIAL_CLUSTER_STATE}
          fi

          for i in $(seq 0 $((${INITIAL_CLUSTER_SIZE} - 1))); do
              while true; do
                  echo "Waiting for ${SET_NAME}-${i}.${SET_NAME}.${CLUSTER_NAMESPACE} to come up"
                  ping -W 1 -c 1 ${SET_NAME}-${i}.${SET_NAME}.${CLUSTER_NAMESPACE} > /dev/null && break
                  sleep 1s
              done
          done

          echo "join member ${HOSTNAME}"
          # join member
          exec etcd --name ${HOSTNAME} \
              --initial-advertise-peer-urls http://${HOSTNAME}.${SET_NAME}.${CLUSTER_NAMESPACE}:2380 \
              --listen-peer-urls http://0.0.0.0:2380 \
              --listen-client-urls http://0.0.0.0:2379 \
              --advertise-client-urls http://${HOSTNAME}.${SET_NAME}.${CLUSTER_NAMESPACE}:2379 \
              --initial-cluster-token etcd-cluster-1 \
              --data-dir /var/run/etcd/default.etcd \
              --initial-cluster $(initial_peers) \
              --initial-cluster-state new
`,
							},
							Env: []corev1.EnvVar{
								{
									Name:  "ETCD_QUOTA_BACKEND_BYTES",
									Value: "4294967296", // 4 Gi
								},
								{
									Name:  "INITIAL_CLUSTER_SIZE",
									Value: "3",
								},
								{
									Name: "CLUSTER_NAMESPACE",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.namespace",
										},
									},
								},
								{
									Name:  "ETCDAPI_VERSION",
									Value: "3",
								},
								{
									Name:  "ROOT_PASSWORD",
									Value: "@123#",
								},
								{
									Name:  "SET_NAME",
									Value: EtcdName,
								},
								{
									Name:  "GOMAXPROCS",
									Value: "4",
								},
							},
							Lifecycle: &corev1.Lifecycle{
								PreStop: &corev1.Handler{
									Exec: &corev1.ExecAction{
										Command: []string{
											"/bin/sh",
											"-ec",
											`
HOSTNAME=$(hostname)

member_hash() {
	etcdctl member list | grep http://${HOSTNAME}.${SET_NAME}.${CLUSTER_NAMESPACE}:2380 | cut -d':' -f1 | cut -d'[' -f1
}

eps() {
	EPS=""
	for i in $(seq 0 $((${INITIAL_CLUSTER_SIZE} - 1))); do
		EPS="${EPS}${EPS:+,}http://${SET_NAME}-${i}.${SET_NAME}.${CLUSTER_NAMESPACE}:2379"
	done
	echo ${EPS}
}

export ETCDCTL_ENDPOINTS=$(eps)

SET_ID=${HOSTNAME##*-}
# Removing member from cluster
if [ "${SET_ID}" -ge ${INITIAL_CLUSTER_SIZE} ]; then
	echo "Removing ${HOSTNAME} from etcd cluster"
	if [ "${ETCDAPI_VERSION}" -eq 3 ]; then
		ETCDCTL_API=3 etcdctl --user=root:${ROOT_PASSWORD} member remove $(member_hash)
	else
		etcdctl --username=root:${ROOT_PASSWORD} member remove $(member_hash)
	fi
	if [ $? -eq 0 ]; then
		# Remove everything otherwise the cluster will no longer scale-up
		rm -rf /var/run/etcd/*
	fi
fi
`,
										},
									},
								},
							},
							Ports: []corev1.ContainerPort{
								{
									Name:          "client",
									ContainerPort: 2379,
								},
								{
									Name:          "server",
									ContainerPort: 2380,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      claimName,
									MountPath: "/var/run/etcd",
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

func (e *etcd) serviceForEtcd() interface{} {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      EtcdName,
			Namespace: e.component.Namespace,
			Labels:    e.labels,
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "None",
			Ports: []corev1.ServicePort{
				{
					Name: "client",
					Port: 2379,
				},
				{
					Name: "server",
					Port: 2380,
				},
			},
			Selector: e.labels,
		},
	}

	return svc
}

func (e *etcd) etcdCluster() *etcdv1beta2.EtcdCluster {
	claimName := "data"
	pvc := createPersistentVolumeClaimRWO(e.component.Namespace, claimName, e.pvcParametersRWO, e.labels, e.storageRequest)

	// make sure the image name is right
	repo, _ := reference.Parse(e.component.Spec.Image)
	named := repo.(reference.Named)
	tag := "latest"
	if t, ok := repo.(reference.Tagged); ok {
		tag = t.Tag()
	}

	defaultSize := 1
	if e.component.Spec.Replicas != nil {
		defaultSize = int(*e.component.Spec.Replicas)
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
									Key:      "app",
									Operator: metav1.LabelSelectorOpIn,
									Values: []string{
										"etcd",
									},
								},
							},
						},
						TopologyKey: "kubernetes.io/hostname",
					},
				},
			},
		},
	}

	ec := &etcdv1beta2.EtcdCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      EtcdName,
			Namespace: e.component.Namespace,
			Labels:    e.labels,
		},
		Spec: etcdv1beta2.ClusterSpec{
			Size:       defaultSize,
			Repository: named.Name(),
			Version:    tag,
			Pod: &etcdv1beta2.PodPolicy{
				Affinity:                  affinity,
				PersistentVolumeClaimSpec: &pvc.Spec,
			},
		},
	}

	return ec
}
