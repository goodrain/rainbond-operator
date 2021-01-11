package framework

import (
	"fmt"
	"time"

	. "github.com/onsi/gomega"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/goodrain/rainbond-operator/pkg/util/k8sutil"
	core "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

// WaitForPodsReady waits for a given amount of time until a group of Pods is running in the given namespace.
func WaitForPodsReady(kubeClientSet kubernetes.Interface, timeout time.Duration, expectedReplicas int, namespace string, opts metav1.ListOptions) error {
	return wait.Poll(Poll, timeout, func() (bool, error) {
		pl, err := kubeClientSet.CoreV1().Pods(namespace).List(opts)
		if err != nil {
			return false, nil
		}

		r := 0
		for _, p := range pl.Items {
			if isRunning, _ := podRunningReady(&p); isRunning {
				r++
			}
		}

		if r == expectedReplicas {
			return true, nil
		}

		return false, nil
	})
}

// podRunningReady checks whether pod p's phase is running and it has a ready
// condition of status true.
func podRunningReady(p *core.Pod) (bool, error) {
	// Check the phase is running.
	if p.Status.Phase != core.PodRunning {
		return false, fmt.Errorf("want pod '%s' on '%s' to be '%v' but was '%v'",
			p.ObjectMeta.Name, p.Spec.NodeName, core.PodRunning, p.Status.Phase)
	}
	// Check the ready condition is true.
	if !k8sutil.IsPodReady(p) {
		return false, fmt.Errorf("pod '%s' on '%s' didn't have condition {%v %v}; conditions: %v",
			p.ObjectMeta.Name, p.Spec.NodeName, core.PodReady, core.ConditionTrue, p.Status.Conditions)
	}
	return true, nil
}

// EnsureRainbondCluster creates a rainbondcluster object or returns it if it already exists.
func (f *Framework) EnsureRainbondCluster(new *rainbondv1alpha1.RainbondCluster) *rainbondv1alpha1.RainbondCluster {
	cluster, err := f.RainbondClientset.RainbondV1alpha1().RainbondClusters(new.Namespace).Get(new.GetName(), metav1.GetOptions{})
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			cluster, err = f.RainbondClientset.RainbondV1alpha1().RainbondClusters(new.Namespace).Create(new)
			Expect(err).NotTo(HaveOccurred(), "unexpected error creating rainbondcluster")
			return cluster
		}

		Expect(err).NotTo(HaveOccurred())
	}

	new.ResourceVersion = cluster.ResourceVersion
	cluster, err = f.RainbondClientset.RainbondV1alpha1().RainbondClusters(new.Namespace).Update(new)

	Expect(cluster).NotTo(BeNil())

	return cluster
}

// DeleteClusterRoleBinding deletes a ClusterRoleBinding
func DeleteClusterRoleBinding(c kubernetes.Interface, name string) error {
	err := c.RbacV1().ClusterRoleBindings().Delete(name, metav1.NewDeleteOptions(0))
	if k8sErrors.IsNotFound(err) {
		return nil
	}
	return err
}
