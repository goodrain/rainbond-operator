package rainbondpackage

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"testing"

	"github.com/docker/docker/pkg/jsonmessage"

	"github.com/docker/docker/client"
)

func TestImageLoad(t *testing.T) {
	cli, _ := client.NewClientWithOpts(client.FromEnv)
	cli.NegotiateAPIVersion(context.TODO())

	file, err := os.Open("/tmp/rainbond/rainbond/api.tgz")
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	res, err := cli.ImageLoad(ctx, file, true)
	if err != nil {
		t.Errorf("path: %s; failed to load image: %v", "/tmp/rainbond-operator.tgz", err)
		return
	}
	if res.Body != nil {
		defer res.Body.Close()
		dec := json.NewDecoder(res.Body)
		for {
			var jm jsonmessage.JSONMessage
			if err := dec.Decode(&jm); err != nil {
				if err == io.EOF {
					break
				}
				t.Fatal(err)
				return
			}
			if jm.Error != nil {
				t.Fatal(jm.Error)
				return
			}
			t.Logf("%s\n", jm.Stream)
		}
	}
}

func TestParseImageName(t *testing.T) {
	tests := []struct {
		name string
		str  string
		want string
	}{
		{
			name: "foo",
			str:  "{\"stream\":\"Loaded image: rainbond/rbd-api:V5.2-dev\\n\"}",
			want: "rainbond/rbd-api:V5.2-dev",
		},
	}

	for idx := range tests {
		tc := tests[idx]
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseImageName(tc.str)
			if err != nil {
				t.Error(err)
				return
			}
			if tc.want != got {
				t.Errorf("want %s, but got %s", tc.want, got)
			}
		})
	}
}
