package handler

import (
	"context"
	"fmt"
	"strings"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
	v2 "github.com/goodrain/rainbond-operator/api/v2"
	"github.com/goodrain/rainbond-operator/util/commonutil"
	"github.com/goodrain/rainbond-operator/util/constants"
	"github.com/goodrain/rainbond-operator/util/k8sutil"
	"github.com/goodrain/rainbond-operator/util/rbdutil"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ApiGatewayName name for rbd-gateway.
var ApiGatewayName = "rbd-gateway"

type apigateway struct {
	ctx       context.Context
	client    client.Client
	cluster   *rainbondv1alpha1.RainbondCluster
	component *rainbondv1alpha1.RbdComponent
	labels    map[string]string
}

// NewApiGateway returns a new rbd-gateway handler.
func NewApiGateway(ctx context.Context, client client.Client, component *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster) ComponentHandler {
	return &apigateway{
		ctx:       ctx,
		client:    client,
		cluster:   cluster,
		component: component,
		labels:    LabelsForRainbondComponent(component),
	}
}

// rbdDefaultRouteForHTTP 代理转发控制台的所有路由
func rbdDefaultRouteForHTTP() client.Object {
	return &v2.ApisixRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rbd-app-ui",
			Namespace: rbdutil.GetenvDefault("RBD_NAMESPACE", constants.Namespace),
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       constants.ApisixRoute,
			APIVersion: constants.APISixAPIVersion,
		},
		Spec: v2.ApisixRouteSpec{
			HTTP: []v2.ApisixRouteHTTP{
				{
					Name:     "proxy",
					Priority: 2,
					Backends: []v2.ApisixRouteHTTPBackend{
						{
							ServiceName: "rbd-app-ui-proxy",
							ServicePort: intstr.FromInt(6060),
						},
					},
					Match: v2.ApisixRouteHTTPMatch{
						Paths: []string{
							"/proxy/*",
						},
						NginxVars: []v2.ApisixRouteHTTPMatchExpr{
							{
								Subject: v2.ApisixRouteHTTPMatchExprSubject{
									Scope: "Variable",
									Name:  "server_port",
								},
								Op:  "In",
								Set: []string{"7070", "7071"},
							},
						},
					},
					Websocket: true,
					Authentication: v2.ApisixRouteAuthentication{
						Enable: false,
						Type:   "basicAuth",
					},
				},
				{
					Name:     "http",
					Priority: 1,
					Backends: []v2.ApisixRouteHTTPBackend{
						{
							ServiceName: "rbd-app-ui",
							ServicePort: intstr.FromInt(7070),
						},
					},
					Match: v2.ApisixRouteHTTPMatch{
						Paths: []string{
							"/*",
						},
						NginxVars: []v2.ApisixRouteHTTPMatchExpr{
							{
								Subject: v2.ApisixRouteHTTPMatchExprSubject{
									Scope: "Variable",
									Name:  "server_port",
								},
								Op:  "In",
								Set: []string{"7070", "7071"},
							},
						},
					},
					Websocket: false,
					Authentication: v2.ApisixRouteAuthentication{
						Enable: false,
						Type:   "basicAuth",
					},
				},
			},
		},
	}
}

func rbdDefaultRouteTemplateForTCP(name string, port int) client.Object {
	return &v2.ApisixRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: rbdutil.GetenvDefault("RBD_NAMESPACE", constants.Namespace),
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       constants.ApisixRoute,
			APIVersion: constants.APISixAPIVersion,
		},
		Spec: v2.ApisixRouteSpec{
			Stream: []v2.ApisixRouteStream{
				{
					Name:     name,
					Protocol: string(corev1.ProtocolTCP),
					Match: v2.ApisixRouteStreamMatch{
						IngressPort: int32(port),
					},
					Backend: v2.ApisixRouteStreamBackend{
						ServiceName: name,
						ServicePort: intstr.FromInt(port),
					},
				},
			},
		},
	}
}

// Before -
func (a *apigateway) Before() error {
	k8sNodes, err := k8sutil.ListNodes(context.Background(), a.client)
	if err != nil {
		logrus.Error("get cluster node list error:", err)
	}
	k8sNodeNames := make(map[string]struct{})
	nodeLabels := make(map[string]struct{})
	for _, k8sNode := range k8sNodes {
		if hostName, ok := k8sNode.Labels["kubernetes.io/hostname"]; ok {
			nodeLabels[hostName] = struct{}{}
		}
		if hostName, ok := k8sNode.Labels["k3s.io/hostname"]; ok {
			nodeLabels[hostName] = struct{}{}
		}
		k8sNodeNames[k8sNode.Name] = struct{}{}
	}
	nodeForGateway := a.cluster.Spec.NodesForGateway
	if nodeForGateway != nil {
		for _, currentNode := range nodeForGateway {
			if _, ok := k8sNodeNames[currentNode.Name]; !ok {
				fmt.Printf("\033[1;31;40m%s\033[0m\n", fmt.Sprintf("Node %v cannot be found in the cluster", currentNode.Name))
			}
			if _, ok := nodeLabels[currentNode.Name]; !ok {
				fmt.Printf("\033[1;31;40m%s\033[0m\n", fmt.Sprintf("Node name %v is not bound to the label of a cluster node", currentNode.Name))
			}
		}
	}
	return nil
}

// Resources -
func (a *apigateway) Resources() []client.Object {
	return []client.Object{
		a.configmap(),
		a.deploy(),
		a.monitorGlobalRule(),
		a.monitorService(),
	}
}

// After -
func (a *apigateway) After() error {
	return nil
}

// ListPods -
func (a *apigateway) ListPods() ([]corev1.Pod, error) {
	return listPods(a.ctx, a.client, a.component.Namespace, a.labels)
}

// deploy -
func (a *apigateway) deploy() client.Object {
	var nodeNames []string
	for _, n := range a.cluster.Spec.NodesForGateway {
		nodeNames = append(nodeNames, n.Name)

		//转换小写如果不一致，则增加一份小写，兼容rke2，rke2安装的k8s的hostname被转小写了
		if strings.ToLower(n.Name) != n.Name {
			nodeNames = append(nodeNames, strings.ToLower(n.Name))
		}
	}
	var affinity *corev1.Affinity
	if len(nodeNames) > 0 {
		affinity = affinityForRequiredNodes(nodeNames)
	}
	if affinity == nil {
		return nil
	}
	envs := append(a.component.Spec.Env, []corev1.EnvVar{
		{
			Name:  "RBD_NAMESPACE",
			Value: a.component.Namespace,
		},
		{
			Name: "POD_NAMESPACE",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					APIVersion: "v1",
					FieldPath:  "metadata.namespace",
				},
			},
		},
		{
			Name: "POD_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					APIVersion: "v1",
					FieldPath:  "metadata.name",
				},
			},
		},
	}...)

	vms := append(a.component.Spec.VolumeMounts, []corev1.VolumeMount{
		{
			Name:      "apisix-config-yaml-configmap",
			MountPath: "/usr/local/apisix/conf/config.yaml",
			SubPath:   "config.yaml",
		},
	}...)

	vs := append(a.component.Spec.Volumes, []corev1.Volume{
		{
			Name: "apisix-config-yaml-configmap",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					DefaultMode: commonutil.Int32(420),
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "apisix-gw-config.yaml",
					},
				},
			},
		},
	}...)

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ApiGatewayName,
			Namespace: rbdutil.GetenvDefault("RBD_NAMESPACE", constants.Namespace),
			Labels:    a.labels,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: a.labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:      ApiGatewayName,
					Namespace: rbdutil.GetenvDefault("RBD_NAMESPACE", constants.Namespace),
					Labels:    a.labels,
				},
				Spec: corev1.PodSpec{
					Affinity:                      affinity,
					TerminationGracePeriodSeconds: commonutil.Int64(0),
					ServiceAccountName:            rbdutil.GetenvDefault("SERVICE_ACCOUNT_NAME", "rainbond-operator"),
					HostNetwork:                   true,
					DNSPolicy:                     corev1.DNSClusterFirst,
					RestartPolicy:                 corev1.RestartPolicyAlways,
					SchedulerName:                 "default-scheduler",
					Tolerations: []corev1.Toleration{
						{
							Operator: corev1.TolerationOpExists,
						},
					},
					SecurityContext: &corev1.PodSecurityContext{
						RunAsUser: commonutil.Int64(0),
					},
					Containers: []corev1.Container{
						{
							Name:            "ingress-apisix",
							Image:           "apache/apisix-ingress-controller:1.8.0",
							ImagePullPolicy: a.component.ImagePullPolicy(),
							SecurityContext: &corev1.SecurityContext{
								Privileged: commonutil.Bool(true),
							},
							Command: []string{
								"/ingress-apisix/apisix-ingress-controller",
								"ingress",
								"--log-output",
								"stdout",
								"--apisix-resource-sync-interval",
								"1h",
								"--apisix-resource-sync-comparison=true",
								"--http-listen",
								":7080",
								"--https-listen",
								":7443",
								"--default-apisix-cluster-name",
								"default",
								"--default-apisix-cluster-base-url",
								"http://127.0.0.1:9180/apisix/admin",
								"--default-apisix-cluster-admin-key",
								"edd1c9f034335f136f87ad84b625c8f1",
								"--api-version",
								"apisix.apache.org/v2",
								"--ingress-status-address",
								"",
								"--disable-status-updates=false",
								"--etcd-server-enabled=true",
							},
							Env:                      envs,
							TerminationMessagePath:   "/dev/termination-log",
							TerminationMessagePolicy: corev1.TerminationMessageReadFile,
						},
						{
							Name:            "apisix",
							Image:           "apache/apisix:3.8.0-debian",
							ImagePullPolicy: a.component.ImagePullPolicy(),
							SecurityContext: &corev1.SecurityContext{
								RunAsUser:  commonutil.Int64(0),
								Privileged: commonutil.Bool(true),
							},
							VolumeMounts:             vms,
							TerminationMessagePath:   "/dev/termination-log",
							TerminationMessagePolicy: corev1.TerminationMessageReadFile,
						},
					},
					Volumes: vs,
				},
			},
		},
	}
}

// monitorService 这里地址不能改变，因为rbd-monitor 会读取这个service
func (a *apigateway) monitorService() client.Object {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "apisix-monitor",
			Namespace: rbdutil.GetenvDefault("RBD_NAMESPACE", constants.Namespace),
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Protocol:   corev1.ProtocolTCP,
					Port:       9091,
					TargetPort: intstr.FromInt(9091),
				},
			},
			Selector: a.labels,
			Type:     corev1.ServiceTypeClusterIP,
		},
	}
}

// monitorGlobalRule 全局监控普罗米修斯，自动让所有的路由都生效
func (a *apigateway) monitorGlobalRule() client.Object {
	return &v2.ApisixGlobalRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "monitor",
			Namespace: rbdutil.GetenvDefault("RBD_NAMESPACE", constants.Namespace),
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       constants.ApisixGlobalRule,
			APIVersion: constants.APISixAPIVersion,
		},
		Spec: v2.ApisixGlobalRuleSpec{
			Plugins: []v2.ApisixRoutePlugin{
				{
					Name:   "prometheus",
					Enable: true,
					Config: v2.ApisixRoutePluginConfig{
						"prefer_name": true,
					},
				},
			},
		},
	}
}

// configmap 配置文件
func (a *apigateway) configmap() client.Object {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "apisix-gw-config.yaml",
			Namespace: rbdutil.GetenvDefault("RBD_NAMESPACE", constants.Namespace),
		},
		Data: map[string]string{
			"config.yaml": `plugin_attr:
  prometheus:
    export_addr:
      ip: 0.0.0.0
      port: 9091

deployment:
  admin:
    allow_admin:
      - 127.0.0.0/24
      - 0.0.0.0/0
    admin_listen:
      ip: 0.0.0.0
      port: 9180
  etcd:
    host:
      - "http://127.0.0.1:12379"
    prefix: "/apisix"
    timeout: 60

apisix:
  proxy_mode: "http&stream"
  ssl:
    enable: true
    listen:
      - port: 443
      - port: 7443
  enable_control: true
  enable_reuseport: true
  node_listen:
    - 80
    - 7070
    - 7071
  stream_proxy:
    tcp:
      - addr: 8443
      - addr: 8889
      - addr: 6060

plugins: # plugin list (sorted by priority)
  - coraza-filter                  # priority: 7999
  - real-ip                        # priority: 23000
  - ai                             # priority: 22900
  - client-control                 # priority: 22000
  - proxy-control                  # priority: 21990
  - request-id                     # priority: 12015
  - zipkin                         # priority: 12011
  #- skywalking                    # priority: 12010
  #- opentelemetry                 # priority: 12009
  - ext-plugin-pre-req             # priority: 12000
  - fault-injection                # priority: 11000
  - mocking                        # priority: 10900
  - serverless-pre-function        # priority: 10000
  #- batch-requests                # priority: 4010
  - cors                           # priority: 4000
  - ip-restriction                 # priority: 3000
  - ua-restriction                 # priority: 2999
  - referer-restriction            # priority: 2990
  - csrf                           # priority: 2980
  - uri-blocker                    # priority: 2900
  - request-validation             # priority: 2800
  - openid-connect                 # priority: 2599
  - cas-auth                       # priority: 2597
  - authz-casbin                   # priority: 2560
  - authz-casdoor                  # priority: 2559
  - wolf-rbac                      # priority: 2555
  - ldap-auth                      # priority: 2540
  - hmac-auth                      # priority: 2530
  - basic-auth                     # priority: 2520
  - jwt-auth                       # priority: 2510
  - key-auth                       # priority: 2500
  - consumer-restriction           # priority: 2400
  - forward-auth                   # priority: 2002
  - opa                            # priority: 2001
  - authz-keycloak                 # priority: 2000
  #- error-log-logger              # priority: 1091
  - proxy-mirror                   # priority: 1010
  - proxy-cache                    # priority: 1009
  - proxy-rewrite                  # priority: 1008
  - workflow                       # priority: 1006
  - api-breaker                    # priority: 1005
  - limit-conn                     # priority: 1003
  - limit-count                    # priority: 1002
  - limit-req                      # priority: 1001
  #- node-status                   # priority: 1000
  - gzip                           # priority: 995
  - traffic-split                  # priority: 966
  - redirect                       # priority: 900
  - response-rewrite               # priority: 899
  - kafka-proxy                    # priority: 508
  #- dubbo-proxy                   # priority: 507
  - grpc-transcode                 # priority: 506
  - grpc-web                       # priority: 505
  - public-api                     # priority: 501
  - prometheus                     # priority: 500
  - datadog                        # priority: 495
  - elasticsearch-logger           # priority: 413
  - echo                           # priority: 412
  - loggly                         # priority: 411
  - http-logger                    # priority: 410
  - splunk-hec-logging             # priority: 409
  - skywalking-logger              # priority: 408
  - google-cloud-logging           # priority: 407
  - sls-logger                     # priority: 406
  - tcp-logger                     # priority: 405
  - kafka-logger                   # priority: 403
  - rocketmq-logger                # priority: 402
  - syslog                         # priority: 401
  - udp-logger                     # priority: 400
  - file-logger                    # priority: 399
  - clickhouse-logger              # priority: 398
  - tencent-cloud-cls              # priority: 397
  - inspect                        # priority: 200
  #- log-rotate                    # priority: 100
  # <- recommend to use priority (0, 100) for your custom plugins
  - example-plugin                 # priority: 0
  #- gm                            # priority: -43
  - aws-lambda                     # priority: -1899
  - azure-functions                # priority: -1900
  - openwhisk                      # priority: -1901
  - openfunction                   # priority: -1902
  - serverless-post-function       # priority: -2000
  - ext-plugin-post-req            # priority: -3000
  - ext-plugin-post-resp           # priority: -4000
        `,
		},
	}

}
