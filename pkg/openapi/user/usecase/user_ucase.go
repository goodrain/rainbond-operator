package usecase

import (
	"errors"
	"time"

	"github.com/goodrain/rainbond-operator/pkg/util/passwordutil"

	"github.com/dgrijalva/jwt-go"
	"github.com/jinzhu/gorm"
	"github.com/sethvargo/go-password/password"

	"github.com/goodrain/rainbond-operator/pkg/library/bcode"
	"github.com/goodrain/rainbond-operator/pkg/openapi/model"
	"github.com/goodrain/rainbond-operator/pkg/openapi/user"
)

var (
	UserNotFound  = errors.New("user not found")
	WrongPassword = errors.New("wrong password")
	// TempMail temp email for encrypt and verify password
	TempMail = "operator@goodrain.com"
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
	users, err := u.userRepo.ListUsers()
	if err != nil {
		return nil, err
	}
	if len(users) > 0 {
		return nil, bcode.DoNotAllowGenerateAdmin
	}

	// generate password len is 8 and all digital of password is number, such as 38726051
	pass, _ := password.Generate(8, 8, 0, false, false)
	userInfo := model.User{
		Username: "admin",
		Password: pass,
	}

	encryPass, err := passwordutil.EncryptionPassword(pass, TempMail)
	if err != nil {
		return nil, err
	}

	userInfo.Password = encryPass

	err = u.userRepo.CreateIfNotExist(&userInfo)
	if err != nil {
		return nil, err
	}
	return &model.User{Username: "admin", Password: pass}, nil
}

// Login -
func (u *userUsecase) Login(username, password string) (string, error) {
	userInfo, err := u.userRepo.GetByUsername(username)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", UserNotFound
		}
		return "", err
	}

	if !passwordutil.CheckPassword(password, userInfo.Password, TempMail) {
		return "", bcode.UserPasswordInCorrect
	}

	token, err := GenerateToken(userInfo, u.secretKey)
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
