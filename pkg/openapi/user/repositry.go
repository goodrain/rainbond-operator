package user

import "github.com/goodrain/rainbond-operator/pkg/openapi/model"

// Repository represent the user's repository contract
type Repository interface {
	CreateIfNotExist(user *model.User) error
	GetByUsername(username string) (*model.User, error)
}
