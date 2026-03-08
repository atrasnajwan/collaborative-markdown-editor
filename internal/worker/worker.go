package worker

import (
	"context"
	"sync"
	"sync/atomic"

	log "github.com/rs/zerolog/log"
)

// Task is a function that represents a background job
type Task func(ctx context.Context) error

type WorkerPool struct {
	taskQueue chan Task
	wg        sync.WaitGroup
	isClosing atomic.Bool // thread-safe value
}

func NewWorkerPool(size int) *WorkerPool {
	wp := &WorkerPool{
		taskQueue: make(chan Task, 1000), // Buffer for 1000 pending tasks
	}

	// Start the workers
	for range size {
		wp.wg.Add(1) // add to WaitGroup
		go wp.startWorker()
	}

	return wp
}

func (wp *WorkerPool) startWorker() {
	defer wp.wg.Done() // signal when worker finished
	for task := range wp.taskQueue {
		ctx := context.Background()
		if err := task(ctx); err != nil { // run task
			log.Error().Err(err).Msg("Worker task failed")
		}
	}
}

func (wp *WorkerPool) Submit(t Task) {
	if wp.isClosing.Load() {
		log.Warn().Msg("task submitted during shutdown, dropping")
		return
	}
	select {
	case wp.taskQueue <- t: // send task to worker pool
	default:
		log.Warn().Msg("Task queue full, dropping task")
	}
}

// Shutdown closes the queue and waits for workers to finish
func (wp *WorkerPool) Shutdown() {
	wp.isClosing.Store(true)
	close(wp.taskQueue) // Stop accepting new tasks
	wp.wg.Wait()        // Wait for all active workers to finish tasks
}
