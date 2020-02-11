package controller

import (
	"fmt"
	"net"

	"github.com/goodrain/rainbond-operator/pkg/openapi/model"
)

func checkConfig(req *model.GlobalConfigs, clusterInfo *model.ClusterInfo) error {
	if len(req.GatewayNodes) == 0 {
		return fmt.Errorf(model.IllegalConfigDataFormat, "pelase select gateway nodes")
	}

	availNode := make(map[string]struct{})
	for _, node := range clusterInfo.NodeAvailPorts {
		availNode[node.NodeIP] = struct{}{}
	}

	for _, node := range req.GatewayNodes {
		if _, ok := availNode[node.NodeIP]; !ok {
			return fmt.Errorf(model.IllegalConfigDataFormat, fmt.Sprintf("gateway node not support:%s", node.NodeIP))
		}
	}

	for _, ingress := range req.GatewayIngressIPs {
		if addr := net.ParseIP(ingress); addr == nil {
			return fmt.Errorf(model.IllegalConfigDataFormat, fmt.Sprintf("gateway ingress illegal :%s", ingress))
		}
	}
	if len(clusterInfo.Storage) > 0 {
		availStorage := make(map[string]struct{})
		for _, storage := range clusterInfo.Storage {
			availStorage[storage.Name] = struct{}{}
		}

		if _, ok := availStorage[req.Storage.Name]; !ok {
			return fmt.Errorf(model.IllegalConfigDataFormat, fmt.Sprintf("storage class not support :%s", req.Storage.Name))
		}
	}
	return nil
}
