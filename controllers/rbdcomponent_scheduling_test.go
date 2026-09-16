package controllers

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestApplyComponentTolerations(t *testing.T) {
	seconds := int64(30)
	configured := []corev1.Toleration{
		{Key: "node.kubernetes.io/unschedulable", Operator: corev1.TolerationOpExists, Effect: corev1.TaintEffectNoSchedule},
		{Key: "dedicated", Operator: corev1.TolerationOpEqual, Value: "rainbond", Effect: corev1.TaintEffectNoExecute, TolerationSeconds: &seconds},
	}
	deployment := &appsv1.Deployment{}
	statefulSet := &appsv1.StatefulSet{}
	daemonSet := &appsv1.DaemonSet{}
	job := &batchv1.Job{}
	for _, tc := range []struct {
		name    string
		object  client.Object
		podSpec *corev1.PodSpec
	}{
		{"deployment", deployment, &deployment.Spec.Template.Spec},
		{"statefulset", statefulSet, &statefulSet.Spec.Template.Spec},
		{"daemonset", daemonSet, &daemonSet.Spec.Template.Spec},
		{"job", job, &job.Spec.Template.Spec},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defaults := []corev1.Toleration{{Operator: corev1.TolerationOpExists}}
			tc.podSpec.Tolerations = defaults
			for _, empty := range [][]corev1.Toleration{nil, {}} {
				applyComponentTolerations(tc.object, empty)
				if !reflect.DeepEqual(tc.podSpec.Tolerations, defaults) {
					t.Fatal("omitted or empty configuration must preserve component defaults")
				}
			}
			applyComponentTolerations(tc.object, configured)
			applyComponentTolerations(tc.object, configured)
			if !reflect.DeepEqual(tc.podSpec.Tolerations, configured) {
				t.Fatalf("expected exact configured list without duplicate entries, got %v", tc.podSpec.Tolerations)
			}
			tc.podSpec.Tolerations[0].Key = "changed"
			*tc.podSpec.Tolerations[1].TolerationSeconds = 60
			if configured[0].Key != "node.kubernetes.io/unschedulable" || seconds != 30 {
				t.Fatal("pod tolerations must not alias the component configuration")
			}
		})
	}
	service := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: "unchanged"}}
	before := service.DeepCopy()
	applyComponentTolerations(service, configured)
	if !reflect.DeepEqual(service, before) {
		t.Fatal("non-workload resources must remain unchanged")
	}
}

func TestRbdComponentReconcileAppliesScheduling(t *testing.T) {
	t.Setenv("DISABLE_LOG", "true")
	for _, name := range []string{"rbd-mq", "minio", "rbd-monitor"} {
		t.Run(name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			for _, add := range []func(*runtime.Scheme) error{corev1.AddToScheme, appsv1.AddToScheme, rainbondv1alpha1.AddToScheme} {
				if err := add(scheme); err != nil {
					t.Fatal(err)
				}
			}
			component := &rainbondv1alpha1.RbdComponent{
				ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "rbd-system", UID: types.UID(name)},
			}
			if err := json.Unmarshal([]byte(`{"tolerations":[{"key":"node.kubernetes.io/unschedulable","operator":"Exists","effect":"NoSchedule"}],"affinity":{"nodeAffinity":{"requiredDuringSchedulingIgnoredDuringExecution":{"nodeSelectorTerms":[{"matchFields":[{"key":"metadata.name","operator":"In","values":["node-a"]}]}]}}}}`), &component.Spec); err != nil {
				t.Fatal(err)
			}
			cluster := &rainbondv1alpha1.RainbondCluster{
				ObjectMeta: metav1.ObjectMeta{Name: "rainbondcluster", Namespace: component.Namespace},
			}
			k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(component, cluster).Build()
			reconciler := &RbdComponentReconciler{
				Client: k8sClient, Scheme: scheme, Log: ctrl.Log.WithName("test"), Recorder: record.NewFakeRecorder(100),
			}
			ctx := context.Background()
			request := ctrl.Request{NamespacedName: client.ObjectKeyFromObject(component)}
			for _, key := range []string{"node.kubernetes.io/unschedulable", "dedicated", ""} {
				if err := k8sClient.Get(ctx, request.NamespacedName, component); err != nil {
					t.Fatal(err)
				}
				// Decode through the public API so this test also detects dropped fields.
				var want []corev1.Toleration
				if key != "" {
					want = []corev1.Toleration{{Key: key, Operator: corev1.TolerationOpExists, Effect: corev1.TaintEffectNoSchedule}}
				}
				data, err := json.Marshal(map[string]interface{}{"tolerations": want})
				if err != nil {
					t.Fatal(err)
				}
				if err := json.Unmarshal(data, &component.Spec); err != nil {
					t.Fatal(err)
				}
				if err := k8sClient.Update(ctx, component); err != nil {
					t.Fatal(err)
				}
				if _, err := reconciler.Reconcile(ctx, request); err != nil {
					t.Fatalf("reconcile: %v", err)
				}
				var podSpec corev1.PodSpec
				if name == "rbd-mq" {
					var deployment appsv1.Deployment
					if err := k8sClient.Get(ctx, request.NamespacedName, &deployment); err != nil {
						t.Fatal(err)
					}
					podSpec = deployment.Spec.Template.Spec
				} else {
					var statefulSet appsv1.StatefulSet
					if err := k8sClient.Get(ctx, request.NamespacedName, &statefulSet); err != nil {
						t.Fatal(err)
					}
					podSpec = statefulSet.Spec.Template.Spec
				}
				if !reflect.DeepEqual(podSpec.Tolerations, want) {
					t.Errorf("expected reconciled tolerations %v, got %v", want, podSpec.Tolerations)
				}
				if !reflect.DeepEqual(podSpec.Affinity, component.Spec.Affinity) {
					t.Errorf("expected component affinity to reach the pod template, got %+v", podSpec.Affinity)
				}
			}
		})
	}
}

func TestApplyComponentTolerationsDefaultsOperator(t *testing.T) {
	configured := []corev1.Toleration{{Key: "dedicated", Value: "rainbond", Effect: corev1.TaintEffectNoSchedule}}
	statefulSet := &appsv1.StatefulSet{}
	applyComponentTolerations(statefulSet, configured)
	if got := statefulSet.Spec.Template.Spec.Tolerations[0].Operator; got != corev1.TolerationOpEqual {
		t.Fatalf("expected omitted operator to default to Equal before comparing templates, got %q", got)
	}
	if configured[0].Operator != "" {
		t.Fatal("defaulting must not mutate the component configuration")
	}
}
