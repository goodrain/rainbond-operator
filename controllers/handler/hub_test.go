package handler

import (
	"context"
	"testing"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestHostsJobToleratesTaintedNodesAndSpreadsOnlyAcrossHostsJobPods(t *testing.T) {
	t.Parallel()

	component := &rainbondv1alpha1.RbdComponent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      HubName,
			Namespace: "rbd-system",
		},
	}
	cluster := &rainbondv1alpha1.RainbondCluster{
		Spec: rainbondv1alpha1.RainbondClusterSpec{
			NodesForGateway: []*rainbondv1alpha1.K8sNode{
				{
					Name:       "gateway-1",
					InternalIP: "10.0.0.10",
				},
			},
		},
	}

	k8sClient := &nodeListClient{
		staticClient: &staticClient{
			scheme:  runtime.NewScheme(),
			objects: map[client.ObjectKey]client.Object{},
		},
		nodes: []corev1.Node{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "worker-1",
					Labels: map[string]string{
						"kubernetes.io/hostname": "worker-1",
					},
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "master-1",
					Labels: map[string]string{
						"kubernetes.io/hostname": "master-1",
					},
				},
				Spec: corev1.NodeSpec{
					Taints: []corev1.Taint{
						{
							Key:    "CriticalAddonsOnly",
							Value:  "true",
							Effect: corev1.TaintEffectNoExecute,
						},
					},
				},
			},
		},
	}

	handler := &hub{
		ctx:       context.Background(),
		client:    k8sClient,
		component: component,
		cluster:   cluster,
		labels:    LabelsForRainbondComponent(component),
	}

	job, ok := handler.hostsJob().(*batchv1.Job)
	if !ok {
		t.Fatalf("expected *batchv1.Job, got %T", handler.hostsJob())
	}
	if job.Spec.Parallelism == nil || *job.Spec.Parallelism != 2 {
		t.Fatalf("expected parallelism 2, got %v", job.Spec.Parallelism)
	}
	if job.Spec.Completions == nil || *job.Spec.Completions != 2 {
		t.Fatalf("expected completions 2, got %v", job.Spec.Completions)
	}

	if len(job.Spec.Template.Spec.Tolerations) != 1 {
		t.Fatalf("expected one toleration for tainted nodes, got %d", len(job.Spec.Template.Spec.Tolerations))
	}
	if got := job.Spec.Template.Spec.Tolerations[0].Operator; got != corev1.TolerationOpExists {
		t.Fatalf("expected toleration operator %q, got %q", corev1.TolerationOpExists, got)
	}
	if got := job.Spec.Template.Labels["rainbond.io/hosts-job"]; got != "true" {
		t.Fatalf("expected hosts-job template label rainbond.io/hosts-job=true, got %q", got)
	}

	affinity := job.Spec.Template.Spec.Affinity
	if affinity == nil || affinity.PodAntiAffinity == nil {
		t.Fatalf("expected pod anti-affinity to keep one job pod per hostname")
	}
	terms := affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution
	if len(terms) != 1 {
		t.Fatalf("expected one required pod anti-affinity term, got %d", len(terms))
	}
	if got := terms[0].TopologyKey; got != "kubernetes.io/hostname" {
		t.Fatalf("expected pod anti-affinity topology key kubernetes.io/hostname, got %q", got)
	}
	if terms[0].LabelSelector == nil {
		t.Fatalf("expected pod anti-affinity label selector to be configured")
	}
	if got := terms[0].LabelSelector.MatchLabels["rainbond.io/hosts-job"]; got != "true" {
		t.Fatalf("expected pod anti-affinity selector rainbond.io/hosts-job=true, got %q", got)
	}
	if _, ok := terms[0].LabelSelector.MatchLabels["name"]; ok {
		t.Fatalf("expected pod anti-affinity selector not to match shared component labels, got %v", terms[0].LabelSelector.MatchLabels)
	}
}

type nodeListClient struct {
	*staticClient
	nodes []corev1.Node
}

func (n *nodeListClient) List(_ context.Context, obj client.ObjectList, _ ...client.ListOption) error {
	switch out := obj.(type) {
	case *corev1.NodeList:
		out.Items = append(out.Items, n.nodes...)
		return nil
	default:
		panic("unexpected List call in test")
	}
}
