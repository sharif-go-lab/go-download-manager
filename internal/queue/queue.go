package queue

import (
	"context"
	"errors"
	"fmt"
	"github.com/sharif-go-lab/go-download-manager/internal/task"
	"github.com/sharif-go-lab/go-download-manager/internal/utils"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Queue struct {
	tasks          []*task.Task
	Name           string
	Directory      string
	MaxDownloads   uint8
	Threads        uint8
	Retries        uint8
	SpeedLimit     uint64
	limiter        <-chan time.Time
	activeInterval *utils.TimeInterval

	ctx        context.Context
	cancelFunc context.CancelFunc
}

func NewQueue(name string, directory string, maxDownloads, threads, retries uint8, speedLimit uint64, activeInterval *utils.TimeInterval) *Queue {
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
		Name:           name,
		Directory:      directory,
		MaxDownloads:   maxDownloads,
		Threads:        threads,
		Retries:        retries,
		SpeedLimit:     speedLimit,
		limiter:        utils.CreateLimiter(speedLimit),
		activeInterval: activeInterval,
		ctx:            ctx,
		cancelFunc:     cancelFunc,
	}
}

func (queue *Queue) AddTask(url string, directory string) error {
	var dir string
	fmt.Println(directory)
	if directory == "" {
		dir = queue.Directory

	} else {
		fmt.Println("sdt")
		info, err := os.Stat(directory)
		if err != nil || !info.IsDir() {
			dirname, err := os.UserHomeDir()
			if err != nil {
				slog.Error(fmt.Sprintf("failed to get user home directory: %v", err))
				return fmt.Errorf("failed to get user home directory: %w", err)
			}

			dir = filepath.Join(dirname, directory)

			if _, err := os.Stat(dir); os.IsNotExist(err) {
				slog.Error(fmt.Sprintf("folder does not exist: %v", directory))
				return fmt.Errorf("folder does not exist: %s", directory)
			}
				//os.Exit(0)


		}
	}
	t := task.NewTask(url, dir, queue.Threads, queue.Retries, queue.limiter)
	queue.tasks = append(queue.tasks, t)
	return nil

}

func (queue *Queue) Run() {
	if queue.activeInterval != nil {
		queue.activeInterval.WaitUntil()
		queue.ctx, queue.cancelFunc = context.WithDeadline(context.Background(), queue.activeInterval.EndTime())
	} else {
		queue.ctx, queue.cancelFunc = context.WithCancel(context.Background())
	}
	defer queue.cancelFunc()

	slog.Info(fmt.Sprintf("queue %s | starting tasks...", queue.Directory))
	for {
		select {
		case <-queue.ctx.Done():
			slog.Info(fmt.Sprintf("queue %s | stopping tasks...", queue.Directory))
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
				if downloadCount >= queue.MaxDownloads {
					break
				}

				if t.Status() == task.InProgress {
					downloadCount++
					done = false
				} else if t.Status() == task.Pending {
					slog.Info(fmt.Sprintf("queue %s | starting task %s...", queue.Directory, t.Url()))
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
func (queue *Queue) Tasks() []*task.Task {
	return queue.tasks
}

func (queue *Queue) SetName(name string) {
	queue.Name = name
}
func (queue *Queue) SetDirectory(folder string) error {
	info, err := os.Stat(folder)
	if err != nil || !info.IsDir() {
		dirname, err := os.UserHomeDir()
		if err != nil {
			slog.Error(fmt.Sprintf("failed to get user home directory: %v", err))
			return fmt.Errorf("failed to get user home directory: %w", err)
		}

		queue.Directory = filepath.Join(dirname, folder)

		if _, err := os.Stat(queue.Directory); os.IsNotExist(err) {
			slog.Error(fmt.Sprintf("folder does not exist: %v", folder))
			return fmt.Errorf("folder does not exist: %s", folder)
		}

	}
	return nil
}

func (queue *Queue) SetMaxDownloads(n uint8) {
	queue.MaxDownloads = n
}
func (queue *Queue) SetSpeedLimit(limit uint64) {
	queue.limiter = utils.CreateLimiter(limit)
}

func parseUserInterval(input string) (*utils.TimeInterval, error) {
	parts := strings.Split(input, "-")
	if len(parts) != 2 {
		return nil, errors.New("invalid interval format (expected 08:00:00-17:00:00)")
	}
	start := strings.TrimSpace(parts[0])
	end := strings.TrimSpace(parts[1])

	ti, err := utils.NewTimeInterval(start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to parse time interval: %w", err)
	}
	return ti, nil
}
func (q *Queue) SetActiveIntervalFromString(input string) error {
	if input == "always"|| input == "Always" {
	return nil
	}
	ti, err := parseUserInterval(input)
	if err != nil {
		return err
	}
	q.activeInterval = ti
	return nil
}
