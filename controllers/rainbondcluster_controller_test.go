package controllers

import (
	"context"
	"testing"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
	clustermgr "github.com/goodrain/rainbond-operator/controllers/cluster-mgr"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestRainbondClusterReconcileRequeuesAfterCreatingImagePullSecret(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("add corev1 to scheme: %v", err)
	}
	if err := rainbondv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("add rainbondv1alpha1 to scheme: %v", err)
	}

	cluster := &rainbondv1alpha1.RainbondCluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rainbond.io/v1alpha1",
			Kind:       "RainbondCluster",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rainbondcluster",
			Namespace: "rbd-system",
		},
		Spec: rainbondv1alpha1.RainbondClusterSpec{
			InstallMode:           rainbondv1alpha1.InstallationModeOffline,
			SuffixHTTPHost:        "172.16.0.1.nip.io",
			GatewayIngressIPs:     []string{"172.16.0.1"},
			NodesForGateway:       []*rainbondv1alpha1.K8sNode{{Name: "node-a", InternalIP: "172.16.0.1"}},
			NodesForChaos:         []*rainbondv1alpha1.K8sNode{{Name: "node-a", InternalIP: "172.16.0.1"}},
			RainbondVolumeSpecRWX: &rainbondv1alpha1.RainbondVolumeSpec{},
			ImageHub: &rainbondv1alpha1.ImageHub{
				Domain:   "goodrain.me",
				Username: "admin",
				Password: "admin1234",
			},
		},
	}

	k8sClient := &rainbondClusterReconcileTestClient{
		scheme:  scheme,
		cluster: cluster,
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
		components: readyRainbondClusterReconcileComponents("rbd-system"),
		secrets:    map[string]*corev1.Secret{},
	}
	reconciler := &RainbondClusterReconciler{
		Client: k8sClient,
		Log:    ctrl.Log.WithName("test"),
		Scheme: scheme,
	}

	result, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: "rbd-system",
			Name:      "rainbondcluster",
		},
	})
	if err != nil {
		t.Fatalf("reconcile returned error: %v", err)
	}
	if !result.Requeue {
		t.Fatalf("expected reconcile to requeue after creating image pull secret, got %#v", result)
	}

	if _, ok := k8sClient.secrets["rbd-system/"+clustermgr.RdbHubCredentialsName]; !ok {
		t.Fatal("expected image pull secret to be created")
	}
	if k8sClient.cluster.Status.ImagePullSecret != nil {
		t.Fatalf("expected first status update to be generated before secret creation, got %#v", k8sClient.cluster.Status.ImagePullSecret)
	}
}

func readyRainbondClusterReconcileComponents(namespace string) []rainbondv1alpha1.RbdComponent {
	names := []string{
		"rbd-chaos",
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

type rainbondClusterReconcileTestClient struct {
	scheme     *runtime.Scheme
	cluster    *rainbondv1alpha1.RainbondCluster
	nodes      []corev1.Node
	components []rainbondv1alpha1.RbdComponent
	secrets    map[string]*corev1.Secret
}

func (c *rainbondClusterReconcileTestClient) Get(_ context.Context, key client.ObjectKey, obj client.Object) error {
	switch out := obj.(type) {
	case *rainbondv1alpha1.RainbondCluster:
		if c.cluster == nil || key.Namespace != c.cluster.Namespace || key.Name != c.cluster.Name {
			return apierrors.NewNotFound(schema.GroupResource{Resource: "rainbondclusters"}, key.Name)
		}
		c.cluster.DeepCopyInto(out)
		return nil
	case *corev1.Secret:
		if secret, ok := c.secrets[key.Namespace+"/"+key.Name]; ok {
			secret.DeepCopyInto(out)
			return nil
		}
		return apierrors.NewNotFound(schema.GroupResource{Resource: "secrets"}, key.Name)
	default:
		return apierrors.NewBadRequest("unsupported object type")
	}
}

func (c *rainbondClusterReconcileTestClient) List(_ context.Context, list client.ObjectList, _ ...client.ListOption) error {
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

func (c *rainbondClusterReconcileTestClient) Create(_ context.Context, obj client.Object, _ ...client.CreateOption) error {
	if secret, ok := obj.(*corev1.Secret); ok {
		key := secret.Namespace + "/" + secret.Name
		if _, exists := c.secrets[key]; exists {
			return apierrors.NewAlreadyExists(schema.GroupResource{Resource: "secrets"}, secret.Name)
		}
		c.secrets[key] = secret.DeepCopy()
		return nil
	}
	return apierrors.NewBadRequest("unsupported create type")
}

func (c *rainbondClusterReconcileTestClient) Delete(context.Context, client.Object, ...client.DeleteOption) error {
	panic("unexpected Delete call in test")
}

func (c *rainbondClusterReconcileTestClient) Update(_ context.Context, obj client.Object, _ ...client.UpdateOption) error {
	switch out := obj.(type) {
	case *rainbondv1alpha1.RainbondCluster:
		c.cluster = out.DeepCopy()
		return nil
	case *corev1.Secret:
		c.secrets[out.Namespace+"/"+out.Name] = out.DeepCopy()
		return nil
	default:
		return apierrors.NewBadRequest("unsupported update type")
	}
}

func (c *rainbondClusterReconcileTestClient) Patch(context.Context, client.Object, client.Patch, ...client.PatchOption) error {
	panic("unexpected Patch call in test")
}

func (c *rainbondClusterReconcileTestClient) DeleteAllOf(context.Context, client.Object, ...client.DeleteAllOfOption) error {
	panic("unexpected DeleteAllOf call in test")
}

func (c *rainbondClusterReconcileTestClient) Status() client.StatusWriter {
	return c
}

func (c *rainbondClusterReconcileTestClient) Scheme() *runtime.Scheme {
	return c.scheme
}

func (c *rainbondClusterReconcileTestClient) RESTMapper() meta.RESTMapper {
	return nil
}
