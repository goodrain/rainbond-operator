package usecase

import (
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/GLYASAI/rainbond-operator/pkg/openapi/model"
	"github.com/GLYASAI/rainbond-operator/pkg/openapi/user/mock"
)

func TestUserUsecase_Login(t *testing.T) {
	tests := []struct {
		name, username, password string
		want, ret                *model.User
		wantErr, repoErr         error
	}{
		{
			name:     "user not found",
			username: "foobar",
			wantErr:  UserNotFound,
			repoErr:  UserNotFound,
		},
		{
			name:     "wrong password",
			username: "admin",
			password: "wrongpassword",
			ret: &model.User{
				Username: "admin",
				Password: "admin",
			},
			wantErr: WrongPassword,
		},
		{
			name:     "ok",
			username: "admin",
			password: "admin",
			ret: &model.User{
				Username: "admin",
				Password: "admin",
			},
		},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	for idx := range tests {
		tc := tests[idx]
		t.Run(tc.name, func(t *testing.T) {
			m := mock.NewMockRepository(ctrl)
			m.EXPECT().GetByUsername(tc.username).Return(tc.ret, tc.repoErr)

			ucase := NewUserUsecase(m)
			_, err := ucase.Login(tc.username, tc.password)
			if err != tc.wantErr {
				t.Errorf("want error %v, but got %v", tc.wantErr, err)
				return
			}
			if tc.wantErr != nil {
				return
			}
		})
	}
}
