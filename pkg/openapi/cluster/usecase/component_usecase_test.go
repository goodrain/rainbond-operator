package usecase

import (
	"testing"
	"time"

	"github.com/goodrain/rainbond-operator/cmd/openapi/option"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_componentUsecase_convertEventMessage(t *testing.T) {
	type fields struct {
		cfg *option.Config
	}
	type args struct {
		events []corev1.Event
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		reason  string
		message string
	}{
		{
			name: "test1",
			args: args{
				events: []corev1.Event{
					{
						Type:          corev1.EventTypeWarning,
						Reason:        "filed shedule",
						Message:       "message 1",
						LastTimestamp: metav1.Time{Time: time.Now().Add(1 * time.Second)},
					},
					{
						Type:          corev1.EventTypeWarning,
						Message:       "message 2",
						LastTimestamp: metav1.Time{Time: time.Now()},
					},
					{
						Type:    corev1.EventTypeNormal,
						Message: "message 3",
					},
				},
			},
			reason:  "filed shedule",
			message: "message 1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cc := &componentUsecase{
				cfg: tt.fields.cfg,
			}
			reason, message := cc.convertEventMessage(tt.args.events)
			if reason != tt.reason {
				t.Errorf("componentUsecase.convertEventMessage() got = %v, want %v", reason, tt.reason)
			}
			if message != tt.message {
				t.Errorf("componentUsecase.convertEventMessage() got1 = %v, want %v", message, tt.message)
			}
		})
	}
}
