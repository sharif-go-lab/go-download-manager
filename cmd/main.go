package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	// Replace these with your actual local imports:
	"github.com/sharif-go-lab/go-download-manager/internal/queue"
	"github.com/sharif-go-lab/go-download-manager/internal/task"
)

// ----- Styles ----------------------------------------------------------------

var (
	tabBorder         = lipgloss.Border{Top: "─", Bottom: "─", Left: "│", Right: "│", TopLeft: "╭", TopRight: "╮", BottomLeft: "╰", BottomRight: "╯"}
	activeTabBorder   = lipgloss.Border{Top: "─", Bottom: " ", Left: "│", Right: "│", TopLeft: "╭", TopRight: "╮", BottomLeft: "│", BottomRight: "│"}
	docStyle          = lipgloss.NewStyle().Padding(1, 2)
	inactiveTabStyle  = lipgloss.NewStyle().Border(tabBorder, true).BorderForeground(lipgloss.Color("240")).Padding(0, 1)
	activeTabStyle    = lipgloss.NewStyle().Border(activeTabBorder, true).BorderForeground(lipgloss.Color("205")).Padding(0, 1)
	windowStyle       = lipgloss.NewStyle().Border(tabBorder).BorderForeground(lipgloss.Color("205")).Padding(2, 2)
	statusBarStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	helpStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	titleStyle        = lipgloss.NewStyle().Background(lipgloss.Color("205")).Foreground(lipgloss.Color("0")).Padding(0, 1)
	buttonStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Padding(0, 3)
	highlightColor    = lipgloss.Color("205")
)

// ----- Key Map ----------------------------------------------------------------

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
	Tab          key.Binding
	Help         key.Binding
	Quit         key.Binding
}

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
			key.WithKeys("d"),
			key.WithHelp("d", "remove/cancel"),
		),
		PauseResume: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "pause/resume"),
		),
		Retry: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "retry failed"),
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
			key.WithHelp("←", "prev tab"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next tab"),
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

func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Tab1, k.Tab2, k.Tab3, k.Help, k.Quit}
}

func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Tab1, k.Tab2, k.Tab3},
		{k.Up, k.Down, k.Left, k.Tab},
		{k.Enter, k.Escape},
		{k.Delete, k.PauseResume, k.Retry},
		{k.EditQueue, k.DeleteQueue, k.AddQueue},
		{k.Help, k.Quit},
	}
}

// ----- Queue Integration ------------------------------------------------------

// We'll maintain a single "default" queue that runs in the background.
var defaultQueue *queue.Queue

// tickMsg is a message used so we can refresh the UI periodically.
type tickMsg time.Time

// ----- Bubble Tea Model -------------------------------------------------------

type Model struct {
	tabs      []string
	activeTab int
	width     int
	height    int
	keys      KeyMap
	help      help.Model
	showHelp  bool

	// Tab 1: Add Download form fields
	urlInput      textinput.Model
	folderInput   textinput.Model
	filenameInput textinput.Model
	addFormFocus  int

	// Which download is selected in the Downloads List tab
	selectedDownload int

	// Tab 3: “Queues”
	queues        []QueueUI
	selectedQueue int

	// For editing queue settings
	editQueueMode       bool
	editQueueIndex      int
	queueEditFormFocus  int
	queueNameInput      textinput.Model
	queueFolderInput    textinput.Model
	queueMaxDlInput     textinput.Model
	queueSpeedInput     textinput.Model
	queueTimeInput      textinput.Model

	errorMsg string
}

// QueueUI is a lightweight struct for the UI. If you want multiple queue.Queue
// objects, you can store them similarly here and keep references to each queue.
type QueueUI struct {
	Name         string
	Folder       string
	MaxDownloads int
	SpeedLimit   string
	TimeWindow   string
}

// initialModel configures everything at startup.
func initialModel() Model {
	// Create the defaultQueue from the internal/queue code
	defaultQueue = queue.NewQueue(
		"Downloads",
		3,   // MaxDownloads
		2,   // Threads
		3,   // Retries
		0,   // speedLimit => 0 = "unlimited"
		nil, // no time interval
	)
	go defaultQueue.Run() // Start running it in the background

	urlInput := textinput.New()
	urlInput.Placeholder = "https://..."
	urlInput.Focus()

	folderInput := textinput.New()
	folderInput.Placeholder = "Select destination folder"

	filenameInput := textinput.New()
	filenameInput.Placeholder = "Output filename (optional)"

	// Prepare text inputs for editing queue settings
	queueNameInput := textinput.New()
	queueNameInput.Placeholder = "Queue Name"

	queueFolderInput := textinput.New()
	queueFolderInput.Placeholder = "Folder path"

	queueMaxDlInput := textinput.New()
	queueMaxDlInput.Placeholder = "Max Downloads"

	queueSpeedInput := textinput.New()
	queueSpeedInput.Placeholder = "Speed Limit"

	queueTimeInput := textinput.New()
	queueTimeInput.Placeholder = "Time Window"

	keys := DefaultKeyMap()
	helpModel := help.New()
	helpModel.ShowAll = false

	// Example local queues for the Tab 3 UI
	var queues []QueueUI
	queues = append(queues, QueueUI{
		Name:         "Default",
		Folder:       "Downloads",
		MaxDownloads: 3,
		SpeedLimit:   "Unlimited",
		TimeWindow:   "Always",
	})

	return Model{
		tabs:                []string{"Add Download", "Downloads List", "Queues List"},
		activeTab:           0,
		keys:                keys,
		help:                helpModel,
		urlInput:            urlInput,
		folderInput:         folderInput,
		filenameInput:       filenameInput,
		addFormFocus:        0,
		selectedDownload:    0,
		queues:              queues,
		selectedQueue:       0,
		editQueueMode:       false,
		editQueueIndex:      -1,
		queueEditFormFocus:  0,
		queueNameInput:      queueNameInput,
		queueFolderInput:    queueFolderInput,
		queueMaxDlInput:     queueMaxDlInput,
		queueSpeedInput:     queueSpeedInput,
		queueTimeInput:      queueTimeInput,
	}
}

// We schedule periodic ticks so we can update the UI with fresh progress info.
func (m Model) Init() tea.Cmd {
	return tea.Every(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case tickMsg:
		// On each tick, we just refresh the UI with new progress.
		return m, tea.Every(time.Second, func(t time.Time) tea.Msg {
			return tickMsg(t)
		})

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			defaultQueue.Stop()
			return m, tea.Quit

		case key.Matches(msg, m.keys.Help):
			m.showHelp = !m.showHelp

		case key.Matches(msg, m.keys.Tab1):
			m.activeTab = 0
			m.addFormFocus = 0
			m.urlInput.Focus()

		case key.Matches(msg, m.keys.Tab2):
			m.activeTab = 1

		case key.Matches(msg, m.keys.Tab3):
			m.activeTab = 2

		case key.Matches(msg, m.keys.Tab):
			// Basic next-tab
			m.activeTab++
			if m.activeTab >= len(m.tabs) {
				m.activeTab = 0
			}

		default:
			// If we are in editQueueMode, we handle that first (so it “overrides” normal tab inputs).
			if m.editQueueMode {
				m, cmd = m.updateEditQueueInputs(msg)
				cmds = append(cmds, cmd)
				return m, tea.Batch(cmds...)
			}

			// Otherwise, handle per-tab logic:
			switch m.activeTab {
			case 0:
				m, cmd = m.updateAddDownloadTab(msg)
				cmds = append(cmds, cmd)

			case 1:
				m = m.updateDownloadsListTab(msg)

			case 2:
				m = m.updateQueuesListTab(msg)
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width
	}

	return m, tea.Batch(cmds...)
}

// ----- Update Logic for each tab ---------------------------------------------

func (m Model) updateAddDownloadTab(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Up):
			m.addFormFocus = max(0, m.addFormFocus-1)

		case key.Matches(msg, m.keys.Down):
			m.addFormFocus = min(2, m.addFormFocus+1)

		case key.Matches(msg, m.keys.Enter):
			if m.addFormFocus < 2 {
				// Move to next input
				m.addFormFocus++
			} else {
				// On the third field (filename), press Enter => add the download
				if m.urlInput.Value() == "" {
					m.errorMsg = "URL is required."
				} else {
					defaultQueue.AddTask(m.urlInput.Value())

					// You could pass folder/filename to your queue if you extend the queue/task code.
					// For now, we just demonstrate adding by URL.

					// Reset
					m.urlInput.Reset()
					m.folderInput.Reset()
					m.filenameInput.Reset()
					m.addFormFocus = 0
					m.urlInput.Focus()

					// Switch to downloads tab to see it
					m.activeTab = 1
				}
			}

		case key.Matches(msg, m.keys.Escape):
			// Reset the form
			m.urlInput.Reset()
			m.folderInput.Reset()
			m.filenameInput.Reset()
			m.urlInput.Focus()
			m.addFormFocus = 0
		}
	}

	// Update whichever text field is focused
	switch m.addFormFocus {
	case 0:
		m.urlInput.Focus()
		m.folderInput.Blur()
		m.filenameInput.Blur()
		m.urlInput, cmd = m.urlInput.Update(msg)
	case 1:
		m.urlInput.Blur()
		m.folderInput.Focus()
		m.filenameInput.Blur()
		m.folderInput, cmd = m.folderInput.Update(msg)
	case 2:
		m.urlInput.Blur()
		m.folderInput.Blur()
		m.filenameInput.Focus()
		m.filenameInput, cmd = m.filenameInput.Update(msg)
	}

	return m, cmd
}

func (m Model) updateDownloadsListTab(msg tea.Msg) Model {
	tasks := defaultQueue.Tasks()

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Up):
			m.selectedDownload = max(0, m.selectedDownload-1)

		case key.Matches(msg, m.keys.Down):
			m.selectedDownload = min(len(tasks)-1, m.selectedDownload+1)

		case key.Matches(msg, m.keys.Delete):
			if len(tasks) > 0 && m.selectedDownload < len(tasks) {
				tasks[m.selectedDownload].Cancel()
			}

		case key.Matches(msg, m.keys.PauseResume):
			if len(tasks) > 0 && m.selectedDownload < len(tasks) {
				t := tasks[m.selectedDownload]
				switch t.Status() {
				case task.InProgress:
					t.Pause()
					break
				case task.Paused, task.Pending:
					t.Resume()
					break
				case task.Failed:
					// treat as a “retry”
					t.Resume()
					break
				}
			}

		case key.Matches(msg, m.keys.Retry):
			if len(tasks) > 0 && m.selectedDownload < len(tasks) {
				t := tasks[m.selectedDownload]
				if t.Status() == task.Failed {
					t.Resume()
				}
			}
		}
	}
	return m
}

func (m Model) updateQueuesListTab(msg tea.Msg) Model {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Up):
			m.selectedQueue = max(0, m.selectedQueue-1)

		case key.Matches(msg, m.keys.Down):
			m.selectedQueue = min(len(m.queues)-1, m.selectedQueue+1)

		case key.Matches(msg, m.keys.DeleteQueue):
			if len(m.queues) > 0 && m.selectedQueue < len(m.queues) {
				m.queues = append(m.queues[:m.selectedQueue], m.queues[m.selectedQueue+1:]...)
				if m.selectedQueue >= len(m.queues) {
					m.selectedQueue = max(0, len(m.queues)-1)
				}
			}

		case key.Matches(msg, m.keys.EditQueue):
			if len(m.queues) > 0 && m.selectedQueue < len(m.queues) {
				m.editQueueMode = true
				m.editQueueIndex = m.selectedQueue
				m.queueEditFormFocus = 0

				// Populate the text fields from the selected queue
				q := m.queues[m.selectedQueue]
				m.queueNameInput.SetValue(q.Name)
				m.queueFolderInput.SetValue(q.Folder)
				m.queueMaxDlInput.SetValue(fmt.Sprintf("%d", q.MaxDownloads))
				m.queueSpeedInput.SetValue(q.SpeedLimit)
				m.queueTimeInput.SetValue(q.TimeWindow)

				m.queueNameInput.Focus()
			}

		case key.Matches(msg, m.keys.AddQueue):
			newQ := QueueUI{
				Name:         "New Queue",
				Folder:       "/Downloads/New",
				MaxDownloads: 2,
				SpeedLimit:   "Unlimited",
				TimeWindow:   "Always",
			}
			m.queues = append(m.queues, newQ)
			m.selectedQueue = len(m.queues) - 1
		}
	}
	return m
}

// ----- Edit Queue Mode -------------------------------------------------------
// This handles the special “edit queue” form when the user has pressed E on Tab 3.

func (m Model) updateEditQueueInputs(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Up):
			m.queueEditFormFocus = max(0, m.queueEditFormFocus-1)

		case key.Matches(msg, m.keys.Down):
			m.queueEditFormFocus = min(4, m.queueEditFormFocus+1)

		case key.Matches(msg, m.keys.Enter):
			if m.queueEditFormFocus < 4 {
				// Move to next field
				m.queueEditFormFocus++
			} else {
				// On the last field, pressing Enter => Save changes
				updated := m.queues[m.editQueueIndex]

				updated.Name = m.queueNameInput.Value()
				updated.Folder = m.queueFolderInput.Value()

				// Safely parse the max downloads integer
				maxDl, err := strconv.Atoi(m.queueMaxDlInput.Value())
				if err != nil {
					maxDl = 1
				}
				updated.MaxDownloads = maxDl
				updated.SpeedLimit = m.queueSpeedInput.Value()
				updated.TimeWindow = m.queueTimeInput.Value()

				m.queues[m.editQueueIndex] = updated

				// If this is the “defaultQueue” (index 0), also update that queue in real time
				if m.editQueueIndex == 0 {
					// We only do MaxDownloads as an example. You can expand for speed/time if queue supports it.
					defaultQueue.MaxDownloads = uint8(maxDl)
				}

				// Exit edit mode
				m.editQueueMode = false
				m.editQueueIndex = -1
				m.queueEditFormFocus = 0
			}

		case key.Matches(msg, m.keys.Escape):
			// Cancel
			m.editQueueMode = false
			m.editQueueIndex = -1
			m.queueEditFormFocus = 0
		}
	}

	// Update whichever queue field is focused
	switch m.queueEditFormFocus {
	case 0:
		m.queueNameInput.Focus()
		m.queueFolderInput.Blur()
		m.queueMaxDlInput.Blur()
		m.queueSpeedInput.Blur()
		m.queueTimeInput.Blur()
		m.queueNameInput, cmd = m.queueNameInput.Update(msg)
	case 1:
		m.queueNameInput.Blur()
		m.queueFolderInput.Focus()
		m.queueMaxDlInput.Blur()
		m.queueSpeedInput.Blur()
		m.queueTimeInput.Blur()
		m.queueFolderInput, cmd = m.queueFolderInput.Update(msg)
	case 2:
		m.queueNameInput.Blur()
		m.queueFolderInput.Blur()
		m.queueMaxDlInput.Focus()
		m.queueSpeedInput.Blur()
		m.queueTimeInput.Blur()
		m.queueMaxDlInput, cmd = m.queueMaxDlInput.Update(msg)
	case 3:
		m.queueNameInput.Blur()
		m.queueFolderInput.Blur()
		m.queueMaxDlInput.Blur()
		m.queueSpeedInput.Focus()
		m.queueTimeInput.Blur()
		m.queueSpeedInput, cmd = m.queueSpeedInput.Update(msg)
	case 4:
		m.queueNameInput.Blur()
		m.queueFolderInput.Blur()
		m.queueMaxDlInput.Blur()
		m.queueSpeedInput.Blur()
		m.queueTimeInput.Focus()
		m.queueTimeInput, cmd = m.queueTimeInput.Update(msg)
	}

	return m, cmd
}

// ----- View Rendering ---------------------------------------------------------

func (m Model) View() string {
	doc := strings.Builder{}

	// Tabs row
	tabs := []string{}
	for i, tab := range m.tabs {
		if i == m.activeTab {
			tabs = append(tabs, activeTabStyle.Render(tab))
		} else {
			tabs = append(tabs, inactiveTabStyle.Render(tab))
		}
	}
	row := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)
	doc.WriteString(row + "\n\n")

	// Content area
	var content string
	if m.editQueueMode {
		content = m.renderEditQueueForm()
	} else {
		switch m.activeTab {
		case 0:
			content = m.renderAddDownloadTab()
		case 1:
			content = m.renderDownloadsListTab()
		case 2:
			content = m.renderQueuesListTab()
		}
	}

	windowContent := windowStyle.Width(m.width - 10).Render(content)
	doc.WriteString(windowContent + "\n\n")

	// Error message
	if m.errorMsg != "" {
		doc.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")).
			Render(m.errorMsg) + "\n\n")
	}

	// Help or status bar
	helpView := m.help.View(m.keys)
	if m.showHelp {
		doc.WriteString(helpStyle.Render(helpView))
	} else {
		statusBar := "F1:Add F2:Downloads F3:Queues | Press ? for help"
		doc.WriteString(statusBarStyle.Render(statusBar))
	}

	return docStyle.Render(doc.String())
}

// ----- Renders for each tab ---------------------------------------------------

func (m Model) renderAddDownloadTab() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(" Add New Download ") + "\n\n")

	// URL field
	urlLabel := "URL (required): "
	if m.addFormFocus == 0 {
		urlLabel = "> " + urlLabel
	} else {
		urlLabel = "  " + urlLabel
	}
	b.WriteString(urlLabel + m.urlInput.View() + "\n\n")

	// Folder field
	folderLabel := "Destination Folder: "
	if m.addFormFocus == 1 {
		folderLabel = "> " + folderLabel
	} else {
		folderLabel = "  " + folderLabel
	}
	b.WriteString(folderLabel + m.folderInput.View() + "\n\n")

	// Filename field
	filenameLabel := "Output Filename (optional): "
	if m.addFormFocus == 2 {
		filenameLabel = "> " + filenameLabel
	} else {
		filenameLabel = "  " + filenameLabel
	}
	b.WriteString(filenameLabel + m.filenameInput.View() + "\n\n")

	b.WriteString("Use Up/Down to move, Enter to proceed. Esc to reset/cancel.\n")
	return b.String()
}

func (m Model) renderDownloadsListTab() string {
	b := strings.Builder{}

	b.WriteString(titleStyle.Render(" Downloads List ") + "\n\n")
	b.WriteString(fmt.Sprintf("%-38s %-12s %-15s %s\n", "URL", "Status", "Progress", "Downloaded"))
	b.WriteString(strings.Repeat("─", 80) + "\n")

	tasks := defaultQueue.Tasks()
	for i, t := range tasks {
		prefix := "  "
		if i == m.selectedDownload {
			prefix = "> "
		}

		statusStr := statusToString(t.Status())
		totalSize := t.TotalSize()
		downloaded := t.Downloaded()
		progress := 0.0
		if totalSize > 0 {
			progress = float64(downloaded) / float64(totalSize)
		}

		line := fmt.Sprintf("%s%-38s %-12s %-15s %s",
			prefix,
			truncateString(t.Url(), 38),
			statusStr,
			renderProgressBar(progress, 15),
			fmt.Sprintf("%d/%d bytes", downloaded, totalSize),
		)

		if i == m.selectedDownload {
			line = lipgloss.NewStyle().Foreground(highlightColor).Render(line)
		}
		b.WriteString(line + "\n")
	}

	if len(tasks) == 0 {
		b.WriteString("\n  No downloads yet. Press F1 to add a download.\n")
	}
	b.WriteString("\nPress D to cancel/remove, P to pause/resume, R to retry\n")
	return b.String()
}

func (m Model) renderQueuesListTab() string {
	b := strings.Builder{}
	b.WriteString(titleStyle.Render(" Download Queues ") + "\n\n")

	b.WriteString(fmt.Sprintf("%-15s %-20s %-15s %-15s %-15s\n",
		"Name", "Folder", "MaxDls", "SpeedLimit", "TimeWindow"))
	b.WriteString(strings.Repeat("─", 80) + "\n")

	for i, q := range m.queues {
		prefix := "  "
		if i == m.selectedQueue {
			prefix = "> "
		}
		line := fmt.Sprintf(
			"%s%-15s %-20s %-15d %-15s %-15s",
			prefix,
			truncateString(q.Name, 15),
			truncateString(q.Folder, 20),
			q.MaxDownloads,
			q.SpeedLimit,
			q.TimeWindow,
		)
		if i == m.selectedQueue {
			line = lipgloss.NewStyle().Foreground(highlightColor).Render(line)
		}
		b.WriteString(line + "\n")
	}

	if len(m.queues) == 0 {
		b.WriteString("\n  No queues defined. Press N to add a new queue.\n")
	}
	b.WriteString("\nPress E to edit, D to delete, N to add a new queue\n")
	return b.String()
}

// ----- Render Edit Queue Form ------------------------------------------------

func (m Model) renderEditQueueForm() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(" Edit Queue ") + "\n\n")

	// Name
	nameLabel := "Name:"
	if m.queueEditFormFocus == 0 {
		nameLabel = "> " + nameLabel
	} else {
		nameLabel = "  " + nameLabel
	}
	b.WriteString(nameLabel + " " + m.queueNameInput.View() + "\n\n")

	// Folder
	folderLabel := "Folder:"
	if m.queueEditFormFocus == 1 {
		folderLabel = "> " + folderLabel
	} else {
		folderLabel = "  " + folderLabel
	}
	b.WriteString(folderLabel + " " + m.queueFolderInput.View() + "\n\n")

	// MaxDownloads
	maxLabel := "Max Downloads:"
	if m.queueEditFormFocus == 2 {
		maxLabel = "> " + maxLabel
	} else {
		maxLabel = "  " + maxLabel
	}
	b.WriteString(maxLabel + " " + m.queueMaxDlInput.View() + "\n\n")

	// SpeedLimit
	speedLabel := "Speed Limit:"
	if m.queueEditFormFocus == 3 {
		speedLabel = "> " + speedLabel
	} else {
		speedLabel = "  " + speedLabel
	}
	b.WriteString(speedLabel + " " + m.queueSpeedInput.View() + "\n\n")

	// TimeWindow
	timeLabel := "Time Window:"
	if m.queueEditFormFocus == 4 {
		timeLabel = "> " + timeLabel
	} else {
		timeLabel = "  " + timeLabel
	}
	b.WriteString(timeLabel + " " + m.queueTimeInput.View() + "\n\n")

	b.WriteString("Use Up/Down to move, Enter to save on the last field, Esc to cancel.\n")
	return b.String()
}

// ----- Helpers ----------------------------------------------------------------

func statusToString(s task.DownloadStatus) string {
	switch s {
	case task.Pending:
		return "Pending"
	case task.InProgress:
		return "downloading"
	case task.Paused:
		return "paused"
	case task.Completed:
		return "completed"
	case task.Canceled:
		return "canceled"
	case task.Failed:
		return "failed"
	default:
		return "unknown"
	}
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

func renderProgressBar(progress float64, width int) string {
	filled := int(progress * float64(width))
	if filled > width {
		filled = width
	}
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

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// ----- main -------------------------------------------------------------------

func main() {
	//level := new(slog.LevelVar)
	//level.Set(slog.LevelDebug)
	//logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
	//	Level: level,
	//}))
	//slog.SetDefault(logger)
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}

}
