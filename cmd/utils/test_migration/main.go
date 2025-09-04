package main

import (
	"flag"
	"log"
	"os"

	"example.com/go-migrator/internal/migrator"
	"example.com/go-migrator/internal/store"
	"github.com/joho/godotenv"
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

	var idStore store.Store
	if mysqlDSN != "" {
		s, err := store.NewGormStore(mysqlDSN)
		if err != nil {
			log.Fatalf("failed to open mysql store: %v", err)
		}
		// MySQLStore implements store.Store (including identity methods)
		idStore = s
		log.Println("using MySQL identity store")
	} else {
		log.Println("no MYSQL_DSN set; running without identity store")
	}

	if zoomUserID == "" {
		log.Fatal("ZOOM_TEST_USER_ID not set in env")
	}
	if *teamName == "" {
		log.Fatal("-teamName is required")
	}
	if *channelName == "" {
		log.Fatal("-channelName is required")
	}

	if err := migrator.MigrateTask(zoomUserID, zoomChannelID, *teamName, *channelName, idStore); err != nil {
		log.Fatalf("migration failed: %v", err)
	}
	log.Println("migration finished")
}
