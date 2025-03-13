package main

import (
	"github.com/sharif-go-lab/go-download-manager/internal/queue"
	"log/slog"
	"os"
)

func main() {
	//p := tea.NewProgram(tui.InitialModel())
	//if _, err := p.Run(); err != nil {
	//	fmt.Printf("Alas, there's been an error: %v", err)
	//	os.Exit(1)
	//}

	level := new(slog.LevelVar)
	level.Set(slog.LevelInfo)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	}))
	slog.SetDefault(logger)

	q := queue.NewQueue("", 2, 3, 0, 0, nil)
	q.AddTask("https://dl2.soft98.ir/soft/h/Hypersnap.9.5.3.x86.rar?1741892707")
	q.AddTask("https://dl2.soft98.ir/soft/h/Hypersnap.8.24.03.x64.rar?1741898862")
	q.AddTask("https://dl2.soft98.ir/soft/h/Hypersnap.8.24.03.x86.rar?1741898993")
	q.AddTask("https://dl2.soft98.ir/soft/h/Hypersnap.9.5.3.x64.rar?1741898896")
	q.Run()
}
