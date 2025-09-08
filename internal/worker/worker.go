package worker

import (
	"context"
	"log"
	"sync"
	"time"

	"example.com/go-migrator/internal/model"
	"example.com/go-migrator/internal/queue"
	"example.com/go-migrator/internal/store"
)

type Worker struct {
	stm        *store.StoreManager
	qclient    queue.Client
	workerPool int
	wg         sync.WaitGroup
}

func NewWorker(s *store.StoreManager, q queue.Client, pool int) *Worker {
	return &Worker{stm: s, qclient: q, workerPool: pool}
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
	t, err := w.stm.Task.GetByID(id)
	if err != nil {
		log.Printf("task %s not found: %v", id, err)
		return
	}

	t.Status = model.StatusRunning
	_ = w.stm.Task.UpdateStatus(t.ID, (string)(model.StatusRunning))

	// // pass the store directly; Store includes identity methods
	// err = migrator.MigrateTask(zoomUserID, zoomChannelID, teamName, channelName, w.store)
	// if err != nil {
	// 	log.Printf("task %s failed: %v", id, err)
	// 	t.Status = model.StatusFailed
	// 	t.Error = err.Error()
	// } else {
	// 	log.Printf("task %s succeeded", id)
	// 	t.Status = model.StatusSuccess
	// 	t.Result = "migrated"
	// }
	// _ = w.stm.Task.UpdateStatus(t.ID, t.Status)
	// small sleep to avoid busy loops in tests
	time.Sleep(10 * time.Millisecond)
}
