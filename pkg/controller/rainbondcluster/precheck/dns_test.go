package precheck

import "testing"

func TestPing(t *testing.T) {
	tests := []struct {
		target, expect string
	}{
		{
			target: "www.rainbond.com",
			expect: "true\n",
		},
		{
			target: "www.foobar12345678900987654321.com",
			expect: "false\n",
		},
		{
			target: "12345678900",
			expect: "false\n",
		},
		{
			target: "registry.cn-hangzhou.aliyuncs.com",
			expect: "true\n",
		},
	}

	for i := range tests {
		tc := tests[i]

		out, err := ping(tc.target)
		if err != nil {
			t.Error(err)
			t.FailNow()
		}
		if out != tc.expect {
			t.Errorf("expect %s, but got %s", tc.expect, out)
		}
	}
}
