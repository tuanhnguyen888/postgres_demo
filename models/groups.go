package models

import "gorm.io/gorm"

type Groups struct {
	ID            int     `gorm:"primary key;autoIncreament" json:"id"`
	GroupName     *string `json:"groupName"`
	ParentGroupID int     `gorm:"foreignKey:ID" json:"parentGroupID"`
}

func MigrateGroups(db *gorm.DB) error {
	err := db.AutoMigrate(&Groups{})
	return err
}
