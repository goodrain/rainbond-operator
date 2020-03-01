package rbdcomponent

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/pkg/apis/rainbond/v1alpha1"
	"github.com/goodrain/rainbond-operator/pkg/util/constants"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const maxNumEventsPerTest = 10

func TestUnsupportedRbdComponent(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = rainbondv1alpha1.AddToScheme(scheme)

	cptName := "foobar"
	cpt := &rainbondv1alpha1.RbdComponent{
		ObjectMeta: metav1.ObjectMeta{
			Name: cptName,
		},
	}
	cli := fake.NewFakeClientWithScheme(scheme, cpt)
	recorder := record.NewFakeRecorder(maxNumEventsPerTest)

	r := &ReconcileRbdComponent{client: cli, recorder: recorder}

	request := reconcile.Request{}
	request.Name = cptName
	res, err := r.Reconcile(request)
	assert.EqualValues(t, reconcile.Result{}, res)
	assert.Nil(t, err)

	expectedEvents := []string{
		fmt.Sprintf("Warning UnsupportedType only supports the following types of rbdcomponent: %s", supportedComponents()),
	}

	events := []string{}
	numEvents := len(recorder.Events)
	for i := 0; i < numEvents; i++ {
		event := <-recorder.Events
		events = append(events, event)
	}
	assert.Equal(t, expectedEvents, events)

	reason := "UnsupportedType"
	msg := fmt.Sprintf("only supports the following types of rbdcomponent: %s", supportedComponents())
	expectedCondition := rainbondv1alpha1.NewRbdComponentCondition(rainbondv1alpha1.RbdComponentReady, corev1.ConditionFalse, reason, msg)
	dummytime := metav1.Now()
	expectedCondition.LastTransitionTime = dummytime
	newCpt := &rainbondv1alpha1.RbdComponent{}
	err = cli.Get(context.TODO(), types.NamespacedName{Name: "foobar"}, newCpt)
	assert.Nil(t, err)
	assert.NotNil(t, newCpt.Status)
	_, condition := newCpt.Status.GetCondition(rainbondv1alpha1.RbdComponentReady)
	assert.NotNil(t, condition)
	condition.LastTransitionTime = dummytime
	assert.Equal(t, expectedCondition, condition)
}

func TestRbdComponentClusterNotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = rainbondv1alpha1.AddToScheme(scheme)

	cptName := "rbd-api"
	cpt := &rainbondv1alpha1.RbdComponent{
		ObjectMeta: metav1.ObjectMeta{
			Name: cptName,
		},
	}
	cli := fake.NewFakeClientWithScheme(scheme, cpt)
	recorder := record.NewFakeRecorder(maxNumEventsPerTest)

	r := &ReconcileRbdComponent{client: cli, recorder: recorder}

	request := reconcile.Request{}
	request.Name = cptName
	res, err := r.Reconcile(request)
	assert.EqualValues(t, reconcile.Result{RequeueAfter: 3 * time.Second}, res)
	assert.Nil(t, err)

	expectedEvents := []string{
		"Warning ClusterNotFound rainbondcluster not found",
	}

	events := []string{}
	numEvents := len(recorder.Events)
	for i := 0; i < numEvents; i++ {
		event := <-recorder.Events
		events = append(events, event)
	}
	assert.Equal(t, expectedEvents, events)

	reason := "ClusterNotFound"
	msg := "rainbondcluster not found"
	expectedCondition := rainbondv1alpha1.NewRbdComponentCondition(rainbondv1alpha1.ClusterConfigCompeleted, corev1.ConditionFalse, reason, msg)
	dummytime := metav1.Now()
	expectedCondition.LastTransitionTime = dummytime
	newCpt := &rainbondv1alpha1.RbdComponent{}
	err = cli.Get(context.TODO(), types.NamespacedName{Name: cptName}, newCpt)
	assert.Nil(t, err)
	assert.NotNil(t, newCpt.Status)
	_, condition := newCpt.Status.GetCondition(rainbondv1alpha1.ClusterConfigCompeleted)
	assert.NotNil(t, condition)
	condition.LastTransitionTime = dummytime
	assert.Equal(t, expectedCondition, condition)
}

func TestRbdComponentPackageNotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = rainbondv1alpha1.AddToScheme(scheme)

	cptName := "rbd-api"
	cpt := &rainbondv1alpha1.RbdComponent{
		ObjectMeta: metav1.ObjectMeta{
			Name: cptName,
		},
	}
	cluster := &rainbondv1alpha1.RainbondCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: constants.RainbondClusterName,
		},
		Spec: rainbondv1alpha1.RainbondClusterSpec{
			ConfigCompleted: true,
		},
	}
	cli := fake.NewFakeClientWithScheme(scheme, cpt, cluster)
	recorder := record.NewFakeRecorder(maxNumEventsPerTest)

	r := &ReconcileRbdComponent{client: cli, recorder: recorder}

	request := reconcile.Request{}
	request.Name = cptName
	res, err := r.Reconcile(request)
	assert.EqualValues(t, reconcile.Result{RequeueAfter: 3 * time.Second}, res)
	if !assert.Nil(t, err) {
		t.FailNow()
	}

	expectedEvents := []string{
		"Warning PackageNotFound rainbondpackage not found",
	}

	events := []string{}
	numEvents := len(recorder.Events)
	for i := 0; i < numEvents; i++ {
		event := <-recorder.Events
		events = append(events, event)
	}
	assert.Equal(t, expectedEvents, events)

	reason := "PackageNotFound"
	msg := "rainbondpackage not found"
	expectedCondition := rainbondv1alpha1.NewRbdComponentCondition(rainbondv1alpha1.RainbondPackageReady, corev1.ConditionFalse, reason, msg)
	dummytime := metav1.Now()
	expectedCondition.LastTransitionTime = dummytime
	newCpt := &rainbondv1alpha1.RbdComponent{}
	err = cli.Get(context.TODO(), types.NamespacedName{Name: cptName}, newCpt)
	assert.Nil(t, err)
	assert.NotNil(t, newCpt.Status)
	_, condition := newCpt.Status.GetCondition(rainbondv1alpha1.RainbondPackageReady)
	if !assert.NotNil(t, condition) {
		t.FailNow()
	}
	condition.LastTransitionTime = dummytime
	assert.Equal(t, expectedCondition, condition)

	// config completed
	expectedCondition = rainbondv1alpha1.NewRbdComponentCondition(rainbondv1alpha1.ClusterConfigCompeleted, corev1.ConditionTrue, "ConfigCompleted", "")
	expectedCondition.LastTransitionTime = dummytime
	_, condition = newCpt.Status.GetCondition(rainbondv1alpha1.ClusterConfigCompeleted)
	if !assert.NotNil(t, condition) {
		t.FailNow()
	}
	condition.LastTransitionTime = dummytime
	assert.Equal(t, expectedCondition, condition)
}

func TestRbdComponentPrerequisitesFailed(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = rainbondv1alpha1.AddToScheme(scheme)

	cptName := "rbd-api"
	cpt := &rainbondv1alpha1.RbdComponent{
		ObjectMeta: metav1.ObjectMeta{
			Name: cptName,
		},
	}
	cluster := &rainbondv1alpha1.RainbondCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: constants.RainbondClusterName,
		},
		Spec: rainbondv1alpha1.RainbondClusterSpec{
			ConfigCompleted: true,
		},
	}
	pkg := &rainbondv1alpha1.RainbondPackage{
		ObjectMeta: metav1.ObjectMeta{
			Name: constants.RainbondPackageName,
		},
		Status: &rainbondv1alpha1.RainbondPackageStatus{
			Conditions: []rainbondv1alpha1.PackageCondition{
				{
					Type:   rainbondv1alpha1.Ready,
					Status: rainbondv1alpha1.Running,
				},
			},
		},
	}
	cli := fake.NewFakeClientWithScheme(scheme, cpt, cluster, pkg)
	recorder := record.NewFakeRecorder(maxNumEventsPerTest)

	r := &ReconcileRbdComponent{client: cli, recorder: recorder}

	request := reconcile.Request{}
	request.Name = cptName
	res, err := r.Reconcile(request)
	assert.EqualValues(t, reconcile.Result{RequeueAfter: 3 * time.Second}, res)
	if !assert.Nil(t, err) {
		t.FailNow()
	}

	reason := "PrerequisitesFailed"
	msg := "failed to check prerequisites"
	expectedEvents := []string{
		fmt.Sprintf("Warning %s %s", reason, msg),
	}

	events := []string{}
	numEvents := len(recorder.Events)
	for i := 0; i < numEvents; i++ {
		event := <-recorder.Events
		events = append(events, event)
	}
	assert.Equal(t, expectedEvents, events)

	dummytime := metav1.Now()
	expectedCondition := rainbondv1alpha1.NewRbdComponentCondition(rainbondv1alpha1.RbdComponentReady, corev1.ConditionFalse, reason, msg)
	expectedCondition.LastTransitionTime = dummytime
	newCpt := &rainbondv1alpha1.RbdComponent{}
	err = cli.Get(context.TODO(), types.NamespacedName{Name: cptName}, newCpt)
	assert.Nil(t, err)
	assert.NotNil(t, newCpt.Status)
	_, condition := newCpt.Status.GetCondition(rainbondv1alpha1.RbdComponentReady)
	if !assert.NotNil(t, condition) {
		t.FailNow()
	}
	condition.LastTransitionTime = dummytime
	assert.Equal(t, expectedCondition, condition)

	// config completed
	expectedCondition = rainbondv1alpha1.NewRbdComponentCondition(rainbondv1alpha1.ClusterConfigCompeleted, corev1.ConditionTrue, "ConfigCompleted", "")
	expectedCondition.LastTransitionTime = dummytime
	_, condition = newCpt.Status.GetCondition(rainbondv1alpha1.ClusterConfigCompeleted)
	if !assert.NotNil(t, condition) {
		t.FailNow()
	}
	condition.LastTransitionTime = dummytime
	assert.Equal(t, expectedCondition, condition)

	// package not ready
	expectedCondition = rainbondv1alpha1.NewRbdComponentCondition(rainbondv1alpha1.RainbondPackageReady, corev1.ConditionFalse, "PackageNotReady", "")
	expectedCondition.LastTransitionTime = dummytime
	_, condition = newCpt.Status.GetCondition(rainbondv1alpha1.RainbondPackageReady)
	if !assert.NotNil(t, condition) {
		t.FailNow()
	}
	condition.LastTransitionTime = dummytime
	assert.Equal(t, expectedCondition, condition)
}

func TestRbdComponentBeforeFailed(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = rainbondv1alpha1.AddToScheme(scheme)

	cptName := "rbd-api"
	cpt := &rainbondv1alpha1.RbdComponent{
		ObjectMeta: metav1.ObjectMeta{
			Name: cptName,
		},
	}
	cluster := &rainbondv1alpha1.RainbondCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: constants.RainbondClusterName,
		},
		Spec: rainbondv1alpha1.RainbondClusterSpec{
			ConfigCompleted: true,
		},
	}
	pkg := &rainbondv1alpha1.RainbondPackage{
		ObjectMeta: metav1.ObjectMeta{
			Name: constants.RainbondPackageName,
		},
		Status: &rainbondv1alpha1.RainbondPackageStatus{
			Conditions: []rainbondv1alpha1.PackageCondition{
				{
					Type:   rainbondv1alpha1.Ready,
					Status: rainbondv1alpha1.Completed,
				},
			},
		},
	}
	cli := fake.NewFakeClientWithScheme(scheme, cpt, cluster, pkg)
	recorder := record.NewFakeRecorder(maxNumEventsPerTest)

	r := &ReconcileRbdComponent{client: cli, recorder: recorder}

	request := reconcile.Request{}
	request.Name = cptName
	res, err := r.Reconcile(request)
	assert.EqualValues(t, reconcile.Result{RequeueAfter: 3 * time.Second}, res)
	if !assert.Nil(t, err) {
		t.FailNow()
	}

	reason := "PrerequisitesFailed"
	msg := ""
	expectedEvent := fmt.Sprintf("Warning %s %s", reason, msg)

	var events []string
	numEvents := len(recorder.Events)
	for i := 0; i < numEvents; i++ {
		event := <-recorder.Events
		events = append(events, event)
	}

	assert.True(t, strings.Contains(events[0], expectedEvent))

	dummytime := metav1.Now()
	expectedCondition := rainbondv1alpha1.NewRbdComponentCondition(rainbondv1alpha1.RbdComponentReady, corev1.ConditionFalse, reason, msg)
	expectedCondition.LastTransitionTime = dummytime
	newCpt := &rainbondv1alpha1.RbdComponent{}
	err = cli.Get(context.TODO(), types.NamespacedName{Name: cptName}, newCpt)
	assert.Nil(t, err)
	assert.NotNil(t, newCpt.Status)
	_, condition := newCpt.Status.GetCondition(rainbondv1alpha1.RbdComponentReady)
	if !assert.NotNil(t, condition) {
		t.FailNow()
	}
	// ignroe LastTransitionTime and Message
	condition.LastTransitionTime = dummytime
	condition.Message = ""
	assert.Equal(t, expectedCondition, condition)

	// config completed
	expectedCondition = rainbondv1alpha1.NewRbdComponentCondition(rainbondv1alpha1.ClusterConfigCompeleted, corev1.ConditionTrue, "ConfigCompleted", "")
	expectedCondition.LastTransitionTime = dummytime
	_, condition = newCpt.Status.GetCondition(rainbondv1alpha1.ClusterConfigCompeleted)
	if !assert.NotNil(t, condition) {
		t.FailNow()
	}
	condition.LastTransitionTime = dummytime
	assert.Equal(t, expectedCondition, condition)

	// package not ready
	expectedCondition = rainbondv1alpha1.NewRbdComponentCondition(rainbondv1alpha1.RainbondPackageReady, corev1.ConditionTrue, "PackageReady", "")
	expectedCondition.LastTransitionTime = dummytime
	_, condition = newCpt.Status.GetCondition(rainbondv1alpha1.RainbondPackageReady)
	if !assert.NotNil(t, condition) {
		t.FailNow()
	}
	condition.LastTransitionTime = dummytime
	assert.Equal(t, expectedCondition, condition)
}
