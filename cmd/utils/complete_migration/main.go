package main

import (
	"flag"
	"fmt"
	"log"

	"example.com/go-migrator/internal/migrator"
	"github.com/joho/godotenv"
)

func main() {
	// load local .env for convenience
	_ = godotenv.Load()
	teamID := flag.String("teamId", "", "Teams team ID to complete migration for")
	flag.Parse()

	if *teamID == "" {
		fmt.Println("Usage: go run cmd\\utils\\complete_migration\\main.go -teamId \"<team-id>\"")
		log.Fatal("-teamId is required")
	}

	if err := migrator.CompleteMigration(*teamID); err != nil {
		log.Fatalf("complete migration failed: %v", err)
	}
}
