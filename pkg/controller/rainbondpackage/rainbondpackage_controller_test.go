package rainbondpackage

import (
	"context"
	"testing"

	"github.com/docker/docker/client"
)

func TestCheckIfImageExists(t *testing.T) {
	tests := []struct {
		name, image string
		want        bool
		wantErr     error
	}{
		{
			name:    "image does not exists",
			image:   "foobar",
			want:    false,
			wantErr: nil,
		},
		{
			name:    "image exists",
			image:   "busyboxy",
			want:    false,
			wantErr: nil,
		},
	}

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.TODO()
	cli.NegotiateAPIVersion(ctx)
	p := pkg{
		dcli: cli,
		ctx:  ctx,
	}

	for i := range tests {
		tc := tests[i]
		t.Run(tc.name, func(t *testing.T) {
			exists, err := p.checkIfImageExists(tc.image)
			if err != nil && err != tc.wantErr {
				t.Errorf("unexpected error: %v", err)
				t.FailNow()
			}
			if exists != tc.want {
				t.Errorf("expected exists equals to %v, but got : %v", tc.want, exists)
			}
		})
	}
}
