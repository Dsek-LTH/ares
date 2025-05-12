package db

import "database/sql"

type User struct {
	StilId   string `gorm:"primaryKey"`
	ImageUrl string `gorm:"not null"`
	Name     string `gorm:"not null"`
}

type Hunt struct {
	HunterId string `gorm:"primaryKey"`
	TargetId string `gorm:"primaryKey"`
	VideoUrl *string
	KilledAt sql.NullTime
	Hunter   User `gorm:"foreignKey:HunterId;references:StilId"`
	Target   User `gorm:"foreignKey:TargetId;references:StilId"`
}

type SignUpData struct {
	Name   string `json:"name"`
	StilId string `json:"stil-id"`
}
