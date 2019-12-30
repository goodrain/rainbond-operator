package usecase

import (
	"errors"
	"github.com/dgrijalva/jwt-go"
	"github.com/jinzhu/gorm"
	"time"

	"github.com/GLYASAI/rainbond-operator/pkg/openapi/model"
	"github.com/GLYASAI/rainbond-operator/pkg/openapi/user"
)

var (
	UserNotFound  = errors.New("user not found")
	WrongPassword = errors.New("wrong password")
)

type userUsecase struct {
	secretKey string
	userRepo  user.Repository
}

// NewUserUsecase creates a new user.Usecase.
func NewUserUsecase(userRepo user.Repository, secretKey string) user.Usecase {
	ucase := &userUsecase{
		userRepo:  userRepo,
		secretKey: secretKey,
	}

	return ucase
}

func (u *userUsecase) Login(username, password string) (string, error) {
	user, err := u.userRepo.GetByUsername(username)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", UserNotFound
		}
		return "", err
	}

	if user.Password != password {
		return "", WrongPassword
	}

	token, err := GenerateToken(user, u.secretKey)
	if err != nil {
		return "", err
	}

	return token, nil
}

// GenerateToken generate a json web token
func GenerateToken(user *model.User, secretKey string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss":      "goodrain",
		"nbf":      time.Now().Unix(),
		"username": user.Username,
	})

	tokenStr, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", err
	}

	return tokenStr, nil
}
