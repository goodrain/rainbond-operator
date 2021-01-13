package probeutil

import (
	"net"

	"github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/util/intstr"
)

// MakeLivenessProbeHTTP -
func MakeLivenessProbeHTTP(host, path string, port int) *corev1.Probe {
	probe := MakeProbe(ProbeKindHTTP, host, path, port, corev1.URISchemeHTTP, nil)
	SetProbeArgs(probe, 10, 5, 10, 0, 0)
	return probe
}

// MakeLivenessProbeTCP -
func MakeLivenessProbeTCP(host string, port int) *corev1.Probe {
	probe := MakeProbe(ProbeKindTCP, host, "", port, corev1.URISchemeHTTP, nil)
	SetProbeArgs(probe, 10, 5, 10, 0, 0)
	return probe
}

// MakeReadinessProbeHTTP -
func MakeReadinessProbeHTTP(host, path string, port int) *corev1.Probe {
	probe := MakeProbe(ProbeKindHTTP, host, path, port, corev1.URISchemeHTTP, nil)
	SetProbeArgs(probe, 5, 5, 5, 0, 0)
	return probe
}

// MakeReadinessProbeTCP -
func MakeReadinessProbeTCP(host string, port int) *corev1.Probe {
	probe := MakeProbe(ProbeKindTCP, host, "", port, corev1.URISchemeHTTP, nil)
	SetProbeArgs(probe, 5, 5, 5, 0, 0)
	return probe
}

// MakeProbe -
func MakeProbe(kind ProbeKind, host, path string, port int, scheme corev1.URIScheme, headers []corev1.HTTPHeader) *corev1.Probe {
	handler := corev1.Handler{}
	switch kind {
	case ProbeKindHTTP:
		handler.HTTPGet = makeHTTPGetAction(host, path, port, scheme, headers)
	case ProbeKindTCP:
		handler.TCPSocket = makeTCPSocketAction(host, port)
	default:
		logrus.Warnf("do not support probe kind: %s", kind)
	}
	probe := &corev1.Probe{Handler: handler}
	return probe
}

// SetProbeArgs if arguments is illegal, reset to default value
func SetProbeArgs(probe *corev1.Probe, initialDelay, timeout, period, successThreshold, failureThreshold int32) {
	if probe == nil {
		return
	}
	if initialDelay < 0 {
		initialDelay = 0
	}
	if timeout <= 0 {
		timeout = 1
	}
	if period <= 0 {
		period = 10
	}
	if successThreshold <= 0 {
		successThreshold = 1
	}
	if failureThreshold <= 0 {
		failureThreshold = 3
	}
	probe.InitialDelaySeconds = initialDelay
	probe.TimeoutSeconds = timeout
	probe.PeriodSeconds = period
	probe.SuccessThreshold = successThreshold
	probe.FailureThreshold = failureThreshold
}

func makeHTTPGetAction(host, path string, port int, scheme corev1.URIScheme, headers []corev1.HTTPHeader) *corev1.HTTPGetAction {
	action := &corev1.HTTPGetAction{
		Port:        intstr.FromInt(port),
		Scheme:      scheme,
		HTTPHeaders: headers,
	}
	if host != "" {
		action.Host = host
	}
	if path != "" {
		action.Path = path
	}

	return action
}

func makeTCPSocketAction(host string, port int) *corev1.TCPSocketAction {
	action := &corev1.TCPSocketAction{
		Port: intstr.FromInt(port),
	}

	// host of tcp socket action can't be service name
	address := net.ParseIP(host)
	if address != nil {
		action.Host = host
	}

	return action
}

// ProbeKind -
type ProbeKind string

// ProbeKindHTTP -
var ProbeKindHTTP ProbeKind = "http"

// ProbeKindTCP -
var ProbeKindTCP ProbeKind = "tcp"
