package usecase

import (
	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/goodrain/rainbond-operator/pkg/generated/clientset/versioned"
	"github.com/goodrain/rainbond-operator/pkg/openapi/model"
	v1 "github.com/goodrain/rainbond-operator/pkg/openapi/types/v1"
	"github.com/goodrain/rainbond-operator/pkg/util/k8sutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
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
