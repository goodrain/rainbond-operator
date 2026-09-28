package k8sutil

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	v2 "github.com/goodrain/rainbond-operator/api/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func testGlobalRule() *v2.ApisixGlobalRule {
	return &v2.ApisixGlobalRule{ObjectMeta: metav1.ObjectMeta{Name: "isolated-monitor", Namespace: "test"}, TypeMeta: metav1.TypeMeta{Kind: "ApisixGlobalRule", APIVersion: "apisix.apache.org/v2"}, Spec: v2.ApisixGlobalRuleSpec{Plugins: []v2.ApisixRoutePlugin{{Name: "prometheus", Enable: true, Config: v2.ApisixRoutePluginConfig{"prefer_name": true}}}}}
}

func TestGlobalRuleUsesIsolatedClient(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := v2.Install(scheme); err != nil {
		t.Fatal(err)
	}
	k8s := fake.NewClientBuilder().WithScheme(scheme).Build()
	expected := testGlobalRule()
	if err := k8s.Create(context.Background(), expected); err != nil {
		t.Fatal(err)
	}
	var actual v2.ApisixGlobalRule
	if err := k8s.Get(context.Background(), client.ObjectKeyFromObject(expected), &actual); err != nil {
		t.Fatal(err)
	}
	if len(actual.Spec.Plugins) != 1 || actual.Spec.Plugins[0].Name != "prometheus" {
		t.Fatal("missing isolated test resource")
	}
}

// The former ad-hoc test used an empty scheme. Reproduce its rejection against
// a loopback server, proving that client.Create stops before sending a request.
func TestUnregisteredGlobalRuleNeverSendsRequest(t *testing.T) {
	var requests int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { atomic.AddInt32(&requests, 1); w.WriteHeader(500) }))
	defer server.Close()
	cfg := &rest.Config{Host: server.URL}
	mapper, err := apiutil.NewDynamicRESTMapper(cfg, apiutil.WithLazyDiscovery)
	if err != nil {
		t.Fatal(err)
	}
	k8s, err := client.New(cfg, client.Options{Scheme: runtime.NewScheme(), Mapper: mapper})
	if err != nil {
		t.Fatal(err)
	}
	if err := k8s.Create(context.Background(), testGlobalRule()); err == nil {
		t.Fatal("accepted an unregistered type")
	}
	if atomic.LoadInt32(&requests) != 0 {
		t.Fatal("unregistered resource reached transport")
	}
}
