package main

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sharif-go-lab/go-download-manager/internal/tui"
	"os"
)

func main() {
	p := tea.NewProgram(tui.InitialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
}
