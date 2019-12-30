package model

import "github.com/jinzhu/gorm"

// User represents the user model
type User struct {
	gorm.Model
	Username string `gorm="username" json:"username"`
	Password string `gorm="password" json:"password"`
}
