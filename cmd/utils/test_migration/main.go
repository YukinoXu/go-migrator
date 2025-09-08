package main

import (
	"flag"
	"log"
	"os"

	"example.com/go-migrator/internal/migrator"
	"example.com/go-migrator/internal/model"
	"example.com/go-migrator/internal/store"
	"github.com/joho/godotenv"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	// load local .env for convenience
	_ = godotenv.Load()

	zoomUserID := os.Getenv("ZOOM_TEST_USER_ID")
	zoomChannelID := os.Getenv("ZOOM_TEST_CHANNEL_ID")
	mysqlDSN := os.Getenv("MYSQL_DSN")

	teamName := flag.String("teamName", "", "Teams team Name to migrate to")
	channelName := flag.String("channelName", "", "Teams channel Name to migrate to")
	flag.Parse()

	db, _ := gorm.Open(mysql.Open(mysqlDSN), &gorm.Config{})
	db.AutoMigrate(&model.Task{}, &model.Identity{}, &model.Project{}, &model.Connector{})
	stm := store.NewStoreManager(db)

	if zoomUserID == "" {
		log.Fatal("ZOOM_TEST_USER_ID not set in env")
	}
	if *teamName == "" {
		log.Fatal("-teamName is required")
	}
	if *channelName == "" {
		log.Fatal("-channelName is required")
	}

	if err := migrator.MigrateTask(zoomUserID, zoomChannelID, *teamName, *channelName, stm); err != nil {
		log.Fatalf("migration failed: %v", err)
	}
	log.Println("migration finished")
}
