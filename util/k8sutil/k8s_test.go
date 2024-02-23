package k8sutil

import (
	"context"
	v2 "github.com/goodrain/rainbond-operator/api/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"log"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"testing"
)

const APIVersion = "apisix.apache.org/v2"
const ApisixTls = "ApisixTls"

var RainbondKubeClient client.Client

func TestName(t *testing.T) {
	scheme := runtime.NewScheme()
	config := ctrl.GetConfigOrDie()

	mapper, err := apiutil.NewDynamicRESTMapper(config, apiutil.WithLazyDiscovery)
	if err != nil {
		log.Println(err)
	}

	runtimeClient, err := client.New(config, client.Options{Scheme: scheme, Mapper: mapper})
	if err != nil {
		log.Println(err)
	}
	globalRule := &v2.ApisixGlobalRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "apisix-monitor",
			Namespace: "rbd-system",
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "ApisixGlobalRule",
			APIVersion: APIVersion,
		},
		Spec: v2.ApisixGlobalRuleSpec{
			Plugins: []v2.ApisixRoutePlugin{
				{
					Name:   "prometheus",
					Enable: true,
					Config: v2.ApisixRoutePluginConfig{
						"prefer_name": "true",
					},
				},
			},
		},
	}

	err = runtimeClient.Create(context.Background(), globalRule)
	if err != nil {
		log.Println(err)
	}

}
