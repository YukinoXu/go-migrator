package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"example.com/go-migrator/internal/api"
	"example.com/go-migrator/internal/queue"
	"example.com/go-migrator/internal/store"
	"example.com/go-migrator/internal/worker"
	"github.com/joho/godotenv"
)

func main() {
	// load .env into the process environment so os.Getenv works for all packages
	if err := godotenv.Load(); err != nil {
		log.Fatalf("failed to load .env file: %v", err)
	}

	get := func(key string) string { return os.Getenv(key) }

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// persistent MySQL store required
	dsn := get("MYSQL_DSN")
	if dsn == "" {
		log.Fatal("MYSQL_DSN is required in .env")
	}
	// use GORM-backed store
	st, err := store.NewGormStore(dsn)
	if err != nil {
		log.Fatalf("failed to open gorm store: %v", err)
	}
	// ensure migrations have run (idempotent)
	if err := store.RunMigrations(st.DB()); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	// configure RabbitMQ
	rabbitURL := get("RABBITMQ_URL")
	if rabbitURL == "" {
		log.Fatal("RABBITMQ_URL is required in .env")
	}
	qclient, err := queue.NewRabbitClient(rabbitURL, "migrator-tasks")
	if err != nil {
		log.Fatalf("failed to connect to rabbitmq: %v", err)
	}
	defer qclient.Close()

	wk := worker.NewWorker(st, qclient, 4)
	wk.Start(ctx)

	h := api.NewHandler(st, qclient)
	srv := &http.Server{
		Addr: ":" + func() string {
			if p := get("PORT"); p != "" {
				return p
			}
			return "8080"
		}(),
		Handler: h.Router(),
	}

	go func() {
		log.Printf("server listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("shutting down")
	ctxSh, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctxSh)
}
