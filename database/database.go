package database

import (
	"go-customer-app/models"
	"log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() {
	var err error
	// SQLiteデータベースに接続します。ファイルが存在しない場合は作成されます。
	DB, err = gorm.Open(sqlite.Open("customers.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("DB接続失敗： %v", err)
	}

	// Customerモデルに基づいてテーブルを自動マイグレーションします。
	DB.AutoMigrate(&models.Customer{})
}
