package download

import (
	"fmt"
	"sync"
)

// DownloadStatus represents the state of a download
type DownloadStatus int

const (
	Queued DownloadStatus = iota
	InProgress
	Paused
	Completed
	Failed
)

// DownloadTask represents a single download in the queue
type DownloadTask struct {
	URL      string
	DestPath string
	Status   DownloadStatus
}

// DownloadQueue manages download tasks
type DownloadQueue struct {
	Tasks     []*DownloadTask
	WorkerPool *WorkerPool
	mu        sync.Mutex
}

// NewDownloadQueue initializes a new queue
func NewDownloadQueue(workerPool *WorkerPool) *DownloadQueue {
	return &DownloadQueue{
		Tasks:      []*DownloadTask{},
		WorkerPool: workerPool,
	}
}

// AddDownload adds a new download to the queue
func (dq *DownloadQueue) AddDownload(url, destPath string) {
	dq.mu.Lock()
	defer dq.mu.Unlock()

	task := &DownloadTask{
		URL:      url,
		DestPath: destPath,
		Status:   Queued,
	}

	dq.Tasks = append(dq.Tasks, task)
	fmt.Printf("📥 Added to queue: %s\n", url)
}

// StartNextDownload finds the next queued task and starts it
func (dq *DownloadQueue) StartNextDownload() {
	dq.mu.Lock()
	defer dq.mu.Unlock()

	for _, task := range dq.Tasks {
		if task.Status == Queued {
			task.Status = InProgress
			dq.WorkerPool.AddJob(task.URL, task.DestPath)
			return
		}
	}
}

// PauseDownload pauses an active download
func (dq *DownloadQueue) PauseDownload(url string) {
	dq.mu.Lock()
	defer dq.mu.Unlock()

	for _, task := range dq.Tasks {
		if task.URL == url && task.Status == InProgress {
			task.Status = Paused
			fmt.Printf("⏸️ Paused: %s\n", url)
			return
		}
	}
}

// ResumeDownload resumes a paused download
func (dq *DownloadQueue) ResumeDownload(url string) {
	dq.mu.Lock()
	defer dq.mu.Unlock()

	for _, task := range dq.Tasks {
		if task.URL == url && task.Status == Paused {
			task.Status = Queued
			fmt.Printf("▶️ Resumed: %s\n", url)
			return
		}
	}
}

// ListDownloads prints all downloads with their statuses
func (dq *DownloadQueue) ListDownloads() {
	dq.mu.Lock()
	defer dq.mu.Unlock()

	fmt.Println("\n📜 Download Queue:")
	for _, task := range dq.Tasks {
		status := ""
		switch task.Status {
		case Queued:
			status = "Queued ⏳"
		case InProgress:
			status = "Downloading ⬇️"
		case Paused:
			status = "Paused ⏸️"
		case Completed:
			status = "Completed ✅"
		case Failed:
			status = "Failed ❌"
		}
		fmt.Printf("- %s [%s]\n", task.URL, status)
	}
}