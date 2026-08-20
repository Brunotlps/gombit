package main

import (
	"fmt"
	"log"

	"github.com/gombit-dev/gombit/config"
	"github.com/gombit-dev/gombit/database"
)

type widget struct {
	ID   uint `gorm:"primaryKey"`
	Name string
}

func main() {
	cfg := config.Default()
	cfg.Database = config.DatabaseConfig{
		Driver: config.DatabaseDriverSQLite,
		DSN:    "file:example?mode=memory&cache=shared&_fk=1",
	}

	db, err := database.Open(cfg.Database)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("close database: %v", err)
		}
	}()

	if err := db.AutoMigrate(&widget{}); err != nil {
		log.Fatal(err)
	}
	if err := db.Create(&widget{Name: "first"}).Error; err != nil {
		log.Fatal(err)
	}

	fmt.Printf("driver=%s returning=%t\n", db.Driver(), db.Capabilities().Returning)
}
