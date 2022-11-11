package models

import "gorm.io/gorm"

type Staffs struct {
	ID        uint    `gorm:"primary key;autoIncreament" json:"id"`
	UserName  *string `json:"userName"`
	FirstName *string `json:"firstName"`
	LastName  *string `json:"lastName"`
	GroupID   uint    `json:"groupID"`
	Groups    Groups  `gorm:"foreignKey:GroupID"`
}

func MigrateStaffs(db *gorm.DB) error {
	err := db.AutoMigrate(&Staffs{})
	return err
}
