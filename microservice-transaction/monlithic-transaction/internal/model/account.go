package model

type Account struct {
	ID      uint   `gorm:"primaryKey" json:"id"`
	Name    string `gorm:"uniqueIndex;not null" json:"name"`
	Balance int64  `gorm:"not null" json:"balance"`
}
