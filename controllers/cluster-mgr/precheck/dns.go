package precheck

import (
	"net"
	"time"

	"github.com/docker/distribution/reference"
	"github.com/go-logr/logr"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type dns struct {
	log     logr.Logger
	cluster *rainbondv1alpha1.RainbondCluster
}

// NewDNSPrechecker creates a new prechecker.
func NewDNSPrechecker(cluster *rainbondv1alpha1.RainbondCluster, log logr.Logger) PreChecker {
	return &dns{
		log:     log.WithName("DNSPreChecker"),
		cluster: cluster,
	}
}

func (d *dns) Check() rainbondv1alpha1.RainbondClusterCondition {
	condition := rainbondv1alpha1.RainbondClusterCondition{
		Type:              rainbondv1alpha1.RainbondClusterConditionTypeDNS,
		Status:            corev1.ConditionTrue,
		LastHeartbeatTime: metav1.NewTime(time.Now()),
	}

	ref, err := reference.Parse(d.cluster.Spec.RainbondImageRepository)
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

func (d *dns) failCondition(condition rainbondv1alpha1.RainbondClusterCondition, msg string) rainbondv1alpha1.RainbondClusterCondition {
	return failConditoin(condition, "DNSFailed", msg)
}
