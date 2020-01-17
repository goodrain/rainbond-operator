package rbdcomponent

import (
	"github.com/GLYASAI/rainbond-operator/pkg/controller/rbdcomponent/handler"
)

func init() {
	AddHandlerFunc(handler.EtcdName, handler.NewETCD)
	AddHandlerFunc(handler.GatewayName, handler.NewGateway)
	AddHandlerFunc(handler.HubName, handler.NewHub)
	AddHandlerFunc(handler.NFSName, handler.NewNFSProvisioner)
	AddHandlerFunc(handler.APIName, handler.NewAPI)
	AddHandlerFunc(handler.AppUIName, handler.NewAppUI)
	AddHandlerFunc(handler.ChaosName, handler.NewChaos)
	AddHandlerFunc(handler.DNSName, handler.NewDNS)
	AddHandlerFunc(handler.EventLogName, handler.NewEventLog)
	AddHandlerFunc(handler.MonitorName, handler.NewMonitor)
	AddHandlerFunc(handler.WorkerName, handler.NewWorker)
	AddHandlerFunc(handler.MQName, handler.NewMQ)
	AddHandlerFunc(handler.RepoName, handler.NewRepo)
	AddHandlerFunc(handler.NodeName, handler.NewNode)
	AddHandlerFunc(handler.DBName, handler.NewDB)
	AddHandlerFunc(handler.WebCliName, handler.NewWebCli)
}
