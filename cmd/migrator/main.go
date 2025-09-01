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
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// persistent MySQL store required
	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" {
		log.Fatal("MYSQL_DSN is required")
	}
	st, err := store.NewMySQLStore(dsn)
	if err != nil {
		log.Fatalf("failed to open mysql store: %v", err)
	}
	log.Println("using MySQL store")

	// configure RabbitMQ
	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		rabbitURL = "amqp://guest:guest@localhost:5672/"
	}
	qclient, err := queue.NewRabbitClient(rabbitURL, "migrator-tasks")
	if err != nil {
		log.Fatalf("failed to connect to rabbitmq: %v", err)
	}
	defer qclient.Close()

	wk := worker.NewWorker(st, qclient, 4)
	wk.Start(ctx)

	h := api.NewHandler(st)
	srv := &http.Server{
		Addr:    ":8080",
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
