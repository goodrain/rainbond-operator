package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"

	"github.com/goodrain/rainbond-operator/pkg/library/bcode"
	"github.com/goodrain/rainbond-operator/pkg/openapi/user"
)

// Authenticate -
func Authenticate(secretKey string, exptime time.Duration, userRepo user.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		userCount, err := userRepo.GetUserCount()
		if err != nil {
			data := map[string]interface{}{"code": bcode.ServerErr.Code(), "msg": bcode.ServerErr.Msg()}
			c.AbortWithStatusJSON(http.StatusInternalServerError, data)
			return
		}

		if userCount == 0 {
			logrus.Info("do not generate user now, do not need authenticate")
			return
		}

		logrus.Info("generated user, do authenticate")

		tokenStr := c.GetHeader("Authorization")
		logrus.Debugf("token str is: %s", tokenStr)
		if strings.TrimSpace(tokenStr) == "" {
			data := map[string]interface{}{"code": bcode.EmptyToken.Code(), "msg": bcode.EmptyToken.Msg()}
			c.AbortWithStatusJSON(http.StatusUnauthorized, data)
			return
		}

		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			// Don't forget to validate the alg is what you expect:
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("Unexpected sign method: %v", token.Header["alg"])
			}
			// hmacSampleSecret is a []byte containing your secret, e.g. []byte("my_secret_key")
			return []byte(secretKey), nil
		})
		if err != nil {
			data := map[string]interface{}{"code": bcode.InvalidToken.Code(), "msg": bcode.InvalidToken.Msg()}
			c.AbortWithStatusJSON(http.StatusForbidden, data)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok || !token.Valid {
			data := map[string]interface{}{"code": bcode.InvalidToken.Code(), "msg": bcode.InvalidToken.Msg()}
			c.AbortWithStatusJSON(http.StatusForbidden, data)
			return
		}
		nbf := claims["nbf"].(float64)
		nbftime := time.Unix(int64(nbf), 0)
		if time.Since(nbftime) > exptime {
			data := map[string]interface{}{"code": bcode.ExpiredToken.Code(), "msg": bcode.ExpiredToken.Msg()}
			c.AbortWithStatusJSON(http.StatusUnauthorized, data)
			return
		}
		// TODO: thread safe
		username := claims["username"]
		userInfo, err := userRepo.GetByUsername(username.(string))
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				data := map[string]interface{}{"code": bcode.UserNotFound.Code(), "msg": bcode.UserNotFound.Msg()}
				c.AbortWithStatusJSON(http.StatusBadRequest, data)
				return
			}
			data := map[string]interface{}{"code": bcode.ServerErr.Code(), "msg": bcode.ServerErr.Msg()}
			c.AbortWithStatusJSON(http.StatusInternalServerError, data)
			return
		}
		c.Set("user", userInfo)
	}
}
