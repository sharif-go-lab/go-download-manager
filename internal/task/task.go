package task

import (
	"context"
	"fmt"
	"github.com/sharif-go-lab/go-download-manager/internal/utils"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

type DownloadStatus int
const (
	Created DownloadStatus = iota
	InProgress
	Paused
	Completed
	Canceled
	Failed
)

type Task struct {
	url string
	directoryPath string
	status DownloadStatus

	randomID string
	fileSize int64
	fileName string

	threads uint8
	downloaded []uint64

	mutex sync.Mutex
	ctx context.Context
	cancelFunc context.CancelFunc

	speedLimit uint64 // bytes per second
	retries uint8
}

func NewTask(url, directoryPath string, speedLimit uint64, threads, retires uint8) *Task {
	if threads == 0 {
		threads = 1
	}

	return &Task{
		url: url,
		status: Created,
		directoryPath: directoryPath,
		randomID: uuid.New().String(),

		fileSize: -1,
		threads: threads,
		downloaded: make([]uint64, threads),

		speedLimit: speedLimit,
		retries: retires,
	}
}

func (t *Task) start() {
	logger := log.Default()

	t.mutex.Lock()
	if t.status == InProgress {
		logger.Printf("task %s already started", t.randomID)
		t.mutex.Unlock()
		return
	}
	if t.status == Completed {
		logger.Printf("task %s already completed", t.randomID)
		t.mutex.Unlock()
		return
	}
	if t.status == Canceled {
		logger.Printf("task %s already canceled", t.randomID)
		t.mutex.Unlock()
		return
	}
	t.ctx, t.cancelFunc = context.WithCancel(context.Background())
	t.status = InProgress
	t.mutex.Unlock()

	if t.fileSize == -1 {
		for i := uint8(0); i <= t.retries; i++ {
			resp, err := http.Head(t.url)
			if err == nil {
				t.fileName = utils.FileName(resp)
				t.fileSize = resp.ContentLength
				break
			}

			if i == t.retries {
				logger.Printf("head url %s failed: %v", t.url, err)
				t.status = Failed
				return
			}
		}
	}
	chunkSize := t.fileSize / int64(t.threads)

	filePath := filepath.Join(t.directoryPath, t.fileName)
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		logger.Printf("open file %s failed: %v", filePath, err)
		t.status = Failed
		return
	}
	defer file.Close()

	var wg sync.WaitGroup
	done := make([]bool, t.threads)
	for i := 0; i < int(t.threads); i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			for try := uint8(0); try <= t.retries; try++ {
				start := int64(i) * chunkSize
				end := start + chunkSize - 1
				if i == int(t.threads)-1 {
					end = t.fileSize
				}

				start += int64(t.downloaded[i])
				if start > end {
					break
				}

				req, err := http.NewRequest("GET", t.url, nil)
				if err != nil {
					return
				}
				req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))

				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					return
				}
				defer resp.Body.Close()

				buffer := make([]byte, 1024)
				var limiter <-chan time.Time
				if t.speedLimit > 0 {
					limiter = time.Tick(time.Second / time.Duration(t.speedLimit))
				}
				for {
					select {
					case <-t.ctx.Done():
						return
					default:
						n, err := resp.Body.Read(buffer)
						if n > 0 {
							if limiter != nil {
								<-limiter
							}

							if _, err := file.WriteAt(buffer[:n], start); err != nil {
								return
							}
							t.downloaded[i] += uint64(n)

							start += int64(n)
						}
						if err == io.EOF {
							break
						}
						if err != nil {
							return
						}
					}
				}
			}

			done[i] = true
		}(i)
	}

	wg.Wait()
	for i := 0; i < int(t.threads); i++ {
		if !done[i] {
			logger.Printf("task %s failed on thread %d", t.randomID, i+1)
			t.status = Failed
			return
		}
	}
	t.status = Completed
}

func (t *Task) Pause() {
	t.mutex.Lock()
	if t.status == InProgress {
		t.cancelFunc()
		t.status = Paused
	}
	t.mutex.Unlock()
}

func (t *Task) Resume() {
	t.mutex.Lock()
	if t.status == Paused {
		go t.start()
	}
	t.mutex.Unlock()
}

func (t *Task) Cancel() {
	t.mutex.Lock()
	if t.status == Paused || t.status == InProgress {
		if t.status == InProgress {
			t.cancelFunc()
		}
		t.status = Canceled
		go os.Remove(t.filePath)
	}
	t.mutex.Unlock()
}

func (t *Task) Status() DownloadStatus {
	return t.status
}

func (t *Task) TotalSize() int64 {
	return t.fileSize
}

func (t *Task) Downloaded() uint64 {
	totalDownloaded := uint64(0)
	for _, d := range t.downloaded {
		totalDownloaded += d
	}
	return totalDownloaded
}