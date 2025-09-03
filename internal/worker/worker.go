package worker

import (
	"context"
	"log"
	"sync"
	"time"

	"example.com/go-migrator/internal/migrator"
	"example.com/go-migrator/internal/model"
	"example.com/go-migrator/internal/queue"
	"example.com/go-migrator/internal/store"
)

type Worker struct {
	store      store.Store
	qclient    queue.Client
	workerPool int
	wg         sync.WaitGroup
}

func NewWorker(s store.Store, q queue.Client, pool int) *Worker {
	return &Worker{store: s, qclient: q, workerPool: pool}
}

func (w *Worker) Start(ctx context.Context) {
	for i := 0; i < w.workerPool; i++ {
		w.wg.Add(1)
		go func(idx int) {
			defer w.wg.Done()
			log.Printf("worker %d started", idx)
			msgs, err := w.qclient.Consume(ctx)
			if err != nil {
				log.Printf("worker %d failed to consume: %v", idx, err)
				return
			}
			for {
				select {
				case <-ctx.Done():
					log.Printf("worker %d stopping", idx)
					return
				case id, ok := <-msgs:
					if !ok {
						log.Printf("worker %d messages channel closed", idx)
						return
					}
					w.process(id)
				}
			}
		}(i)
	}
	// wait in background for ctx cancellation then wg
	go func() {
		<-ctx.Done()
		log.Println("waiting for workers to finish...")
		w.wg.Wait()
	}()
}

func (w *Worker) process(id string) {
	log.Printf("processing task %s", id)
	t, err := w.store.GetTask(id)
	if err != nil {
		log.Printf("task %s not found: %v", id, err)
		return
	}

	t.Status = model.StatusRunning
	_ = w.store.UpdateTask(t)

	// execute migration via orchestrator adapter
	zoomUserID := t.Payload["zoom_user_id"]
	zoomChannelID := t.Payload["zoom_channel_id"]
	teamName := t.Payload["team_name"]
	channelName := t.Payload["channel_name"]
	// pass the store directly; Store includes identity methods
	err = migrator.MigrateTask(zoomUserID, zoomChannelID, teamName, channelName, w.store)
	if err != nil {
		log.Printf("task %s failed: %v", id, err)
		t.Status = model.StatusFailed
		t.Error = err.Error()
	} else {
		log.Printf("task %s succeeded", id)
		t.Status = model.StatusSuccess
		t.Result = "migrated"
	}
	_ = w.store.UpdateTask(t)
	// small sleep to avoid busy loops in tests
	time.Sleep(10 * time.Millisecond)
}
