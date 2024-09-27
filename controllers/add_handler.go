package controllers

import (
	"context"
	"sort"
	"strings"

	rainbondv1alpha1 "github.com/goodrain/rainbond-operator/api/v1alpha1"
	"github.com/goodrain/rainbond-operator/controllers/handler"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func init() {
	AddHandlerFunc(handler.ApiGatewayName, handler.NewApiGateway)
	AddHandlerFunc(handler.HubName, handler.NewHub)
	AddHandlerFunc(handler.APIName, handler.NewAPI)
	AddHandlerFunc(handler.AppUIName, handler.NewAppUI)
	AddHandlerFunc(handler.ChaosName, handler.NewChaos)
	AddHandlerFunc(handler.EventLogName, handler.NewEventLog)
	AddHandlerFunc(handler.MonitorName, handler.NewMonitor)
	AddHandlerFunc(handler.WorkerName, handler.NewWorker)
	AddHandlerFunc(handler.MQName, handler.NewMQ)
	AddHandlerFunc(handler.DBName, handler.NewDB)
	AddHandlerFunc(handler.WebCliName, handler.NewWebCli)
	AddHandlerFunc(handler.LocalPathName, handler.NewLocalPath)
	AddHandlerFunc(handler.MinIOName, handler.NewMinIO)
}

type handlerFunc func(ctx context.Context, client client.Client, component *rainbondv1alpha1.RbdComponent, cluster *rainbondv1alpha1.RainbondCluster) handler.ComponentHandler

var handlerFuncs map[string]handlerFunc

// AddHandlerFunc adds handlerFunc to handlerFuncs.
func AddHandlerFunc(name string, fn handlerFunc) {
	if handlerFuncs == nil {
		handlerFuncs = map[string]handlerFunc{}
	}
	handlerFuncs[name] = fn
}

func supportedComponents() string {
	var supported []string
	for name := range handlerFuncs {
		supported = append(supported, name)
	}
	sort.Strings(supported)
	return strings.Join(supported, ",")
}
