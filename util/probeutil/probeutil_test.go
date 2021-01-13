package probeutil

import (
	"reflect"
	"testing"

	"k8s.io/apimachinery/pkg/util/intstr"

	corev1 "k8s.io/api/core/v1"
)

func TestMakeReadinessProbeTCP(t *testing.T) {
	type args struct {
		host string
		port int
	}
	tests := []struct {
		name string
		args args
		want *corev1.Probe
	}{
		{
			name: "success",
			args: args{
				"",
				123,
			},
			want: &corev1.Probe{
				Handler: corev1.Handler{
					TCPSocket: &corev1.TCPSocketAction{
						Port: intstr.FromInt(123),
						Host: "",
					},
				},
				InitialDelaySeconds: 5,
				TimeoutSeconds:      5,
				PeriodSeconds:       5,
				SuccessThreshold:    1,
				FailureThreshold:    3,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MakeReadinessProbeTCP(tt.args.host, tt.args.port); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MakeReadinessProbeTCP() = %v, want %v", got, tt.want)
			}
		})
	}
}
