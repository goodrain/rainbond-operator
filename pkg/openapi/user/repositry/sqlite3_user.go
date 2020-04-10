package repositry

import (
	"github.com/goodrain/rainbond-operator/pkg/openapi/model"
	"github.com/goodrain/rainbond-operator/pkg/openapi/user"
	"github.com/jinzhu/gorm"
)

type sqlite3UserRepo struct {
	db *gorm.DB
}

// NewSqlite3UserRepository will create an object that represent the user.Repository interface
func NewSqlite3UserRepository(db *gorm.DB) user.Repository {
	return &sqlite3UserRepo{db: db}
}

func (s *sqlite3UserRepo) CreateIfNotExist(user *model.User) error {
	var oldUser model.User
	if !s.db.Where("username=?", user.Username).Find(&oldUser).RecordNotFound() {
		return nil
	}
	return s.db.Create(user).Error
}

// GetByUsername returns the user according to the given username.
func (s *sqlite3UserRepo) GetByUsername(username string) (*model.User, error) {
	var user model.User
	if err := s.db.Where("username=?", username).Find(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// ListUsers return users list
func (s *sqlite3UserRepo) ListUsers() ([]*model.User, error) {
	var users []*model.User
	if err := s.db.Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// GetUserCount get user counts
func (s *sqlite3UserRepo) GetUserCount() (count int, err error) {
	if err := s.db.Model(&model.User{}).Count(&count).Error; err != nil {
		return 0, err
	}
	return
}
