package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
	v2 "github.com/goodrain/rainbond-operator/api/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func TestDefaultTCPRouteUsesEndpointGranularity(t *testing.T) {
	route := rbdDefaultRouteTemplateForTCP("rbd-app-ui", 7070).(*v2.ApisixRoute)

	if len(route.Spec.Stream) != 1 {
		t.Fatalf("expected one stream route, got %d", len(route.Spec.Stream))
	}
	if got := route.Spec.Stream[0].Backend.ResolveGranularity; got != "endpoint" {
		t.Fatalf("expected endpoint granularity, got %q", got)
	}
	if got := route.Annotations[systemTCPRouteVersionAnnotation]; got != systemTCPRouteVersion {
		t.Fatalf("expected route version annotation %q, got %q", systemTCPRouteVersion, got)
	}
}

func TestDefaultTCPUpstreamConfiguresHealthChecksAndRetry(t *testing.T) {
	upstream := rbdDefaultUpstreamForTCP("rbd-app-ui", 7070).(*v2.ApisixUpstream)

	if upstream.Spec == nil {
		t.Fatal("expected upstream spec")
	}
	if upstream.Spec.Retries == nil || *upstream.Spec.Retries != 1 {
		t.Fatalf("expected one upstream retry, got %v", upstream.Spec.Retries)
	}
	if upstream.Spec.HealthCheck == nil || upstream.Spec.HealthCheck.Active == nil {
		t.Fatal("expected an active health check")
	}
	active := upstream.Spec.HealthCheck.Active
	if active.Type != "tcp" || active.Port != 7070 {
		t.Fatalf("expected TCP health check on port 7070, got type=%q port=%d", active.Type, active.Port)
	}
	if active.Unhealthy == nil || active.Unhealthy.TCPFailures != 2 || active.Unhealthy.Timeouts != 2 {
		t.Fatalf("unexpected active unhealthy thresholds: %#v", active.Unhealthy)
	}
	if upstream.Spec.HealthCheck.Passive == nil || upstream.Spec.HealthCheck.Passive.Type != "tcp" {
		t.Fatalf("expected passive TCP health check, got %#v", upstream.Spec.HealthCheck.Passive)
	}
	raw, err := json.Marshal(upstream)
	if err != nil {
		t.Fatalf("marshal upstream: %v", err)
	}
	if got := bytes.Count(raw, []byte(`"timeouts":`)); got != 2 {
		t.Fatalf("expected active and passive timeouts fields, got %d in %s", got, raw)
	}
	if bytes.Contains(raw, []byte(`"timeout":`)) {
		t.Fatalf("unexpected legacy singular timeout field in %s", raw)
	}
}

func TestAppUIInstallResourcesIncludeRouteAndUpstream(t *testing.T) {
	handler := newAppUIHandlerForTest(nil)

	resources := handler.ResourcesCreateIfNotExists()
	assertRouteAndUpstream(t, resources, "rbd-app-ui", 7070)
}

func TestAPIInstallResourcesIncludeRoutesAndUpstreams(t *testing.T) {
	handler := &api{
		component: &rainbondv1alpha1.RbdComponent{
			ObjectMeta: metav1.ObjectMeta{Name: APIName, Namespace: "rbd-system"},
		},
	}

	resources := handler.ResourcesCreateIfNotExists()
	assertRouteAndUpstream(t, resources, "rbd-api-api", 8443)
	assertRouteAndUpstream(t, resources, "rbd-api-healthz", 8889)
	assertRouteAndUpstream(t, resources, "rbd-api-websocket", 6060)
}

func TestMigrateLegacySystemTCPRoute(t *testing.T) {
	component, route, k8sClient := legacyRouteClient(t, false, true)

	changed, err := migrateLegacySystemTCPRoute(context.Background(), k8sClient, component, "rbd-app-ui", 7070)
	if err != nil {
		t.Fatalf("migrate legacy route: %v", err)
	}
	if !changed {
		t.Fatal("expected legacy route to be migrated")
	}

	updated := &v2.ApisixRoute{}
	if err := k8sClient.Get(context.Background(), client.ObjectKeyFromObject(route), updated); err != nil {
		t.Fatalf("get migrated route: %v", err)
	}
	if got := updated.Spec.Stream[0].Backend.ResolveGranularity; got != "endpoint" {
		t.Fatalf("expected migrated endpoint granularity, got %q", got)
	}
	if got := updated.Annotations[systemTCPRouteVersionAnnotation]; got != systemTCPRouteVersion {
		t.Fatalf("expected migration annotation %q, got %q", systemTCPRouteVersion, got)
	}

	changed, err = migrateLegacySystemTCPRoute(context.Background(), k8sClient, component, "rbd-app-ui", 7070)
	if err != nil {
		t.Fatalf("repeat migration: %v", err)
	}
	if changed {
		t.Fatal("expected repeated migration to be a no-op")
	}
}

func TestMigrateLegacySystemTCPRoutePreservesCustomizedRoute(t *testing.T) {
	component, _, k8sClient := legacyRouteClient(t, true, true)

	changed, err := migrateLegacySystemTCPRoute(context.Background(), k8sClient, component, "rbd-app-ui", 7070)
	if err != nil {
		t.Fatalf("migrate customized route: %v", err)
	}
	if changed {
		t.Fatal("expected customized route to remain unchanged")
	}
}

func TestMigrateLegacySystemTCPRouteIgnoresUnownedRoute(t *testing.T) {
	component, _, k8sClient := legacyRouteClient(t, false, false)

	changed, err := migrateLegacySystemTCPRoute(context.Background(), k8sClient, component, "rbd-app-ui", 7070)
	if err != nil {
		t.Fatalf("migrate unowned route: %v", err)
	}
	if changed {
		t.Fatal("expected unowned route to remain unchanged")
	}
}

func TestAppUIAfterAllowsMissingRouteDuringFastInstall(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := rainbondv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("add Rainbond scheme: %v", err)
	}
	if err := v2.AddToScheme(scheme); err != nil {
		t.Fatalf("add APISIX scheme: %v", err)
	}
	component := &rainbondv1alpha1.RbdComponent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      AppUIName,
			Namespace: "rbd-system",
			UID:       types.UID("component-uid"),
		},
	}
	k8sClient := fake.NewFakeClientWithScheme(scheme, component)
	handler := &appui{ctx: context.Background(), client: k8sClient, component: component}

	if err := handler.After(); err != nil {
		t.Fatalf("expected a missing route to be retried by a later reconcile, got %v", err)
	}
}

func TestAppUIAfterMigratesLegacySystemRoute(t *testing.T) {
	component, route, k8sClient := legacyRouteClient(t, false, true)
	handler := &appui{ctx: context.Background(), client: k8sClient, component: component}

	if err := handler.After(); err != nil {
		t.Fatalf("migrate App UI route: %v", err)
	}

	updated := &v2.ApisixRoute{}
	if err := k8sClient.Get(context.Background(), client.ObjectKeyFromObject(route), updated); err != nil {
		t.Fatalf("get migrated App UI route: %v", err)
	}
	if got := updated.Spec.Stream[0].Backend.ResolveGranularity; got != "endpoint" {
		t.Fatalf("expected App UI route to use endpoint resolution, got %q", got)
	}
}

func TestAPIAfterMigratesAllLegacySystemRoutes(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := rainbondv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("add Rainbond scheme: %v", err)
	}
	if err := v2.AddToScheme(scheme); err != nil {
		t.Fatalf("add APISIX scheme: %v", err)
	}

	component := &rainbondv1alpha1.RbdComponent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      APIName,
			Namespace: "rbd-system",
			UID:       types.UID("api-component-uid"),
		},
	}
	objects := []runtime.Object{component}
	routes := []struct {
		name string
		port int
	}{
		{name: "rbd-api-api", port: 8443},
		{name: "rbd-api-healthz", port: 8889},
		{name: "rbd-api-websocket", port: 6060},
	}
	for _, current := range routes {
		route := rbdRouteTemplateForTCP(current.name, current.port, "service")
		if err := controllerutil.SetControllerReference(component, route, scheme); err != nil {
			t.Fatalf("set controller reference for %s: %v", current.name, err)
		}
		objects = append(objects, route)
	}

	k8sClient := fake.NewFakeClientWithScheme(scheme, objects...)
	handler := &api{ctx: context.Background(), client: k8sClient, component: component}
	if err := handler.After(); err != nil {
		t.Fatalf("migrate API routes: %v", err)
	}

	for _, current := range routes {
		route := &v2.ApisixRoute{}
		if err := k8sClient.Get(context.Background(), client.ObjectKey{Name: current.name, Namespace: component.Namespace}, route); err != nil {
			t.Fatalf("get route %s: %v", current.name, err)
		}
		if got := route.Spec.Stream[0].Backend.ResolveGranularity; got != "endpoint" {
			t.Fatalf("expected route %s to use endpoint resolution, got %q", current.name, got)
		}
	}
}

func assertRouteAndUpstream(t *testing.T, resources []client.Object, name string, port int32) {
	t.Helper()

	var route *v2.ApisixRoute
	var upstream *v2.ApisixUpstream
	routeIndex := -1
	upstreamIndex := -1
	for index, resource := range resources {
		if resource.GetName() != name {
			continue
		}
		switch typed := resource.(type) {
		case *v2.ApisixRoute:
			route = typed
			routeIndex = index
		case *v2.ApisixUpstream:
			upstream = typed
			upstreamIndex = index
		}
	}
	if route == nil {
		t.Fatalf("expected route %q", name)
	}
	if upstream == nil {
		t.Fatalf("expected upstream %q", name)
	}
	if upstreamIndex >= routeIndex {
		t.Fatalf("expected upstream %q to be created before its route", name)
	}
	if got := route.Spec.Stream[0].Backend.ResolveGranularity; got != "endpoint" {
		t.Fatalf("expected route %q to use endpoint resolution, got %q", name, got)
	}
	if upstream.Spec == nil || upstream.Spec.HealthCheck == nil || upstream.Spec.HealthCheck.Active == nil {
		t.Fatalf("expected route %q to have an active upstream health check", name)
	}
	if got := upstream.Spec.HealthCheck.Active.Port; got != port {
		t.Fatalf("expected route %q health check port %d, got %d", name, port, got)
	}
}

func legacyRouteClient(t *testing.T, customized, owned bool) (*rainbondv1alpha1.RbdComponent, *v2.ApisixRoute, client.Client) {
	t.Helper()

	scheme := runtime.NewScheme()
	if err := rainbondv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("add Rainbond scheme: %v", err)
	}
	if err := v2.AddToScheme(scheme); err != nil {
		t.Fatalf("add APISIX scheme: %v", err)
	}

	component := &rainbondv1alpha1.RbdComponent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      AppUIName,
			Namespace: "rbd-system",
			UID:       types.UID("component-uid"),
		},
	}
	route := rbdDefaultRouteTemplateForTCP("rbd-app-ui", 7070).(*v2.ApisixRoute)
	route.Spec.Stream[0].Backend.ResolveGranularity = "service"
	route.Annotations = nil
	if customized {
		route.Spec.Stream[0].Match.Host = "custom.example.com"
	}
	if owned {
		if err := controllerutil.SetControllerReference(component, route, scheme); err != nil {
			t.Fatalf("set controller reference: %v", err)
		}
	}

	k8sClient := fake.NewFakeClientWithScheme(scheme, component, route)
	return component, route, k8sClient
}
