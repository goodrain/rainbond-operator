package rainbondcluster

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/goodrain/rainbond-operator/test/e2e/framework"
	corev1 "github.com/openshift/api/operator/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	waitForStatusSync = 25 * time.Second
)

var _ = Describe("RainbondCluster Status Condition", func() {
	f := framework.NewDefaultFramework("condition")

	BeforeEach(func() {
	})

	AfterEach(func() {
	})

	It("should fail to check database", func() {
		// create rainbondcluster
		rianbondcluster := &rainbondv1alpha1.RainbondCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "wrong-database-cluster",
				Namespace: f.Namespace,
			},
			Spec: rainbondv1alpha1.RainbondClusterSpec{
				RegionDatabase: &rainbondv1alpha1.Database{
					Host:     "",
					Port:     0,
					Username: "",
					Password: "",
					Name:     "",
				},
			},
		}
	
		f.EnsureRainbondCluster(rianbondcluster)
	
		time.Sleep(waitForStatusSync)
	
		cluster, err := f.RainbondClientset.RainbondV1alpha1().RainbondClusters(rianbondcluster.Namespace).Get(rianbondcluster.GetName(), metav1.GetOptions{})
		Expect(err).To(BeNil(), "unexpected error obtaining rainbondcluster")
		Expect(cluster).NotTo(BeNil(), "expected a cluster but none returned")
		Expect(cluster.Status).NotTo(BeNil(), "expected a cluster status but none returned")
		Expect(cluster.Status.Conditions).NotTo(BeNil(), "expected a cluster status conditions but none returned")
	
		_, condition := cluster.Status.GetCondition(rainbondv1alpha1.RainbondClusterConditionTypeDatabaseRegion)
		Expect(string(condition.Status)).To(Equal(string(corev1.ConditionFalse)))
	})

	It("should fail to check image repository", func() {
		// create rainbondcluster
		rianbondcluster := &rainbondv1alpha1.RainbondCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "wrong-imagerepo-cluster",
				Namespace: f.Namespace,
			},
			Spec: rainbondv1alpha1.RainbondClusterSpec{
				ImageHub: &rainbondv1alpha1.ImageHub{
					Domain: "foobar.com",
					Namespace: "",
					Username: "foo",
					Password: "bar",
				},
			},
		}

		f.EnsureRainbondCluster(rianbondcluster)

		time.Sleep(waitForStatusSync)

		cluster, err := f.RainbondClientset.RainbondV1alpha1().RainbondClusters(rianbondcluster.Namespace).Get(rianbondcluster.GetName(), metav1.GetOptions{})
		Expect(err).To(BeNil(), "unexpected error obtaining rainbondcluster")
		Expect(cluster).NotTo(BeNil(), "expected a cluster but none returned")
		Expect(cluster.Status).NotTo(BeNil(), "expected a cluster status but none returned")
		Expect(cluster.Status.Conditions).NotTo(BeNil(), "expected a cluster status conditions but none returned")

		_, condition := cluster.Status.GetCondition(rainbondv1alpha1.RainbondClusterConditionTypeImageRepository)
		Expect(string(condition.Status)).To(Equal(string(corev1.ConditionFalse)))
	})
})
