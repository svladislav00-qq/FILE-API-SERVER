package models

import "gorm.io/gorm"

type FileMeta struct {
	gorm.Model
	ObjectName   string
	OriginalName string
	Bucket       string
	Size         int
}
