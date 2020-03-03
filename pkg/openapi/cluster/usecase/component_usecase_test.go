package usecase

import (
	"testing"

	"github.com/goodrain/rainbond-operator/cmd/openapi/option"
	corev1 "k8s.io/api/core/v1"
)

func Test_componentUsecase_convertEventMessage(t *testing.T) {
	type fields struct {
		cfg *option.Config
	}
	type args struct {
		events []corev1.Event
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		{
			name: "test1",
			args: args{
				events: []corev1.Event{
					{
						Type:    corev1.EventTypeWarning,
						Message: "message 1",
					},
					{
						Type:    corev1.EventTypeWarning,
						Message: "message 2",
					},
					{
						Type:    corev1.EventTypeNormal,
						Message: "message 3",
					},
				},
			},
			want: "message 1\nmessage 2\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cc := &componentUsecase{
				cfg: tt.fields.cfg,
			}
			if got := cc.convertEventMessage(tt.args.events); got != tt.want {
				t.Errorf("componentUsecase.convertEventMessage() = %v, want %v", got, tt.want)
			}
		})
	}
}
