package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserType string

const (
	UserTypeRider  UserType = "RIDER"
	UserTypeDriver UserType = "DRIVER"
)

type User struct {
	ID             uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	FullName       string         `gorm:"type:varchar(255);not null"`
	Email          string         `gorm:"type:varchar(255);uniqueIndex;not null"`
	HashedPassword string         `gorm:"type:varchar(255);not null"`
	UserType       UserType       `gorm:"type:varchar(20);not null;check:user_type IN ('RIDER', 'DRIVER')"`
	CreatedAt      time.Time      `gorm:"autoCreateTime"`
	UpdatedAt      time.Time      `gorm:"autoUpdateTime"`
	DeletedAt      gorm.DeletedAt `gorm:"index"`
}

func (User) TableName() string {
	return "users"
}
