package db

import (
	"collaborative-markdown-editor/internal/document"
	"collaborative-markdown-editor/internal/user"
	"fmt"
	"log"

	"gorm.io/gorm"
)

func Migrate(appDb *gorm.DB) {
	err := appDb.AutoMigrate(&user.User{}, &document.Document{}, &document.DocumentEdit{}, &document.DocumentPermission{})

	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Success Migrating")
}
