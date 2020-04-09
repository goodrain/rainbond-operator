package middleware

import (
	"fmt"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"

	"github.com/goodrain/rainbond-operator/pkg/openapi/user"
)

// Authenticate -
func Authenticate(secretKey string, exptime time.Duration, userRepo user.Repository) gin.HandlerFunc {
	// TODO use code instead of map[string]interface{} for return
	return func(c *gin.Context) {
		userCount, err := userRepo.GetUserCount()
		if err != nil {
			c.AbortWithStatusJSON(500, map[string]interface{}{"msg": "get user failed"})
			return
		}

		if userCount == 0 {
			logrus.Info("do not generate user now, do not need authenticate")
			return
		}

		logrus.Info("generated user, do authenticate")

		tokenStr := c.GetHeader("Authorization")
		if strings.TrimSpace(tokenStr) == "" {
			c.AbortWithStatusJSON(401, map[string]interface{}{"msg": "emtpy token"})
			return
		}

		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
			// Don't forget to validate the alg is what you expect:
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
			}
			// hmacSampleSecret is a []byte containing your secret, e.g. []byte("my_secret_key")
			return []byte(secretKey), nil // TODO: use real secret key
		})
		if err != nil {
			c.AbortWithStatusJSON(403, map[string]interface{}{"msg": "parse failed, invalid token"})
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok || !token.Valid {
			c.AbortWithStatusJSON(403, map[string]interface{}{"msg": "invalid token"})
			return
		}
		nbf := claims["nbf"].(float64)
		nbftime := time.Unix(int64(nbf), 0)
		if time.Since(nbftime) > exptime {
			c.AbortWithStatusJSON(401, map[string]interface{}{"msg": "token expired"})
			return
		}
		// TODO: thread safe
		username := claims["username"]
		user, err := userRepo.GetByUsername(username.(string))
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				c.AbortWithStatusJSON(400, map[string]interface{}{"msg": "user not found"})
				return
			}
			c.AbortWithStatusJSON(500, map[string]interface{}{"msg": "get user failed"})
			return
		}
		c.Set("user", user)
	}
}
