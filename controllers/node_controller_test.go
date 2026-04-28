package controllers

import (
	"context"
	"testing"
	"time"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestNodeReconcilerDeletesHostsJobBeforeTriggeringHubReconcile(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("add corev1 to scheme: %v", err)
	}
	if err := batchv1.AddToScheme(scheme); err != nil {
		t.Fatalf("add batchv1 to scheme: %v", err)
	}
	if err := rainbondv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("add rainbondv1alpha1 to scheme: %v", err)
	}

	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "new-node",
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Type:   corev1.NodeReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}
	cluster := &rainbondv1alpha1.RainbondCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rainbondcluster",
			Namespace: "rbd-system",
		},
	}
	hub := &rainbondv1alpha1.RbdComponent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rbd-hub",
			Namespace: "rbd-system",
		},
	}
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "hosts-job",
			Namespace: "rbd-system",
		},
	}

	k8sClient := &nodeControllerTestClient{
		scheme:  scheme,
		node:    node,
		cluster: cluster,
		hub:     hub,
		job:     job,
	}

	reconciler := &NodeReconciler{
		Client: k8sClient,
		Log:    ctrl.Log.WithName("test"),
		Scheme: scheme,
	}

	result, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: node.Name},
	})
	if err != nil {
		t.Fatalf("reconcile returned error: %v", err)
	}
	if result.RequeueAfter != 2*time.Second {
		t.Fatalf("expected requeue after deleting hosts-job, got %#v", result)
	}

	updatedHub := &rainbondv1alpha1.RbdComponent{}
	if err := k8sClient.Get(context.Background(), types.NamespacedName{
		Namespace: "rbd-system",
		Name:      "rbd-hub",
	}, updatedHub); err != nil {
		t.Fatalf("get updated hub: %v", err)
	}
	if updatedHub.Annotations["rainbond.io/node-change-time"] != "" {
		t.Fatalf("expected hub annotation to remain untouched until hosts-job is fully removed")
	}
}

func TestNodeReconcilerTriggersHubReconcileWhenHostsJobIsGone(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("add corev1 to scheme: %v", err)
	}
	if err := batchv1.AddToScheme(scheme); err != nil {
		t.Fatalf("add batchv1 to scheme: %v", err)
	}
	if err := rainbondv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("add rainbondv1alpha1 to scheme: %v", err)
	}

	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "new-node",
		},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{
					Type:   corev1.NodeReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}
	cluster := &rainbondv1alpha1.RainbondCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rainbondcluster",
			Namespace: "rbd-system",
		},
	}
	hub := &rainbondv1alpha1.RbdComponent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rbd-hub",
			Namespace: "rbd-system",
		},
	}

	k8sClient := &nodeControllerTestClient{
		scheme:  scheme,
		node:    node,
		cluster: cluster,
		hub:     hub,
	}

	reconciler := &NodeReconciler{
		Client: k8sClient,
		Log:    ctrl.Log.WithName("test"),
		Scheme: scheme,
	}

	result, err := reconciler.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: node.Name},
	})
	if err != nil {
		t.Fatalf("reconcile returned error: %v", err)
	}
	if result.Requeue || result.RequeueAfter != 0 {
		t.Fatalf("expected no extra requeue when hosts-job is already gone, got %#v", result)
	}

	updatedHub := &rainbondv1alpha1.RbdComponent{}
	if err := k8sClient.Get(context.Background(), types.NamespacedName{
		Namespace: "rbd-system",
		Name:      "rbd-hub",
	}, updatedHub); err != nil {
		t.Fatalf("get updated hub: %v", err)
	}
	if updatedHub.Annotations["rainbond.io/node-change-time"] == "" {
		t.Fatalf("expected hub annotation to be updated once hosts-job is gone")
	}
}

type nodeControllerTestClient struct {
	scheme  *runtime.Scheme
	node    *corev1.Node
	cluster *rainbondv1alpha1.RainbondCluster
	hub     *rainbondv1alpha1.RbdComponent
	job     *batchv1.Job
}

func (c *nodeControllerTestClient) Get(_ context.Context, key client.ObjectKey, obj client.Object) error {
	switch out := obj.(type) {
	case *corev1.Node:
		if c.node == nil || key.Name != c.node.Name {
			return apierrors.NewNotFound(schema.GroupResource{Resource: "nodes"}, key.Name)
		}
		c.node.DeepCopyInto(out)
		return nil
	case *batchv1.Job:
		if c.job == nil || key.Name != c.job.Name || key.Namespace != c.job.Namespace {
			return apierrors.NewNotFound(schema.GroupResource{Resource: "jobs"}, key.Name)
		}
		c.job.DeepCopyInto(out)
		return nil
	case *rainbondv1alpha1.RbdComponent:
		if c.hub == nil || key.Name != c.hub.Name || key.Namespace != c.hub.Namespace {
			return apierrors.NewNotFound(schema.GroupResource{Resource: "rbdcomponents"}, key.Name)
		}
		c.hub.DeepCopyInto(out)
		return nil
	default:
		return apierrors.NewBadRequest("unsupported object type")
	}
}

func (c *nodeControllerTestClient) List(_ context.Context, obj client.ObjectList, _ ...client.ListOption) error {
	switch out := obj.(type) {
	case *rainbondv1alpha1.RainbondClusterList:
		if c.cluster != nil {
			out.Items = []rainbondv1alpha1.RainbondCluster{*c.cluster.DeepCopy()}
		}
		return nil
	default:
		return apierrors.NewBadRequest("unsupported list type")
	}
}

func (c *nodeControllerTestClient) Create(context.Context, client.Object, ...client.CreateOption) error {
	panic("unexpected Create call in test")
}

func (c *nodeControllerTestClient) Delete(_ context.Context, obj client.Object, _ ...client.DeleteOption) error {
	switch target := obj.(type) {
	case *batchv1.Job:
		if c.job == nil || c.job.Name != target.Name || c.job.Namespace != target.Namespace {
			return apierrors.NewNotFound(schema.GroupResource{Resource: "jobs"}, target.Name)
		}
		c.job = nil
		return nil
	default:
		return apierrors.NewBadRequest("unsupported delete type")
	}
}

func (c *nodeControllerTestClient) Update(_ context.Context, obj client.Object, _ ...client.UpdateOption) error {
	switch in := obj.(type) {
	case *rainbondv1alpha1.RbdComponent:
		if c.hub == nil || c.hub.Name != in.Name || c.hub.Namespace != in.Namespace {
			return apierrors.NewNotFound(schema.GroupResource{Resource: "rbdcomponents"}, in.Name)
		}
		c.hub = in.DeepCopy()
		return nil
	default:
		return apierrors.NewBadRequest("unsupported update type")
	}
}

func (c *nodeControllerTestClient) Patch(context.Context, client.Object, client.Patch, ...client.PatchOption) error {
	panic("unexpected Patch call in test")
}

func (c *nodeControllerTestClient) DeleteAllOf(context.Context, client.Object, ...client.DeleteAllOfOption) error {
	panic("unexpected DeleteAllOf call in test")
}

func (c *nodeControllerTestClient) Status() client.StatusWriter {
	return nodeControllerTestStatusWriter{}
}

func (c *nodeControllerTestClient) Scheme() *runtime.Scheme {
	return c.scheme
}

func (c *nodeControllerTestClient) RESTMapper() meta.RESTMapper {
	return nil
}

type nodeControllerTestStatusWriter struct{}

func (nodeControllerTestStatusWriter) Update(context.Context, client.Object, ...client.UpdateOption) error {
	panic("unexpected Status().Update call in test")
}

func (nodeControllerTestStatusWriter) Patch(context.Context, client.Object, client.Patch, ...client.PatchOption) error {
	panic("unexpected Status().Patch call in test")
}
