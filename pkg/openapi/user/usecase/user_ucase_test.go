package usecase

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/goodrain/rainbond-operator/pkg/openapi/model"
	"github.com/goodrain/rainbond-operator/pkg/openapi/user"
	"github.com/goodrain/rainbond-operator/pkg/openapi/user/mock"
	"github.com/jinzhu/gorm"
	"github.com/sethvargo/go-password/password"
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

			ucase := NewUserUsecase(m, "foobar")
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

func Test_userUsecase_GenerateUser(t *testing.T) {
	type fields struct {
		secretKey string
		userRepo  user.Repository
	}
	tests := []struct {
		name     string
		fields   fields
		want     *model.User
		mockFunc func(mockRepo mock.MockRepository)
		wantErr  bool
	}{
		{
			name:   "success",
			fields: fields{secretKey: ""},
			want:   &model.User{Username: "", Password: ""},
			mockFunc: func(mockRepo mock.MockRepository) {
				mockRepo.EXPECT().Listusers().Return(nil, gorm.ErrRecordNotFound)
				mockRepo.EXPECT().CreateIfNotExist(gomock.Any()).Return(nil)
			},
			wantErr: false,
		},
		{
			name:   "user already exists",
			fields: fields{secretKey: ""},
			want:   &model.User{Username: "", Password: ""},
			mockFunc: func(mockRepo mock.MockRepository) {
				mockRepo.EXPECT().Listusers().Return([]*model.User{{Username: ""}}, nil)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			m := mock.NewMockRepository(ctrl)
			tt.fields.userRepo = m
			u := userUsecase{
				secretKey: tt.fields.secretKey,
				userRepo:  m,
			}

			tt.mockFunc(*m)

			_, err := u.GenerateUser()
			if (err != nil) != tt.wantErr {
				t.Errorf("userUsecase.GenerateUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestGeneratePassword(t *testing.T) {
	fmt.Println(password.Generate(8, 8, 0, false, false))
}
