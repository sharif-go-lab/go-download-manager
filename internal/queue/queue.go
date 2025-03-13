package queue

import (
	"context"
	"github.com/sharif-go-lab/go-download-manager/internal/task"
	"github.com/sharif-go-lab/go-download-manager/internal/utils"
	"time"
)

type Queue struct {
	tasks []*task.Task
	directory string
	speedLimit uint64
	threads uint8
	retries uint8
	activeInterval *utils.TimeInterval
}

func NewQueue(directory string, speedLimit uint64, threads, retries uint8, activeInterval *utils.TimeInterval) *Queue {
	return &Queue{
		tasks:          make([]*task.Task, 0),
		directory:      directory,
		speedLimit:     speedLimit,
		threads:        threads,
		retries:        retries,
		activeInterval: activeInterval,
	}
}

func (queue *Queue) AddTask(url string) {
	t := task.NewTask(url, queue.directory, queue.speedLimit, queue.threads, queue.retries)
	queue.tasks = append(queue.tasks, t)
}

func (queue *Queue) Run() {
	time.Sleep(queue.activeInterval.StartTime.Sub(time.Now()))
	ctx, cancelFunc := context.WithDeadline(context.Background(), queue.activeInterval.EndTime)
	defer cancelFunc()

	for {
		select {
		case <-ctx.Done():
			for _, t := range queue.tasks {
				if t.Status() == task.InProgress {
					t.Pause()
				}
			}
			break
		default:
			done := true
			for _, t := range queue.tasks {
				if t.Status() == task.InProgress {
					done = false
					break
				} else if t.Status() == task.Created {
					done = false
					t.Resume()
					break
				}
			}
			if done {
				break
			}
		}
	}
}