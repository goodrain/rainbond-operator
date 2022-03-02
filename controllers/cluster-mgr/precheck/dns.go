package precheck

import (
	"net"
	"time"

	"github.com/docker/distribution/reference"
	"github.com/go-logr/logr"

	wutongv1alpha1 "github.com/wutong/wutong-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type dns struct {
	log     logr.Logger
	cluster *wutongv1alpha1.WutongCluster
}

// NewDNSPrechecker creates a new prechecker.
func NewDNSPrechecker(cluster *wutongv1alpha1.WutongCluster, log logr.Logger) PreChecker {
	return &dns{
		log:     log.WithName("DNSPreChecker"),
		cluster: cluster,
	}
}

func (d *dns) Check() wutongv1alpha1.WutongClusterCondition {
	condition := wutongv1alpha1.WutongClusterCondition{
		Type:              wutongv1alpha1.WutongClusterConditionTypeDNS,
		Status:            corev1.ConditionTrue,
		LastHeartbeatTime: metav1.NewTime(time.Now()),
	}

	ref, err := reference.Parse(d.cluster.Spec.WutongImageRepository)
	if err != nil {
		return d.failCondition(condition, err.Error())
	}
	domain := reference.Domain(ref.(reference.Named))

	// TODO: support offline installation
	if err := nslookup(domain); err != nil {
		return d.failCondition(condition, err.Error())
	}

	return condition
}

func nslookup(target string) error {
	_, err := net.LookupIP(target)
	if err != nil {
		return err
	}
	return nil
}

func (d *dns) failCondition(condition wutongv1alpha1.WutongClusterCondition, msg string) wutongv1alpha1.WutongClusterCondition {
	return failConditoin(condition, "DNSFailed", msg)
}
