package rbdcomponent

import (
	"github.com/GLYASAI/rainbond-operator/pkg/controller/rbdcomponent/handler"
)

func init() {
	AddHandlerFunc(handler.GatewayName, handler.NewGateway)
	AddHandlerFunc(handler.HubName, handler.NewHub)
	AddHandlerFunc(handler.NFSName, handler.NewNFSProvisioner)
}
