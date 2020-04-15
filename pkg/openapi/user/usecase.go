package user

import "github.com/goodrain/rainbond-operator/pkg/openapi/model"

// Usecase represent the user's usecases
type Usecase interface {
	Login(username, password string) (string, error)
	GenerateUser() (*model.User, error)
	IsGenerated() (bool, error)
	UpdateAdminPassword(pass string) error
}
