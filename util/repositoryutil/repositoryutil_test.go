package repositoryutil

import "testing"

func TestLoginRepository(t *testing.T) {
	testcase := []struct {
		serverAddress string
		username      string
		password      string
		wantErr       bool
	}{
		{
			serverAddress: "registry.cn-hangzhou.aliyuncs.com",
			username:      "testuser",
			password:      "123456",
			wantErr:       true,
		},
		{
			serverAddress: "goodrain.me",
			username:      "admin",
			password:      "123456",
			wantErr:       true,
		},
		{
			serverAddress: "docker.io",
			username:      "testuser",
			password:      "123456",
			wantErr:       true,
		},
	}
	for _, tc := range testcase {
		err := LoginRepository(tc.serverAddress, tc.username, tc.password)
		if err != nil && !tc.wantErr {
			t.Fatalf("LoginRepository() error = %v, wantErr %v", err, tc.wantErr)
		}
	}
}
