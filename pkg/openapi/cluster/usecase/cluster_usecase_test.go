package usecase

import (
	"testing"

	"github.com/goodrain/rainbond-operator/pkg/generated/clientset/versioned"
	"github.com/goodrain/rainbond-operator/pkg/openapi/cluster/repository"
	"github.com/goodrain/rainbond-operator/pkg/openapi/model"
	"github.com/goodrain/rainbond-operator/pkg/util/k8sutil"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	v1 "github.com/goodrain/rainbond-operator/pkg/openapi/types/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestHandleStatus(t *testing.T) {
	c := ClusterUsecaseImpl{}
	var cluster *rainbondv1alpha1.RainbondCluster
	var pkg *rainbondv1alpha1.RainbondPackage
	var components []*v1.RbdComponentStatus
	var status model.ClusterStatus

	// waiting
	status = c.handleStatus(cluster, pkg, components)
	t.Logf("waiting status is %+v", status) // {FinalStatus:Waiting ClusterInfo:{NodeAvailPorts:[] Storage:[]}}

	// initing
	cluster = &rainbondv1alpha1.RainbondCluster{}
	status = c.handleStatus(cluster, pkg, components)
	t.Logf("initing status is %+v", status) // {FinalStatus:Waiting ClusterInfo:{NodeAvailPorts:[] Storage:[]}}

	// setting
	cluster = &rainbondv1alpha1.RainbondCluster{Status: &rainbondv1alpha1.RainbondClusterStatus{}}
	pkg = nil
	components = nil
	status = c.handleStatus(cluster, pkg, components)
	t.Logf(" setting status is %+v", status) // {FinalStatus:Waiting ClusterInfo:{NodeAvailPorts:[] Storage:[]}}

	// installing
	cluster = &rainbondv1alpha1.RainbondCluster{Status: &rainbondv1alpha1.RainbondClusterStatus{}}
	pkg = &rainbondv1alpha1.RainbondPackage{}
	components = []*v1.RbdComponentStatus{&v1.RbdComponentStatus{
		Name:          "rbd-api",
		Replicas:      1,
		ReadyReplicas: 0,
		Status:        "Waiting",
		Message:       "",
		Reason:        "",
		PodStatuses:   nil,
	}}
	status = c.handleStatus(cluster, pkg, components)
	t.Logf(" installing status is %+v", status) // {FinalStatus:Waiting ClusterInfo:{NodeAvailPorts:[] Storage:[]}}

	//installing
	cluster = &rainbondv1alpha1.RainbondCluster{Status: &rainbondv1alpha1.RainbondClusterStatus{}}
	pkg = &rainbondv1alpha1.RainbondPackage{}
	components = []*v1.RbdComponentStatus{}
	status = c.handleStatus(cluster, pkg, components)
	t.Logf(" installing status is %+v", status) // {FinalStatus:Waiting ClusterInfo:{NodeAvailPorts:[] Storage:[]}}

	//installing
	cluster = &rainbondv1alpha1.RainbondCluster{Status: &rainbondv1alpha1.RainbondClusterStatus{}}
	components = []*v1.RbdComponentStatus{}
	status = c.handleStatus(cluster, pkg, components)
	t.Logf(" installing status is %+v", status) // {FinalStatus:Waiting ClusterInfo:{NodeAvailPorts:[] Storage:[]}}

	//running
	cluster = &rainbondv1alpha1.RainbondCluster{Status: &rainbondv1alpha1.RainbondClusterStatus{}}
	pkg = &rainbondv1alpha1.RainbondPackage{}
	components = []*v1.RbdComponentStatus{&v1.RbdComponentStatus{
		Name:          "rbd-api",
		Replicas:      1,
		ReadyReplicas: 1,
		Status:        "Running",
		Message:       "",
		Reason:        "",
		PodStatuses:   nil,
	}}
	status = c.handleStatus(cluster, pkg, components)
	t.Logf(" running status is %+v", status) // {FinalStatus:Waiting ClusterInfo:{NodeAvailPorts:[] Storage:[]}}

	// uninstalling
	cluster = nil //
	pkg = &rainbondv1alpha1.RainbondPackage{}
	components = []*v1.RbdComponentStatus{&v1.RbdComponentStatus{
		Name:          "rbd-api",
		Replicas:      1,
		ReadyReplicas: 1,
		Status:        "Terminating",
		Message:       "",
		Reason:        "",
		PodStatuses:   nil,
	}}
	status = c.handleStatus(cluster, pkg, components)
	t.Logf(" uninstallingstatus is %+v", status) // {FinalStatus:Waiting ClusterInfo:{NodeAvailPorts:[] Storage:[]}}
}

func TestSelector(t *testing.T) {
	restConfig := k8sutil.MustNewKubeConfig("/Users/fanyangyang/Documents/company/goodrain/local/192.168.31.131.kubeconfig")
	client := versioned.NewForConfigOrDie(restConfig)
	list, err := client.RainbondV1alpha1().RbdComponents("rbd-system").List(metav1.ListOptions{LabelSelector: "name!=rbd-nfs"})
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range list.Items {
		t.Logf("component name is : %s", item.Name)
	}
}

func TestInitCluster(t *testing.T) {

	installMode := rainbondv1alpha1.InstallationModeWithoutPackage

	cluster := &rainbondv1alpha1.RainbondCluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "rbd-system",
			Name:      "mycluster",
		},
		Spec: rainbondv1alpha1.RainbondClusterSpec{
			RainbondShareStorage: rainbondv1alpha1.RainbondShareStorage{
				FstabLine: &rainbondv1alpha1.FstabLine{},
			},
			InstallPackageConfig: rainbondv1alpha1.InstallPackageConfig{},
			InstallMode:          installMode,
		},
	}

	repo := repository.NewClusterRepo("/opt/rainbond/.init")
	enterpriseID := repo.EnterpriseID()
	installID := repo.InstallID()

	annotations := make(map[string]string)
	annotations["enterprise_id"] = enterpriseID
	annotations["install_id"] = installID
	cluster.Annotations = annotations

	restConfig := k8sutil.MustNewKubeConfig("/Users/fanyangyang/Documents/company/goodrain/local/192.168.31.7.kubeconfig")
	client := versioned.NewForConfigOrDie(restConfig)
	_, err := client.RainbondV1alpha1().RainbondClusters("rbd-system").Create(cluster)
	if err != nil {
		t.Fatal(err)
	}
}
