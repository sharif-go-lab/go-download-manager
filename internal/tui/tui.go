package main

import (
	"fmt"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"os"
	"strings"
	//"github.com/charmbracelet/bubbles/list"
	//"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Styles
var (
	tabBorder         = lipgloss.Border{Top: "─", Bottom: "─", Left: "│", Right: "│", TopLeft: "╭", TopRight: "╮", BottomLeft: "╰", BottomRight: "╯"}
	activeTabBorder   = lipgloss.Border{Top: "─", Bottom: " ", Left: "│", Right: "│", TopLeft: "╭", TopRight: "╮", BottomLeft: "│", BottomRight: "│"}
	tabGap           = lipgloss.Border{Bottom: " "}
	docStyle         = lipgloss.NewStyle().Padding(1, 2)
	highlightColor   = lipgloss.Color("205")
	inactiveTabStyle = lipgloss.NewStyle().Border(tabBorder, true).BorderForeground(lipgloss.Color("240")).Padding(0, 1)
	activeTabStyle   = lipgloss.NewStyle().Border(activeTabBorder, true).BorderForeground(highlightColor).Padding(0, 1)
	windowStyle      = lipgloss.NewStyle().Border(tabBorder).BorderForeground(highlightColor).Padding(2, 2)
	statusBarStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	helpStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	titleStyle       = lipgloss.NewStyle().Background(highlightColor).Foreground(lipgloss.Color("0")).Padding(0, 1)
	buttonStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Padding(0, 3)
	activeButtonStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("0")).Background(lipgloss.Color("205")).Padding(0, 3)
)

// KeyMap defines a set of keybindings
type KeyMap struct {
	Tab1         key.Binding
	Tab2         key.Binding
	Tab3         key.Binding
	Enter        key.Binding
	Escape       key.Binding
	Delete       key.Binding
	PauseResume  key.Binding
	Retry        key.Binding
	EditQueue    key.Binding
	DeleteQueue  key.Binding
	AddQueue     key.Binding
	Up           key.Binding
	Down         key.Binding
	Left         key.Binding
	Right        key.Binding
	Help         key.Binding
	Quit         key.Binding
}

// DefaultKeyMap returns a set of keybindings
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Tab1: key.NewBinding(
			key.WithKeys("f1"),
			key.WithHelp("F1", "add download"),
		),
		Tab2: key.NewBinding(
			key.WithKeys("f2"),
			key.WithHelp("F2", "downloads"),
		),
		Tab3: key.NewBinding(
			key.WithKeys("f3"),
			key.WithHelp("F3", "queues"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "confirm"),
		),
		Escape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
		Delete: key.NewBinding(
			key.WithKeys("delete"),
			key.WithHelp("delete", "remove download"),
		),
		PauseResume: key.NewBinding(
			key.WithKeys("space"),
			key.WithHelp("space", "pause/resume"),
		),
		Retry: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "retry"),
		),
		EditQueue: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "edit queue"),
		),
		DeleteQueue: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete queue"),
		),
		AddQueue: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "new queue"),
		),
		Up: key.NewBinding(
			key.WithKeys("up"),
			key.WithHelp("↑", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down"),
			key.WithHelp("↓", "down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left"),
			key.WithHelp("←", "previous tab"),
		),
		Right: key.NewBinding(
			key.WithKeys("right"),
			key.WithHelp("→", "next tab"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "toggle help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c", "q"),
			key.WithHelp("ctrl+c/q", "quit"),
		),
	}
}

// Download represents a download item
type Download struct {
	URL      string
	Folder   string
	Status   string // "downloading", "paused", "failed", "completed"
	Progress float64
	Speed    string
}

// Queue represents a download queue
type Queue struct {
	Name        string
	Folder      string
	MaxDownloads int
	SpeedLimit  string
	TimeWindow  string
}

// ShortHelp returns keybindings to be shown in the mini help view
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Tab1, k.Tab2, k.Tab3, k.Help, k.Quit}
}

// FullHelp returns keybindings for the expanded help view
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Tab1, k.Tab2, k.Tab3},
		{k.Up, k.Down, k.Left, k.Right},
		{k.Enter, k.Escape},
		{k.Delete, k.PauseResume, k.Retry},
		{k.EditQueue, k.DeleteQueue, k.AddQueue},
		{k.Help, k.Quit},
	}
}

// Model represents the application state
type Model struct {
	tabs        []string
	activeTab   int
	width       int
	height      int
	keys        KeyMap
	help        help.Model
	showHelp    bool

	// Tab 1: Add Download
	urlInput        textinput.Model
	folderInput     textinput.Model
	filenameInput   textinput.Model
	addFormFocus    int

	// Tab 2: Downloads List
	downloads       []Download
	selectedDownload int

	// Tab 3: Queues List
	queues         []Queue
	selectedQueue  int

	// Shared
	errorMsg       string
}

func initialModel() Model {
	urlInput := textinput.New()
	urlInput.Placeholder = "https://..."
	urlInput.Focus()

	folderInput := textinput.New()
	folderInput.Placeholder = "Select destination folder"

	filenameInput := textinput.New()
	filenameInput.Placeholder = "Output filename (optional)"

	keys := DefaultKeyMap()
	helpModel := help.New()
	helpModel.ShowAll = false

	// Sample data
	var downloads []Download
	//downloads := []Download{
	//	{URL: "https://example.com/file1.zip", Folder: "/Downloads", Status: "downloading", Progress: 0.45, Speed: "1.2MB/s"},
	//	{URL: "https://example.com/file2.iso", Folder: "/Downloads/ISOs", Status: "paused", Progress: 0.78, Speed: "0KB/s"},
	//	{URL: "https://example.com/file3.exe", Folder: "/Applications", Status: "completed", Progress: 1.0, Speed: "0KB/s"},
	//	{URL: "https://example.com/file4.mp4", Folder: "/Videos", Status: "failed", Progress: 0.21, Speed: "0KB/s"},
	//}
	//
	var queues []Queue
	//queues := []Queue{
	//	{Name: "Default", Folder: "/Downloads", MaxDownloads: 3, SpeedLimit: "Unlimited", TimeWindow: "Always"},
	//	{Name: "Videos", Folder: "/Videos", MaxDownloads: 2, SpeedLimit: "5MB/s", TimeWindow: "22:00-08:00"},
	//	{Name: "Documents", Folder: "/Documents", MaxDownloads: 5, SpeedLimit: "10MB/s", TimeWindow: "Always"},
	//}

	return Model{
		tabs:            []string{"Add Download", "Downloads List", "Queues List"},
		activeTab:       0,
		keys:            keys,
		help:            helpModel,

		urlInput:        urlInput,
		folderInput:     folderInput,
		filenameInput:   filenameInput,
		addFormFocus:    0,

		downloads:       downloads,
		selectedDownload: 0,

		queues:          queues,
		selectedQueue:   0,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg. (type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.keys.Help):
			m.showHelp = !m.showHelp

		case key.Matches(msg, m.keys.Tab1):
			m.activeTab = 0
			m.urlInput.Focus()
			m.addFormFocus = 0

		case key.Matches(msg, m.keys.Tab2):
			m.activeTab = 1

		case key.Matches(msg, m.keys.Tab3):
			m.activeTab = 2

		case key.Matches(msg, m.keys.Left):
			m.activeTab = max(0, m.activeTab-1)
			if m.activeTab == 0 {
				m.urlInput.Focus()
				m.addFormFocus = 0
			}

		case key.Matches(msg, m.keys.Right):
			m.activeTab = min(len(m.tabs)-1, m.activeTab+1)

		default:
			// Handle tab-specific keys
			switch m.activeTab {
			case 0: // Add Download Tab
				switch {
				case key.Matches(msg, m.keys.Enter):
					if m.addFormFocus < 2 {
						m.addFormFocus++
						if m.addFormFocus == 1 {
							m.urlInput.Blur()
							m.folderInput.Focus()
						} else if m.addFormFocus == 2 {
							m.folderInput.Blur()
							m.filenameInput.Focus()
						}
					} else {
						// Submit the form - add a new download
						if m.urlInput.Value() != "" {
							newDownload := Download{
								URL:      m.urlInput.Value(),
								Folder:   m.folderInput.Value(),
								Status:   "downloading",
								Progress: 0.0,
								Speed:    "0KB/s",
							}
							m.downloads = append(m.downloads, newDownload)

							// Reset form
							m.urlInput.Reset()
							m.folderInput.Reset()
							m.filenameInput.Reset()
							m.urlInput.Focus()
							m.addFormFocus = 0
							m.activeTab = 1 // Switch to Downloads tab
						} else {
							m.errorMsg = "URL is required"
						}
					}

				case key.Matches(msg, m.keys.Escape):
					// Reset form
					m.urlInput.Reset()
					m.folderInput.Reset()
					m.filenameInput.Reset()
					m.urlInput.Focus()
					m.addFormFocus = 0
				}

				// Handle input updates
				switch m.addFormFocus {
				case 0:
					m.urlInput, cmd = m.urlInput.Update(msg)
					cmds = append(cmds, cmd)
				case 1:
					m.folderInput, cmd = m.folderInput.Update(msg)
					cmds = append(cmds, cmd)
				case 2:
					m.filenameInput, cmd = m.filenameInput.Update(msg)
					cmds = append(cmds, cmd)
				}

			case 1: // Downloads List Tab
				switch {
				case key.Matches(msg, m.keys.Up):
					m.selectedDownload = max(0, m.selectedDownload-1)

				case key.Matches(msg, m.keys.Down):
					m.selectedDownload = min(len(m.downloads)-1, m.selectedDownload)

				case key.Matches(msg, m.keys.Delete):
					if len(m.downloads) > 0 && m.selectedDownload < len(m.downloads) {
						// Remove the selected download
						m.downloads = append(
							m.downloads[:m.selectedDownload],
							m.downloads[m.selectedDownload+1:]...,
						)
						if m.selectedDownload >= len(m.downloads) {
							m.selectedDownload = max(0, len(m.downloads)-1)
						}
					}

				case key.Matches(msg, m.keys.PauseResume):
					if len(m.downloads) > 0 && m.selectedDownload < len(m.downloads) {
						// Toggle download status
						if m.downloads[m.selectedDownload].Status == "downloading" {
							m.downloads[m.selectedDownload].Status = "paused"
							m.downloads[m.selectedDownload].Speed = "0KB/s"
						} else if m.downloads[m.selectedDownload].Status == "paused" ||
							m.downloads[m.selectedDownload].Status == "failed" {
							m.downloads[m.selectedDownload].Status = "downloading"
							m.downloads[m.selectedDownload].Speed = "1.2MB/s"
						}
					}

				case key.Matches(msg, m.keys.Retry):
					if len(m.downloads) > 0 && m.selectedDownload < len(m.downloads) {
						if m.downloads[m.selectedDownload].Status == "failed" {
							m.downloads[m.selectedDownload].Status = "downloading"
							m.downloads[m.selectedDownload].Speed = "1.2MB/s"
						}
					}
				}

			case 2: // Queues List Tab
				switch {
				case key.Matches(msg, m.keys.Up):
					m.selectedQueue = max(0, m.selectedQueue-1)

				case key.Matches(msg, m.keys.Down):
					m.selectedQueue = min(len(m.queues)-1, m.selectedQueue)

				case key.Matches(msg, m.keys.DeleteQueue):
					if len(m.queues) > 0 && m.selectedQueue < len(m.queues) {
						// Remove the selected queue
						m.queues = append(
							m.queues[:m.selectedQueue],
							m.queues[m.selectedQueue+1:]...,
						)
						if m.selectedQueue >= len(m.queues) {
							m.selectedQueue = max(0, len(m.queues)-1)
						}
					}

				case key.Matches(msg, m.keys.EditQueue):
					// In a real app, this would open a form to edit the queue
					if len(m.queues) > 0 && m.selectedQueue < len(m.queues) {
						m.errorMsg = "Editing queue: " + m.queues[m.selectedQueue].Name
					}

				case key.Matches(msg, m.keys.AddQueue):
					// In a real app, this would open a form to add a new queue
					newQueue := Queue{
						Name:        "New Queue",
						Folder:      "/Downloads/New",
						MaxDownloads: 2,
						SpeedLimit:  "Unlimited",
						TimeWindow:  "Always",
					}
					m.queues = append(m.queues, newQueue)
					m.selectedQueue = len(m.queues) - 1
				}
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	doc := strings.Builder{}

	// Tabs row
	tabs := []string{}
	for i, tab := range m.tabs {
		var t string
		if i == m.activeTab {
			t = activeTabStyle.Render(tab)
		} else {
			t = inactiveTabStyle.Render(tab)
		}
		tabs = append(tabs, t)
	}
	row := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
	gap := "\n"
	doc.WriteString(row + "\n" + gap + "\n")

	// Content area
	content := ""
	switch m.activeTab {
	case 0:
		content = m.renderAddDownloadTab()
	case 1:
		content = m.renderDownloadsListTab()
	case 2:
		content = m.renderQueuesListTab()
	}

	windowContent := windowStyle.Width(m.width - 10).Render(content)
	doc.WriteString(windowContent + "\n\n")

	// Error message
	if m.errorMsg != "" {
		doc.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(m.errorMsg) + "\n\n")
	}

	// Help
	helpView := m.help.View(m.keys)
	if m.showHelp {
		doc.WriteString(helpStyle.Render(helpView))
	} else {
		// Footer/status bar
		statusBar := fmt.Sprintf("F1:Add F2:Downloads F3:Queues | Press ? for help")
		doc.WriteString(statusBarStyle.Render(statusBar))
	}

	return docStyle.Render(doc.String())
}

func (m Model) renderAddDownloadTab() string {
	content := strings.Builder{}

	// Title
	content.WriteString(titleStyle.Render(" Add New Download ") + "\n\n")

	// URL field
	urlLabel := "URL (required): "
	if m.addFormFocus == 0 {
		urlLabel = "> " + urlLabel
	} else {
		urlLabel = "  " + urlLabel
	}
	content.WriteString(urlLabel + m.urlInput.View() + "\n\n")

	// Folder field
	folderLabel := "Destination Folder: "
	if m.addFormFocus == 1 {
		folderLabel = "> " + folderLabel
	} else {
		folderLabel = "  " + folderLabel
	}
	content.WriteString(folderLabel + m.folderInput.View() + "\n\n")

	// Filename field
	filenameLabel := "Output Filename (optional): "
	if m.addFormFocus == 2 {
		filenameLabel = "> " + filenameLabel
	} else {
		filenameLabel = "  " + filenameLabel
	}
	content.WriteString(filenameLabel + m.filenameInput.View() + "\n\n")

	// Buttons
	okButton := buttonStyle.Render("[ OK ]")
	cancelButton := buttonStyle.Render("[ Cancel ]")
	content.WriteString("\n" + okButton + "  " + cancelButton + "\n")
	content.WriteString("\nPress Enter to move between fields and submit, Esc to cancel")

	return content.String()
}

// renderProgressBar renders a simple ASCII progress bar
func renderProgressBar(progress float64, width int) string {
	filled := int(progress * float64(width))
	bar := strings.Builder{}
	bar.WriteString("[")
	for i := 0; i < width; i++ {
		if i < filled {
			bar.WriteString("=")
		} else {
			bar.WriteString(" ")
		}
	}
	bar.WriteString("]")
	return bar.String()
}

func (m Model) renderDownloadsListTab() string {
	content := strings.Builder{}

	// Title
	content.WriteString(titleStyle.Render(" Downloads List ") + "\n\n")

	// Table header
	content.WriteString(fmt.Sprintf("%-40s %-20s %-15s %-25s\n", "URL", "Folder", "Status", "Progress/Speed"))
	content.WriteString(strings.Repeat("─", 100) + "\n")

	// Table rows
	for i, dl := range m.downloads {
		prefix := "  "
		if i == m.selectedDownload {
			prefix = "> "
		}

		status := dl.Status
		if dl.Status == "downloading" {
			status = fmt.Sprintf("%-15s %s %.1f%% %s", status, renderProgressBar(dl.Progress, 15), dl.Progress*100, dl.Speed)
		}

		line := fmt.Sprintf("%s%-40s %-20s %-15s %-25s",
			prefix,
			truncateString(dl.URL, 40),
			truncateString(dl.Folder, 20),
			dl.Status,
			status)

		if i == m.selectedDownload {
			line = lipgloss.NewStyle().Foreground(highlightColor).Render(line)
		}

		content.WriteString(line + "\n")
	}

	if len(m.downloads) == 0 {
		content.WriteString("\n  No downloads yet. Press F1 to add a download.\n")
	}

	// Controls help
	content.WriteString("\n")
	content.WriteString("Press Delete to remove, Space to pause/resume, R to retry failed downloads\n")

	return content.String()
}

func (m Model) renderQueuesListTab() string {
	content := strings.Builder{}

	// Title
	content.WriteString(titleStyle.Render(" Download Queues ") + "\n\n")

	// Table header
	content.WriteString(fmt.Sprintf("%-15s %-20s %-15s %-15s %-15s\n",
		"Name", "Folder", "Max Downloads", "Speed Limit", "Time Window"))
	content.WriteString(strings.Repeat("─", 80) + "\n")

	// Table rows
	for i, q := range m.queues {
		prefix := "  "
		if i == m.selectedQueue {
			prefix = "> "
		}

		line := fmt.Sprintf("%s%-15s %-20s %-15d %-15s %-15s",
			prefix,
			truncateString(q.Name, 15),
			truncateString(q.Folder, 20),
			q.MaxDownloads,
			q.SpeedLimit,
			q.TimeWindow)

		if i == m.selectedQueue {
			line = lipgloss.NewStyle().Foreground(highlightColor).Render(line)
		}

		content.WriteString(line + "\n")
	}

	if len(m.queues) == 0 {
		content.WriteString("\n  No queues defined. Press N to add a new queue.\n")
	}

	// Controls help
	content.WriteString("\n")
	content.WriteString("Press E to edit queue, D to delete queue, N to add new queue\n")

	return content.String()
}

// Utility functions
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}