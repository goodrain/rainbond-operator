package clustermgr

import (
	"context"
	"strings"
	"testing"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestGenerateConditionsDefersRunningCheckWhenPrecheckNotReady(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("add corev1 to scheme: %v", err)
	}
	if err := rainbondv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("add rainbondv1alpha1 to scheme: %v", err)
	}

	cluster := &rainbondv1alpha1.RainbondCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rainbondcluster",
			Namespace: "rbd-system",
		},
		Spec: rainbondv1alpha1.RainbondClusterSpec{
			InstallMode: rainbondv1alpha1.InstallationModeOffline,
		},
	}

	k8sClient := &clusterStatusTestClient{
		scheme: scheme,
		nodes: []corev1.Node{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "node-a"},
				Status: corev1.NodeStatus{
					Allocatable: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("4Gi"),
					},
					NodeInfo: corev1.NodeSystemInfo{
						KubeletVersion: "v1.20.0",
					},
				},
			},
		},
		components: append(readyRbdComponents("rbd-system"), notReadyRbdComponent("rbd-system", "rbd-api")),
	}
	mgr := NewClusterMgr(context.Background(), k8sClient, ctrl.Log.WithName("test"), cluster, scheme)

	status, err := mgr.GenerateRainbondClusterStatus()
	if err != nil {
		t.Fatalf("generate status: %v", err)
	}

	_, running := status.GetCondition(rainbondv1alpha1.RainbondClusterConditionTypeRunning)
	if running == nil {
		t.Fatal("expected Running condition")
	}
	if running.Status != corev1.ConditionFalse {
		t.Fatalf("expected Running=False, got %s", running.Status)
	}
	if running.Reason != "PrecheckNotReady" {
		t.Fatalf("expected precheck reason, got %q with message %q", running.Reason, running.Message)
	}
	if !strings.Contains(running.Message, "Storage") {
		t.Fatalf("expected Running message to point at Storage precheck, got %q", running.Message)
	}
	if strings.Contains(running.Message, "rbd-api") {
		t.Fatalf("expected Running message not to report component readiness before prechecks pass, got %q", running.Message)
	}
}

func TestGenerateConditionsIgnoresIrrelevantHistoricalPrecheckFailures(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("add corev1 to scheme: %v", err)
	}
	if err := rainbondv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("add rainbondv1alpha1 to scheme: %v", err)
	}

	cluster := &rainbondv1alpha1.RainbondCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rainbondcluster",
			Namespace: "rbd-system",
		},
		Spec: rainbondv1alpha1.RainbondClusterSpec{
			InstallMode:           rainbondv1alpha1.InstallationModeOffline,
			RainbondVolumeSpecRWX: &rainbondv1alpha1.RainbondVolumeSpec{},
		},
		Status: rainbondv1alpha1.RainbondClusterStatus{
			Conditions: []rainbondv1alpha1.RainbondClusterCondition{
				{
					Type:    rainbondv1alpha1.RainbondClusterConditionTypeDNS,
					Status:  corev1.ConditionFalse,
					Reason:  "DNSFailed",
					Message: "historical online-mode DNS failure",
				},
			},
		},
	}

	k8sClient := &clusterStatusTestClient{
		scheme: scheme,
		nodes: []corev1.Node{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "node-a"},
				Status: corev1.NodeStatus{
					Allocatable: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("4Gi"),
					},
					NodeInfo: corev1.NodeSystemInfo{
						KubeletVersion: "v1.20.0",
					},
				},
			},
		},
		components: append(readyRbdComponents("rbd-system"), notReadyRbdComponent("rbd-system", "rbd-api")),
	}
	mgr := NewClusterMgr(context.Background(), k8sClient, ctrl.Log.WithName("test"), cluster, scheme)

	status, err := mgr.GenerateRainbondClusterStatus()
	if err != nil {
		t.Fatalf("generate status: %v", err)
	}

	_, running := status.GetCondition(rainbondv1alpha1.RainbondClusterConditionTypeRunning)
	if running == nil {
		t.Fatal("expected Running condition")
	}
	if running.Reason != "RbdComponentNotReady" {
		t.Fatalf("expected component readiness reason, got %q with message %q", running.Reason, running.Message)
	}
	if strings.Contains(running.Message, "DNS") {
		t.Fatalf("expected Running message not to include irrelevant historical DNS failure, got %q", running.Message)
	}
}

func readyRbdComponents(namespace string) []rainbondv1alpha1.RbdComponent {
	names := []string{
		"rbd-db",
		"rbd-etcd",
		"rbd-gateway",
		"rbd-hub",
		"rbd-mq",
		"rbd-monitor",
		"rbd-node",
		"rbd-webcli",
		"rbd-worker",
	}

	components := make([]rainbondv1alpha1.RbdComponent, 0, len(names))
	for _, name := range names {
		components = append(components, rainbondv1alpha1.RbdComponent{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Status: rainbondv1alpha1.RbdComponentStatus{
				Conditions: []rainbondv1alpha1.RbdComponentCondition{
					{
						Type:   rainbondv1alpha1.RbdComponentReady,
						Status: corev1.ConditionTrue,
					},
				},
			},
		})
	}
	return components
}

func notReadyRbdComponent(namespace, name string) rainbondv1alpha1.RbdComponent {
	return rainbondv1alpha1.RbdComponent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Status: rainbondv1alpha1.RbdComponentStatus{
			Conditions: []rainbondv1alpha1.RbdComponentCondition{
				{
					Type:   rainbondv1alpha1.RbdComponentReady,
					Status: corev1.ConditionFalse,
				},
			},
		},
	}
}

type clusterStatusTestClient struct {
	scheme     *runtime.Scheme
	nodes      []corev1.Node
	components []rainbondv1alpha1.RbdComponent
}

func (c *clusterStatusTestClient) Get(_ context.Context, key client.ObjectKey, obj client.Object) error {
	switch obj.(type) {
	case *corev1.Secret:
		return apierrors.NewNotFound(schema.GroupResource{Resource: "secrets"}, key.Name)
	default:
		return apierrors.NewBadRequest("unsupported object type")
	}
}

func (c *clusterStatusTestClient) List(_ context.Context, list client.ObjectList, _ ...client.ListOption) error {
	switch out := list.(type) {
	case *corev1.NodeList:
		out.Items = append([]corev1.Node(nil), c.nodes...)
		return nil
	case *storagev1.StorageClassList:
		out.Items = nil
		return nil
	case *rainbondv1alpha1.RbdComponentList:
		out.Items = append([]rainbondv1alpha1.RbdComponent(nil), c.components...)
		return nil
	default:
		return apierrors.NewBadRequest("unsupported list type")
	}
}

func (c *clusterStatusTestClient) Create(context.Context, client.Object, ...client.CreateOption) error {
	panic("unexpected Create call in test")
}

func (c *clusterStatusTestClient) Delete(context.Context, client.Object, ...client.DeleteOption) error {
	panic("unexpected Delete call in test")
}

func (c *clusterStatusTestClient) Update(context.Context, client.Object, ...client.UpdateOption) error {
	panic("unexpected Update call in test")
}

func (c *clusterStatusTestClient) Patch(context.Context, client.Object, client.Patch, ...client.PatchOption) error {
	panic("unexpected Patch call in test")
}

func (c *clusterStatusTestClient) DeleteAllOf(context.Context, client.Object, ...client.DeleteAllOfOption) error {
	panic("unexpected DeleteAllOf call in test")
}

func (c *clusterStatusTestClient) Status() client.StatusWriter {
	return c
}

func (c *clusterStatusTestClient) Scheme() *runtime.Scheme {
	return c.scheme
}

func (c *clusterStatusTestClient) RESTMapper() meta.RESTMapper {
	return nil
}
