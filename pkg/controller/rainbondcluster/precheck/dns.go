package precheck

import (
	"fmt"
	"github.com/docker/distribution/reference"
	"github.com/go-logr/logr"
	"os/exec"
	"time"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
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
	out, err := ping(domain)
	if err != nil {
		return d.failCondition(condition, err.Error())
	}
	if out != "true\n" {
		return d.failCondition(condition, fmt.Sprintf("failed to ping %s", domain))
	}
	return condition
}

func ping(target string) (string, error) {
	cmd := fmt.Sprintf("ping -c 1 %s > /dev/null && echo true || echo false", target)
	output, err := exec.Command("/bin/sh", "-c", cmd).Output()
	if err != nil {
		return string(output), err
	}
	return string(output), nil
}

func (d *dns) failCondition(condition rainbondv1alpha1.RainbondClusterCondition, msg string) rainbondv1alpha1.RainbondClusterCondition {
	return failConditoin(condition, "DNSFailed", msg)
}
