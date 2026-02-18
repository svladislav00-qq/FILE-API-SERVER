package models

import "gorm.io/gorm"

type FileMeta struct {
	gorm.Model
	Name     string
	Location string
	Size     int
}
