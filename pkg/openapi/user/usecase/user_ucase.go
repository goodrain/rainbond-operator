package usecase

import (
	"errors"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/jinzhu/gorm"
	"github.com/sethvargo/go-password/password"

	"github.com/goodrain/rainbond-operator/pkg/openapi/model"
	"github.com/goodrain/rainbond-operator/pkg/openapi/user"
)

var (
	UserNotFound  = errors.New("user not found")
	WrongPassword = errors.New("wrong password")
	NotAllow      = errors.New("do not allow more than one administrator")
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

// GenerateUser -
func (u userUsecase) GenerateUser() (*model.User, error) {
	users, err := u.userRepo.Listusers()
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	if len(users) > 0 {
		return nil, NotAllow
	}

	// generate password len is 8 and all digital of password is number, such as 38726051
	pass, _ := password.Generate(8, 8, 0, false, false)
	userInfo := model.User{
		Username: "admin",
		Password: pass,
	}
	err = u.userRepo.CreateIfNotExist(&userInfo)
	if err != nil {
		return nil, err
	}
	return &userInfo, nil
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
