package v1alpha1

// FirstGatewayEIP returns the first external ip of the nodesforgateway list.
func (r *RainbondCluster) FirstGatewayEIP() string {
	if len(r.Spec.NodesForGateway) == 0 {
		return ""
	}
	node := r.Spec.NodesForGateway[0]
	return node.InternalIP
}
