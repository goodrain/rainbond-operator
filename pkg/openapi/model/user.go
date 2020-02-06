package model

import "github.com/jinzhu/gorm"

// User represents the user model
type User struct {
	gorm.Model
	Username string `json:"username" gorm:"username" `
	Password string `json:"password" gorm:"password"`
}
