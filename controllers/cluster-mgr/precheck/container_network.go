package precheck

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/go-logr/logr"
	wutongv1alpha1 "github.com/wutong/wutong-operator/api/v1alpha1"
	"github.com/wutong/wutong-operator/util/commonutil"
	"github.com/wutong/wutong-operator/util/constants"
	"github.com/wutong/wutong-operator/util/k8sutil"
	"github.com/wutong/wutong-operator/util/wtutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// SentinelName -
const SentinelName = "wutong-operator-sentinel"

// ErrSentinelNotReady -
var ErrSentinelNotReady = errors.New("wutong-operator-sentinel not ready")

type containerNetwork struct {
	ctx     context.Context
	log     logr.Logger
	client  client.Client
	scheme  *runtime.Scheme
	cluster *wutongv1alpha1.WutongCluster
}

// NewContainerNetworkPrechecker creates a new prechecker.
func NewContainerNetworkPrechecker(ctx context.Context, client client.Client, scheme *runtime.Scheme, log logr.Logger, cluster *wutongv1alpha1.WutongCluster) PreChecker {
	return &containerNetwork{
		log:     log.WithName("ContainerNetworkPreChecker"),
		ctx:     ctx,
		cluster: cluster,
		client:  client,
		scheme:  scheme,
	}
}

func (c *containerNetwork) Check() wutongv1alpha1.WutongClusterCondition {
	condition := wutongv1alpha1.WutongClusterCondition{
		Type:              wutongv1alpha1.WutongClusterConditionTypeContainerNetwork,
		Status:            corev1.ConditionTrue,
		LastHeartbeatTime: metav1.NewTime(time.Now()),
	}

	// Check if sentinel is ready
	if msg, err := c.isSentinelReady(); err != nil {
		if err == ErrSentinelNotReady {
			condition.Status = corev1.ConditionFalse
			condition.Reason = "SentinelNotReady"
			condition.Message = msg
			return condition
		}
		return c.failCondition(condition, err.Error())
	}

	// Check whether it can communicate with each pods
	if err := c.communicate(); err != nil {
		return c.failCondition(condition, err.Error())
	}

	return condition
}

func (c *containerNetwork) failCondition(condition wutongv1alpha1.WutongClusterCondition, msg string) wutongv1alpha1.WutongClusterCondition {
	return failConditoin(condition, "ContainerNetworkFailed", msg)
}

func (c *containerNetwork) isSentinelReady() (string, error) {
	ds := appsv1.DaemonSet{}
	err := c.client.Get(c.ctx, types.NamespacedName{Namespace: c.cluster.GetNamespace(), Name: SentinelName}, &ds)
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			// creates a new one
			return "", c.createSentinel()
		}
		return "", err
	}

	if ds.Status.NumberAvailable != ds.Status.DesiredNumberScheduled {
		msg := "desired %d pods to be available, but only got %d"
		return fmt.Sprintf(msg, ds.Status.DesiredNumberScheduled, ds.Status.NumberAvailable), ErrSentinelNotReady
	}

	return "", nil
}

func (c *containerNetwork) communicate() error {
	podList := corev1.PodList{}
	labels := wtutil.LabelsForWutong(map[string]string{
		"name": SentinelName,
	})
	err := c.client.List(c.ctx, &podList, client.InNamespace(c.cluster.Namespace), client.MatchingLabels(labels))
	if err != nil {
		return err
	}

	var badPods []string
	for _, pod := range podList.Items {
		if err := dial(pod.Status.PodIP, 8080); err != nil {
			badPods = append(badPods, fmt.Sprintf("%s(%s)", pod.GetName(), pod.Status.PodIP))
		}
	}

	if len(badPods) > 0 {
		return fmt.Errorf("can not communicate with %s", strings.Join(badPods, ","))
	}

	return nil
}

func dial(ip string, port int) error {
	var d net.Dialer
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	conn, err := d.DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		return err
	}
	defer conn.Close()

	return nil
}

func (c *containerNetwork) createSentinel() error {
	ds := c.daemonsetForSentinel()

	// Set rainboncluster as the owner and controller
	if err := controllerutil.SetControllerReference(c.cluster, ds, c.scheme); err != nil {
		return err
	}

	return k8sutil.CreateIfNotExists(c.ctx, c.client, ds)
}

func (c *containerNetwork) daemonsetForSentinel() *appsv1.DaemonSet {
	labels := wtutil.LabelsForWutong(map[string]string{
		"name": SentinelName,
	})
	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      SentinelName,
			Namespace: c.cluster.GetNamespace(),
			Labels: wtutil.LabelsForWutong(map[string]string{
				"name": SentinelName,
			}),
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   SentinelName,
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					TerminationGracePeriodSeconds: commonutil.Int64(0),
					ServiceAccountName:            constants.ServiceAccountName,
					Tolerations: []corev1.Toleration{
						{
							Operator: corev1.TolerationOpExists, // tolerate everything.
						},
					},
					Containers: []corev1.Container{
						{
							Name:            SentinelName,
							Image:           c.cluster.Spec.SentinelImage,
							ImagePullPolicy: corev1.PullIfNotPresent,
						},
					},
				},
			},
		},
	}
}
