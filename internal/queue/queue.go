package queue

import (
	"context"
	"fmt"
	"github.com/sharif-go-lab/go-download-manager/internal/task"
	"github.com/sharif-go-lab/go-download-manager/internal/utils"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

type Queue struct {
	tasks []*task.Task
	directory string
	maxDownloads uint8
	threads uint8
	retries uint8

	limiter <-chan time.Time
	activeInterval *utils.TimeInterval

	ctx context.Context
	cancelFunc context.CancelFunc
}

func NewQueue(directory string, maxDownloads, threads, retries uint8, speedLimit uint64, activeInterval *utils.TimeInterval) *Queue {
	info, err := os.Stat(directory)
	if err != nil || !info.IsDir() { // configurable
		dirname, err := os.UserHomeDir()
		if err != nil {
			slog.Error(fmt.Sprintf("failed to create queue: %v", err))
		}

		directory = filepath.Join(dirname, "Downloads")
		if _, err := os.Stat(directory); os.IsNotExist(err) {
			slog.Error(fmt.Sprintf("failed to create queue: %v", err))
		}
	}
	if maxDownloads == 0 {
		maxDownloads = 3 // configurable
	}
	if threads == 0 {
		threads = 1 // configurable
	}
	ctx, cancelFunc := context.WithCancel(context.Background())

	return &Queue{
		tasks:          make([]*task.Task, 0),
		directory:      directory,
		maxDownloads:   maxDownloads,
		threads:        threads,
		retries:        retries,
		limiter:        utils.CreateLimiter(speedLimit),
		activeInterval: activeInterval,
		ctx:            ctx,
		cancelFunc:     cancelFunc,
	}
}

func (queue *Queue) AddTask(url string) {
	t := task.NewTask(url, queue.directory, queue.threads, queue.retries, queue.limiter)
	queue.tasks = append(queue.tasks, t)
}

func (queue *Queue) Run() {
	if queue.activeInterval != nil {
		queue.activeInterval.WaitUntil()
		queue.ctx, queue.cancelFunc = context.WithDeadline(context.Background(), queue.activeInterval.EndTime())
	} else {
		queue.ctx, queue.cancelFunc = context.WithCancel(context.Background())
	}
	defer queue.cancelFunc()

	slog.Info(fmt.Sprintf("queue %s | starting tasks...", queue.directory))
	for {
		select {
		case <-queue.ctx.Done():
			slog.Info(fmt.Sprintf("queue %s | stopping tasks...", queue.directory))
			for _, t := range queue.tasks {
				if t.Status() == task.InProgress {
					t.Pause()
				}
			}
			break

		default:
			done := true
			downloadCount := uint8(0)
			for _, t := range queue.tasks {
				if downloadCount >= queue.maxDownloads {
					break
				}

				if t.Status() == task.InProgress {
					downloadCount++
					done = false
				} else if t.Status() == task.Created {
					slog.Info(fmt.Sprintf("queue %s | starting task %s...", queue.directory, t.Url()))
					downloadCount++
					done = false
					t.Resume()
				}
			}

			if done {
				break
			}
			time.Sleep(time.Second)
		}
	}
}

func (queue *Queue) Stop() {
	queue.cancelFunc()
}