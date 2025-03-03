package download

import (
	"fmt"
	"sync"
	"time"
)

// ScheduledTask represents a download scheduled for a later time
type ScheduledTask struct {
	URL      string
	DestPath string
	StartAt  time.Time
	Active   bool
}

// Scheduler manages scheduled downloads
type Scheduler struct {
	Tasks []*ScheduledTask
	Queue *DownloadQueue
	mu    sync.Mutex
}

// NewScheduler initializes a new scheduler
func NewScheduler(queue *DownloadQueue) *Scheduler {
	return &Scheduler{
		Tasks: []*ScheduledTask{},
		Queue: queue,
	}
}

// ScheduleDownload adds a download task to be executed at a specific time
func (s *Scheduler) ScheduleDownload(url, destPath string, startAt time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task := &ScheduledTask{
		URL:      url,
		DestPath: destPath,
		StartAt:  startAt,
		Active:   true,
	}

	s.Tasks = append(s.Tasks, task)

	// Start a goroutine to wait until the scheduled time
	go func(task *ScheduledTask) {
		duration := time.Until(task.StartAt)

		if duration > 0 {
			fmt.Printf("⏳ Scheduled: %s at %v\n", task.URL, task.StartAt)
			time.Sleep(duration) // Wait until the scheduled time
		}

		// Only start the task if it's still active
		s.mu.Lock()
		if task.Active {
			fmt.Printf("🚀 Starting scheduled download: %s\n", task.URL)
			s.Queue.AddDownload(task.URL, task.DestPath)
			s.Queue.StartNextDownload()
		}
		s.mu.Unlock()
	}(task)
}

// CancelScheduledDownload removes a scheduled task before it starts
func (s *Scheduler) CancelScheduledDownload(url string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, task := range s.Tasks {
		if task.URL == url && task.Active {
			task.Active = false
			fmt.Printf("❌ Canceled scheduled download: %s\n", url)
			return
		}
	}
}

// ListScheduledDownloads prints all scheduled downloads
func (s *Scheduler) ListScheduledDownloads() {
	s.mu.Lock()
	defer s.mu.Unlock()

	fmt.Println("\n📅 Scheduled Downloads:")
	for _, task := range s.Tasks {
		if task.Active {
			fmt.Printf("- %s [Scheduled for %v]\n", task.URL, task.StartAt)
		}
	}
}