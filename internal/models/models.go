package models

import "gorm.io/gorm"

type FileMeta struct {
	gorm.Model
	ObjectName   string
	OriginalName string
	Bucket       string
	Size         int

	UserID uint
	User   User
}

type User struct {
	gorm.Model
	Email    string
	Password string
	Name     string
	Role     string
}
