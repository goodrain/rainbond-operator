package rbdcomponent

import (
	"github.com/goodrain/rainbond-operator/pkg/controller/rbdcomponent/handler"
	"github.com/goodrain/rainbond-operator/pkg/util/constants"
)

func init() {
	AddHandlerFunc(handler.EtcdName, handler.NewETCD)
	AddHandlerFunc(handler.GatewayName, handler.NewGateway)
	AddHandlerFunc(handler.HubName, handler.NewHub)
	AddHandlerFunc(handler.APIName, handler.NewAPI)
	AddHandlerFunc(handler.AppUIName, handler.NewAppUI)
	AddHandlerFunc(handler.ChaosName, handler.NewChaos)
	AddHandlerFunc(handler.EventLogName, handler.NewEventLog)
	AddHandlerFunc(handler.MonitorName, handler.NewMonitor)
	AddHandlerFunc(handler.WorkerName, handler.NewWorker)
	AddHandlerFunc(handler.MQName, handler.NewMQ)
	AddHandlerFunc(handler.ResourceProxyName, handler.NewResourceProxy)
	AddHandlerFunc(handler.NodeName, handler.NewNode)
	AddHandlerFunc(handler.DBName, handler.NewDB)
	AddHandlerFunc(handler.WebCliName, handler.NewWebCli)
	AddHandlerFunc(handler.MetricsServerName, handler.NewMetricsServer)
	AddHandlerFunc(handler.NFSName, handler.NewNFS)
	AddHandlerFunc(handler.KubernetesDashboardName, handler.NewK8sDashboard)
	AddHandlerFunc(handler.DashboardMetricsScraperName, handler.NewDashboardMetricsScraper)
	AddHandlerFunc(handler.NFSName, handler.NewNFS)
	AddHandlerFunc(constants.AliyunCSINasPlugin, handler.NewAliyunCSINasPlugin)
	AddHandlerFunc(constants.AliyunCSINasProvisioner, handler.NewAliyunCSINasProvisioner)
	AddHandlerFunc(constants.AliyunCSIDiskPlugin, handler.NewAliyunCSIDiskPlugin)
	AddHandlerFunc(constants.AliyunCSIDiskProvisioner, handler.NewaliyunCSIDiskProvisioner)
}
