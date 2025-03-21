package task

import (
	"context"
	"fmt"
	"github.com/sharif-go-lab/go-download-manager/internal/utils"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type DownloadStatus int
const (
	Pending DownloadStatus = iota
	InProgress
	Paused
	Completed
	Canceled
	Failed
)

type Task struct {
	url           string
	DirectoryPath string
	status        DownloadStatus

	fileSize int64
	filePath string

	threads uint8
	downloaded []uint64

	mutex sync.Mutex
	ctx context.Context
	cancelFunc context.CancelFunc

	retries uint8
	limiter <-chan time.Time
}

func NewTask(url, directoryPath string, threads, retires uint8, limiter <-chan time.Time) *Task {
	return &Task{
		url:           url,
		status:        Pending,
		retries:       retires,
		DirectoryPath: directoryPath,

		fileSize: -1,
		threads: threads,
		limiter: limiter,
		downloaded: make([]uint64, threads),
	}
}
func (t *Task)setDirectory(directory string)  {
	t.DirectoryPath = directory
}
func (t *Task) start() {
	t.mutex.Lock()
	if t.status == InProgress {
		slog.Error("task already started")
		t.mutex.Unlock()
		return
	} else if t.status == Completed {
		slog.Error("task already completed")
		t.mutex.Unlock()
		return
	} else if t.status == Canceled {
		slog.Error("task already canceled")
		t.mutex.Unlock()
		return
	}
	t.ctx, t.cancelFunc = context.WithCancel(context.Background())
	t.status = InProgress
	t.mutex.Unlock()

	if t.fileSize == -1 {
		for try := uint8(0); try <= t.retries; try++ {
			resp, err := http.Head(t.url)
			if err == nil {
				t.filePath = filepath.Join(t.DirectoryPath, utils.FileName(resp))
				t.fileSize = resp.ContentLength
				if _, err := os.Stat(t.filePath); err == nil {
					// If file already exists, pick a unique name:
					newPath := utils.FindUniqueFilePath(t.filePath)
					//	slog.Debug(fmt.Sprintf("task %s exists, using new name %s", t.filePath, newPath))
					t.filePath = newPath
				}
				slog.Debug(fmt.Sprintf("task %s | retry %d | file size: %d", t.filePath, try, t.fileSize))
				break
			}

			if try == t.retries {
				slog.Error(fmt.Sprintf("task %s | retry %d | head url %s failed: %v", t.filePath, try, t.url, err))
				t.status = Failed
				return
			}
			time.Sleep(time.Second * (1 << try))
		}
	}
	chunkSize := t.fileSize / int64(t.threads)

	//if t.status != Paused {
	//if _, err := os.Stat(t.filePath); err == nil {
		// If file already exists, pick a unique name:
		//newPath := utils.FindUniqueFilePath(t.filePath)
	//	slog.Debug(fmt.Sprintf("task %s exists, using new name %s", t.filePath, newPath))
	//	t.filePath = newPath
	//}
	//}

	file, err := os.OpenFile(t.filePath, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		slog.Error(fmt.Sprintf("task %s | open file failed: %v", t.filePath, err))
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
				slog.Debug(fmt.Sprintf("task %s | thread %d | retry %d | downloading part %d-%d...", t.filePath, i+1, try, start, end))

				req, err := http.NewRequest("GET", t.url, nil)
				if err != nil {
					slog.Error(fmt.Sprintf("task %s | thread %d | retry %d | create request failed: %v", t.filePath, i+1, try, err))
					return
				}
				req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))

				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					slog.Error(fmt.Sprintf("task %s | thread %d | retry %d | send request failed: %v", t.filePath, i+1, try, err))
					return
				}
				defer resp.Body.Close()

				buffer := make([]byte, 1024)
				for {
					select {
					case <-t.ctx.Done():
						slog.Debug(fmt.Sprintf("task %s | thread %d | retry %d | download cancelled", t.filePath, i+1, try))
						return
					default:
						n, err := resp.Body.Read(buffer)
						if n > 0 {
							if t.limiter != nil {
								<-t.limiter
							}

							if _, err := file.WriteAt(buffer[:n], start); err != nil {
								return
							}
							t.downloaded[i] += uint64(n)

							start += int64(n)
						}
						if err == io.EOF {
							slog.Debug(fmt.Sprintf("task %s | thread %d | retry %d | download finished!", t.filePath, i+1, try))
							done[i] = true
							return
						}
						if err != nil {
							return
						}
					}
				}
			}
		}(i)
	}

	ch := make(chan struct{})
	go func(ch chan struct{}) {
		for {
			select {
			case <-ch:
				return
			default:
				slog.Info(fmt.Sprintf("task %s | downloading %.2f%%...", t.filePath, float64(t.downloaded[0])/float64(t.fileSize)*100))
				time.Sleep(time.Second)
			}
		}
	}(ch)
	wg.Wait()
	ch <- struct{}{}

	for i := 0; i < int(t.threads); i++ {
		if !done[i] {
			slog.Error(fmt.Sprintf("task %s | thread %d failed", t.filePath, i+1))
			t.mutex.Lock()
			if t.status == InProgress {
				t.status = Failed
			}
			t.mutex.Unlock()
			return
		}
	}
	t.mutex.Lock()
	t.status = Completed
	t.mutex.Unlock()
	slog.Info(fmt.Sprintf("task %s | download finished!", t.filePath))
}

func (t *Task) Pause() {
	t.mutex.Lock()
	if t.status == InProgress {
		t.cancelFunc()
		t.status = Paused
		slog.Info(fmt.Sprintf("task %s | paused", t.filePath))
	}
	t.mutex.Unlock()
}

func (t *Task) Resume() {
	t.mutex.Lock()
	if t.status == Paused || t.status == Pending {
		if t.status == Pending {
			slog.Info("new task started!")
		} else {
			slog.Info(fmt.Sprintf("task %s | resumed", t.filePath))
		}
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
		slog.Info(fmt.Sprintf("task %s | canceled", t.filePath))
	}
	t.mutex.Unlock()
}

//func (t *Task)Retry()  {
//	//t.mutex.Loc
//	t.Cancel()
//	t.cancelFunc()
//	t.start()
//	//t.mutex.Unlock()
//
//}

func (t *Task) Url() string {
	return t.url
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
