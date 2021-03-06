package model

import (
	"github.com/gofrs/uuid"
)

// Star starの構造体
type Star struct {
	UserID    uuid.UUID `gorm:"type:char(36);not null;primary_key"`
	ChannelID uuid.UUID `gorm:"type:char(36);not null;primary_key"`
}

// TableName dbの名前を指定する
func (star *Star) TableName() string {
	return "stars"
}
