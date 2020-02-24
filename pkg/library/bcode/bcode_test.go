package bcode

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErr2Coder(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{
			name: "ok",
			err:  newCode(10001),
			want: 10001,
		},
		{
			name: "200",
			err:  nil,
			want: 200,
		},
		{
			name: "not recognized",
			err:  fmt.Errorf("not recognized error"),
			want: 500,
		},
	}

	for idx := range tests {
		tc := tests[idx]
		t.Run(tc.name, func(t *testing.T) {
			coder := Err2Coder(tc.err)
			assert.Equal(t, tc.want, coder.Code())
		})
	}
}

func TestStr2Coder(t *testing.T) {
	tests := []struct {
		name string
		str  string
		want int
	}{
		{
			name: "ok",
			str:  "10001",
			want: 10001,
		},
		{
			name: "200",
			str:  "",
			want: 200,
		},
		{
			name: "not int",
			str:  "not int",
			want: 500,
		},
	}

	for idx := range tests {
		tc := tests[idx]
		t.Run(tc.name, func(t *testing.T) {
			coder := Str2Coder(tc.str)
			assert.Equal(t, tc.want, coder.Code())
		})
	}
}
