package rainbondcluster

import (
	"net"
	"testing"
)

type occupyPortFunc func() (net.Listener, error)

func TestCheckPortOccupation(t *testing.T) {
	tests := []struct {
		name, address  string
		want           bool
		occupyPortFunc occupyPortFunc
	}{
		{
			name:    "The port is already occupied",
			address: ":38080",
			occupyPortFunc: func() (net.Listener, error) {
				return net.Listen("tcp", ":38080")
			},
			want: true,
		},
		{
			name:    "Okay",
			address: ":38080",
			occupyPortFunc: func() (net.Listener, error) {
				return nil, nil
			},
			want: false,
		},
	}

	for idx := range tests {
		tc := tests[idx]
		t.Run(tc.name, func(t *testing.T) {
			l, err := tc.occupyPortFunc()
			if err != nil {
				t.Errorf("failed exec function 'occupyPortFunc': %v", err)
				return
			}
			defer func() {
				if l != nil {
					l.Close()
				}
			}()

			got := isPortOccupied(tc.address)
			if tc.want != got {
				t.Errorf("Expected %v, but got %v", tc.want, got)
			}
		})
	}
}
