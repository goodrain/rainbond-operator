package precheck

import "testing"

func TestNslookup(t *testing.T) {
	tests := []struct {
		target  string
		wantErr bool
	}{
		{
			target:  "www.wutong.com",
			wantErr: false,
		},
		{
			target:  "www.foobar12345678900987654321.com",
			wantErr: true,
		},
		{
			target:  "12345678900",
			wantErr: true,
		},
		{
			target:  "swr.cn-southwest-2.myhuaweicloud.com",
			wantErr: false,
		},
	}

	for i := range tests {
		tc := tests[i]

		err := nslookup(tc.target)
		if err != nil && !tc.wantErr {
			t.Error(err)
			t.FailNow()
		}
	}
}
