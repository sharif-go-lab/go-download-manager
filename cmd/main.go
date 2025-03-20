package main

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	// Your real local imports:
	"github.com/sharif-go-lab/go-download-manager/internal/queue"
	"github.com/sharif-go-lab/go-download-manager/internal/task"
)

// -----------------------------------------------------------------------------
// Styles
// -----------------------------------------------------------------------------

var (
	tabBorder        = lipgloss.Border{Top: "─", Bottom: "─", Left: "│", Right: "│", TopLeft: "╭", TopRight: "╮", BottomLeft: "╰", BottomRight: "╯"}
	activeTabBorder  = lipgloss.Border{Top: "─", Bottom: " ", Left: "│", Right: "│", TopLeft: "╭", TopRight: "╮", BottomLeft: "│", BottomRight: "│"}
	docStyle         = lipgloss.NewStyle().Padding(1, 2)
	inactiveTabStyle = lipgloss.NewStyle().Border(tabBorder, true).BorderForeground(lipgloss.Color("240")).Padding(0, 1)
	activeTabStyle   = lipgloss.NewStyle().Border(activeTabBorder, true).BorderForeground(lipgloss.Color("205")).Padding(0, 1)
	windowStyle      = lipgloss.NewStyle().Border(tabBorder).BorderForeground(lipgloss.Color("205")).Padding(2, 2)
	statusBarStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	helpStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	titleStyle       = lipgloss.NewStyle().Background(lipgloss.Color("205")).Foreground(lipgloss.Color("0")).Padding(0, 1)
	highlightColor   = lipgloss.Color("205")
)

type queuedTask struct {
	queue *queue.Queue
	task      *task.Task
}

func (m Model) getAllDownloads() []queuedTask {
	var result []queuedTask
	for _, rq := range m.realQueues {
		for _, t := range rq.Tasks() {
			result = append(result, queuedTask{
				queue: rq, // or rq.Name()
				task:      t,
			})
		}
	}
	return result
}

func formatSpeed(bps uint64) string {
	// bps = bytes per second
	if bps < 1024 {
		return fmt.Sprintf("%d B/s", bps)
	} else if bps < 1024*1024 {
		return fmt.Sprintf("%.1f KB/s", float64(bps)/1024.0)
	} else if bps < 1024*1024*1024 {
		return fmt.Sprintf("%.1f MB/s", float64(bps)/(1024.0*1024.0))
	}
	return fmt.Sprintf("%.1f GB/s", float64(bps)/(1024.0*1024.0*1024.0))
}

func (m *Model) updateSpeeds() {
	// Build a list of all tasks from all queues:
	for _, rq := range m.realQueues {
		for _, t := range rq.Tasks() {
			current := t.Downloaded()
			prev := m.prevDownloaded[t]
			diff := current - prev // bytes downloaded in last second

			m.speeds[t] = diff // bytes/sec
			m.prevDownloaded[t] = current
		}
	}
}

//--------------------------------------------
// -----------------------------------------------------------------------------
// Key Map
// -----------------------------------------------------------------------------

type KeyMap struct {
	Tab1        key.Binding
	Tab2        key.Binding
	Tab3        key.Binding
	Enter       key.Binding
	Escape      key.Binding
	Delete      key.Binding
	PauseResume key.Binding
	Retry       key.Binding
	EditQueue   key.Binding
	DeleteQueue key.Binding
	AddQueue    key.Binding
	Up          key.Binding
	Down        key.Binding
	Left        key.Binding
	Right       key.Binding
	Tab         key.Binding
	Help        key.Binding
	Quit        key.Binding
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
			key.WithHelp("←", "left"),
		),
		Right: key.NewBinding(
			key.WithKeys("right"),
			key.WithHelp("→", "right"),
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
		{k.Up, k.Down, k.Left, k.Right, k.Tab},
		{k.Enter, k.Escape},
		{k.Delete, k.PauseResume, k.Retry},
		{k.EditQueue, k.DeleteQueue, k.AddQueue},
		{k.Help, k.Quit},
	}
}

// -----------------------------------------------------------------------------
// tickMsg for periodic refresh
// -----------------------------------------------------------------------------

type tickMsg time.Time

// -----------------------------------------------------------------------------
// Model
// -----------------------------------------------------------------------------

type Model struct {
	width     int
	height    int
	keys      KeyMap
	help      help.Model
	showHelp  bool
	activeTab int

	// We'll keep a slice of real queues:
	realQueues    []*queue.Queue
	selectedQueue int // which queue is selected in the Queues List
	mu            sync.Mutex

	// Tab 1: Add Download
	urlInput         textinput.Model
	folderInput      textinput.Model
	filenameInput    textinput.Model
	addFormFocus     int  // 0=URL,1=Queue selection,2=Folder,3=Filename
	selectedQForAdd  int  // which queue is chosen for the new download
	creatingDownload bool // not strictly needed, but a simple state marker

	// Tab 2: Downloads
	selectedDownload int // index into the combined tasks of all queues

	// Tab 3: "Queues UI" (lightweight info for each queue)
	queues           []QueueUI
	editQueueMode    bool
	editQueueIndex   int
	queueEditFocus   int
	queueNameInput   textinput.Model
	queueFolderInput textinput.Model
	queueMaxDlInput  textinput.Model
	queueSpeedInput  textinput.Model
	queueTimeInput   textinput.Model
	// For speed tracking:
	prevDownloaded map[*task.Task]uint64 // how many bytes were downloaded at last tick
	speeds         map[*task.Task]uint64 // current speed in bytes/s for each task
	errorMsg       string
}

// QueueUI is a minimal struct that parallels the real queues in `m.realQueues`.
type QueueUI struct {
	Name         string
	Folder       string
	MaxDownloads int
	SpeedLimit   uint64
	TimeWindow   string
}

// -----------------------------------------------------------------------------
// init Model
// -----------------------------------------------------------------------------

func initialModel() Model {
	// Create some example queues for demonstration
	q1 := queue.NewQueue("Default", "~/Downloads", 3, 2, 3, 0, nil)
	//q1.SetDirectory("~/Downloads")
	go q1.Run()

	//q2 := queue.NewQueue("Videos", "~/Documents", 2, 3, 2, 0, nil)
	//q2.SetDirectory("VideosFolder")
	//go q2.Run()

	// If you have any more, add them here:
	// q3 := ...
	// go q3.Run()

	realQueues := []*queue.Queue{q1}

	// Build a parallel UI slice:
	queuesUI := []QueueUI{
		{
			Name:         q1.Name,
			Folder:       q1.Directory,
			MaxDownloads: int(q1.MaxDownloads),
			SpeedLimit:   q1.SpeedLimit,
			TimeWindow:   "Always",
		},
		//{
		//	Name:         q2.Name,
		//	Folder:       q2.Directory,
		//	MaxDownloads: int(q2.MaxDownloads),
		//	SpeedLimit:   q2.SpeedLimit,
		//	TimeWindow:   "Always",
		//},
	}

	// Prepare text inputs
	urlInput := textinput.New()
	urlInput.Placeholder = "https://example.com/file.zip"
	urlInput.Focus()

	folderInput := textinput.New()
	folderInput.Placeholder = "(Optional) Save to folder"

	filenameInput := textinput.New()
	filenameInput.Placeholder = "(Optional) Custom filename"

	// For editing queue settings
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

	return Model{
		keys:       keys,
		help:       helpModel,
		showHelp:   false,
		activeTab:  0,
		realQueues: realQueues,
		queues:     queuesUI,
		mu:         sync.Mutex{},

		// Tab 1 (Add)
		urlInput:      urlInput,
		folderInput:   folderInput,
		filenameInput: filenameInput,
		addFormFocus:  0,
		// selectedQForAdd = 0 means queue #0 is chosen by default

		// Tab 2 (Downloads)
		selectedDownload: 0,

		// Tab 3 (Queues)
		selectedQueue:    0,
		editQueueMode:    false,
		editQueueIndex:   -1,
		queueEditFocus:   0,
		queueNameInput:   queueNameInput,
		queueFolderInput: queueFolderInput,
		queueMaxDlInput:  queueMaxDlInput,
		queueSpeedInput:  queueSpeedInput,
		queueTimeInput:   queueTimeInput,
		prevDownloaded: make(map[*task.Task]uint64),
		speeds:         make(map[*task.Task]uint64),
	}
}

func main() {
	// Example logger at error-level
	level := new(slog.LevelVar)
	level.Set(slog.LevelError)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	}))
	slog.SetDefault(logger)

	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}

// -----------------------------------------------------------------------------
// Tea Lifecycle
// -----------------------------------------------------------------------------

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
		// refresh the UI
		m.updateSpeeds()
		return m, tea.Every(time.Second, func(t time.Time) tea.Msg {
			return tickMsg(t)
		})

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width
		return m, nil

	case tea.KeyMsg:
		m.errorMsg = ""
		switch {
		case key.Matches(msg, m.keys.Quit):
			// Stop all real queues
			for _, rq := range m.realQueues {
				rq.Stop()
			}
			return m, tea.Quit

		case key.Matches(msg, m.keys.Help):
			m.showHelp = !m.showHelp

		// Tabs by function keys
		case key.Matches(msg, m.keys.Tab1):
			m.activeTab = 0
			m.addFormFocus = 0
			m.urlInput.Focus()
		case key.Matches(msg, m.keys.Tab2):
			m.activeTab = 1
		case key.Matches(msg, m.keys.Tab3):
			m.activeTab = 2

		// "Tab" to cycle tabs
		case key.Matches(msg, m.keys.Tab):
			m.activeTab++
			if m.activeTab >= 3 {
				m.activeTab = 0
			}
		}

		// If we're editing a queue (Tab 3 / E) we handle that first
		if m.editQueueMode {
			m, cmd = m.updateEditQueue(msg)
			cmds = append(cmds, cmd)
			return m, tea.Batch(cmds...)
		}

		// Otherwise handle per-tab logic
		switch m.activeTab {
		case 0:
			m, cmd = m.updateTabAdd(msg)
			cmds = append(cmds, cmd)

		case 1:
			m = m.updateTabDownloads(msg)

		case 2:
			m = m.updateTabQueues(msg)
		}

	}
	return m, tea.Batch(cmds...)
}

// -----------------------------------------------------------------------------
// Update logic: Tab 0 (Add Download)
// -----------------------------------------------------------------------------

func (m Model) updateTabAdd(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Up):
			m.addFormFocus = max(0, m.addFormFocus-1)

		case key.Matches(msg, m.keys.Down):
			m.addFormFocus = min(2, m.addFormFocus+1)

		case key.Matches(msg, m.keys.Left):
			// If we are on the queue selection row, pressing left changes the queue
			if m.addFormFocus == 1 && len(m.realQueues) > 1 {
				m.selectedQForAdd--
				if m.selectedQForAdd < 0 {
					m.selectedQForAdd = len(m.realQueues) - 1
				}
			}
		case key.Matches(msg, m.keys.Right):
			// If we are on the queue selection row, pressing right changes the queue
			if m.addFormFocus == 1 && len(m.realQueues) > 1 {
				m.selectedQForAdd++
				if m.selectedQForAdd >= len(m.realQueues) {
					m.selectedQForAdd = 0
				}
			}

		case key.Matches(msg, m.keys.Enter):
			// If not yet at last field, move forward
			if m.addFormFocus < 2 {
				m.addFormFocus++
			} else {
				// On last field => attempt to add
				if m.urlInput.Value() == "" {
					m.errorMsg = "URL is required"
				} else {
					// Add to whichever queue is selected
					if m.selectedQForAdd < len(m.realQueues) {
						chosenQ := m.realQueues[m.selectedQForAdd]
						chosenQ.AddTask(m.urlInput.Value(), m.folderInput.Value())
						// If your queue supports specifying folder/filename, pass them too:
						// chosenQ.AddTaskWithDetails(m.urlInput.Value(), m.folderInput.Value(), m.filenameInput.Value())
					}

					// Reset
					m.urlInput.Reset()
					m.folderInput.Reset()
					m.filenameInput.Reset()
					m.urlInput.Focus()
					m.addFormFocus = 0
					m.selectedQForAdd = 0

					// Switch to downloads tab
					m.activeTab = 1
				}
			}

		case key.Matches(msg, m.keys.Escape):
			// reset
			m.urlInput.Reset()
			m.folderInput.Reset()
			m.filenameInput.Reset()
			m.urlInput.Focus()
			m.addFormFocus = 0
			m.selectedQForAdd = 0
		}
	}

	// Update whichever textinput is in focus
	switch m.addFormFocus {
	case 0:
		m.urlInput.Focus()
		m.folderInput.Blur()
		m.filenameInput.Blur()
		m.urlInput, cmd = m.urlInput.Update(msg)
	case 1:
		// The "queue selection" row is not a textinput,
		// so we only handle left/right keys above.
		m.urlInput.Blur()
		m.folderInput.Blur()
		m.filenameInput.Blur()
	case 2:
		m.urlInput.Blur()
		m.folderInput.Focus()
		m.filenameInput.Blur()
		m.folderInput, cmd = m.folderInput.Update(msg)
		//case 3:
		//	m.urlInput.Blur()
		//	m.folderInput.Blur()
		//	m.filenameInput.Focus()
		//	m.filenameInput, cmd = m.filenameInput.Update(msg)
	}
	return m, cmd
}

// -----------------------------------------------------------------------------
// Update logic: Tab 1 (Downloads)
// -----------------------------------------------------------------------------

func (m Model) updateTabDownloads(msg tea.Msg) Model {
	// We gather tasks from all queues and let the user pick one by index
	//var allTasks []*task.Task
	//for _, rq := range m.realQueues {
	//	allTasks = append(allTasks, rq.Tasks()...)
	//}
	allTasks := m.getAllDownloads()
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Up):
			m.selectedDownload = max(0, m.selectedDownload-1)
		case key.Matches(msg, m.keys.Down):
			m.selectedDownload = min(len(allTasks)-1, m.selectedDownload+1)

		case key.Matches(msg, m.keys.Delete):
			if len(allTasks) > 0 && m.selectedDownload < len(allTasks) {
				allTasks[m.selectedDownload].task.Cancel()
			}

		case key.Matches(msg, m.keys.PauseResume):
			if len(allTasks) > 0 && m.selectedDownload < len(allTasks) {
				t := allTasks[m.selectedDownload]
				switch t.task.Status() {
				case task.InProgress:
					t.task.Pause()
				case task.Paused, task.Pending:
					t.task.Resume()
				case task.Failed:
					// treat as a “retry”
					//t.task.Cancel()
					t.queue.AddTask(t.task.Url(),t.task.DirectoryPath)


				}
			}

		case key.Matches(msg, m.keys.Retry):
			if len(allTasks) > 0 && m.selectedDownload < len(allTasks) {
				t := allTasks[m.selectedDownload]
				//if t.task.Status() == task.Failed {
				t.task.Cancel()
				//t.task.Resume()
				t.queue.AddTask(t.task.Url(),t.task.DirectoryPath)
				//}
			}
		}
	}
	return m
}

// -----------------------------------------------------------------------------
// Update logic: Tab 2 (Queues)
// -----------------------------------------------------------------------------

func (m Model) updateTabQueues(msg tea.Msg) Model {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Up):
			m.selectedQueue = max(0, m.selectedQueue-1)
		case key.Matches(msg, m.keys.Down):
			m.selectedQueue = min(len(m.queues)-1, m.selectedQueue+1)

		case key.Matches(msg, m.keys.DeleteQueue):
			if m.selectedQueue < len(m.realQueues) {
				// Stop & remove from realQueues
				toRemove := m.realQueues[m.selectedQueue]
				toRemove.Stop()

				m.realQueues = append(m.realQueues[:m.selectedQueue],
					m.realQueues[m.selectedQueue+1:]...)

				// Remove from the UI slice
				m.queues = append(m.queues[:m.selectedQueue],
					m.queues[m.selectedQueue+1:]...)

				if m.selectedQueue >= len(m.queues) {
					m.selectedQueue = max(0, len(m.queues)-1)
				}
			}

		case key.Matches(msg, m.keys.EditQueue):
			if m.selectedQueue < len(m.queues) {
				m.editQueueMode = true
				m.editQueueIndex = m.selectedQueue
				m.queueEditFocus = 0

				qUI := m.queues[m.selectedQueue]
				m.queueNameInput.SetValue(qUI.Name)
				m.queueFolderInput.SetValue(qUI.Folder)
				m.queueMaxDlInput.SetValue(fmt.Sprintf("%d", qUI.MaxDownloads))
				m.queueSpeedInput.SetValue(fmt.Sprintf("%d", qUI.SpeedLimit))
				m.queueTimeInput.SetValue(qUI.TimeWindow)
			}

		case key.Matches(msg, m.keys.AddQueue):
			// Create a brand new real queue
			newRealQ := queue.NewQueue("NewQueue", "Downloads", 2, 2, 3, 0, nil)
			//newRealQ.SetDirectory("~/Downloads")
			go newRealQ.Run()

			m.realQueues = append(m.realQueues, newRealQ)

			// Also add to UI
			m.queues = append(m.queues, QueueUI{
				Name:         newRealQ.Name,
				Folder:       newRealQ.Directory,
				MaxDownloads: int(newRealQ.MaxDownloads),
				SpeedLimit:   newRealQ.SpeedLimit,
				TimeWindow:   "Always",
			})
			m.selectedQueue = len(m.queues) - 1
		}
	}
	return m
}

// -----------------------------------------------------------------------------
// Update logic: Editing a queue
// -----------------------------------------------------------------------------

func (m Model) updateEditQueue(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Up):
			m.queueEditFocus = max(0, m.queueEditFocus-1)
		case key.Matches(msg, m.keys.Down):
			m.queueEditFocus = min(4, m.queueEditFocus+1)

		case key.Matches(msg, m.keys.Enter):
			if m.queueEditFocus < 4 {
				m.queueEditFocus++
			} else {
				// Save changes
				if m.editQueueIndex >= 0 && m.editQueueIndex < len(m.realQueues) {
					rQ := m.realQueues[m.editQueueIndex]
					qUI := m.queues[m.editQueueIndex]

					// name
					newName := m.queueNameInput.Value()
					rQ.SetName(newName)
					qUI.Name = newName

					// folder
					folder := m.queueFolderInput.Value()
					err := rQ.SetDirectory(folder)
					if err != nil {
						m.errorMsg = err.Error()
					} else {
						qUI.Folder = folder
					}
					// max downloads
					maxStr := m.queueMaxDlInput.Value()
					maxDl, err := strconv.Atoi(maxStr)
					if err == nil && maxDl > 0 {
						rQ.SetMaxDownloads(uint8(maxDl))
						qUI.MaxDownloads = maxDl
					}

					// speed limit
					speedStr := m.queueSpeedInput.Value()
					speed, err2 := strconv.ParseInt(speedStr, 10, 64)
					if err2 == nil && speed >= 0 {
						rQ.SetSpeedLimit(uint64(speed))
						qUI.SpeedLimit = uint64(speed)
					}

					// time window
					timeWindowStr := m.queueTimeInput.Value()
					// Attempt to parse & store in the real queue
					err3 := rQ.SetActiveIntervalFromString(timeWindowStr)
					if err3 != nil {
						m.errorMsg = "Invalid time window: " + err3.Error()
					} else {
						qUI.TimeWindow = timeWindowStr
					}
					// For demonstration only
					qUI.TimeWindow = m.queueTimeInput.Value()

					// Save back to the UI slice
					m.queues[m.editQueueIndex] = qUI
				}

				// Exit edit mode
				m.editQueueMode = false
				m.editQueueIndex = -1
				m.queueEditFocus = 0
			}

		case key.Matches(msg, m.keys.Escape):
			m.editQueueMode = false
			m.editQueueIndex = -1
			m.queueEditFocus = 0
		}
	}

	// Update whichever text input is in focus
	switch m.queueEditFocus {
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

// -----------------------------------------------------------------------------
// Views
// -----------------------------------------------------------------------------

func (m Model) View() string {
	var doc strings.Builder

	// Tabs
	var tabs []string
	tabTitles := []string{"Add Download", "Downloads", "Queues"}
	for i, t := range tabTitles {
		if i == m.activeTab {
			tabs = append(tabs, activeTabStyle.Render(t))
		} else {
			tabs = append(tabs, inactiveTabStyle.Render(t))
		}
	}
	doc.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, tabs...) + "\n\n")

	var content string
	if m.editQueueMode {
		content = m.viewEditQueueForm()
	} else {
		switch m.activeTab {
		case 0:
			content = m.viewTabAdd()
		case 1:
			content = m.viewTabDownloads()
		case 2:
			content = m.viewTabQueues()
		}
	}

	windowContent := windowStyle.Width(m.width - 10).Render(content)
	doc.WriteString(windowContent + "\n\n")

	// Error
	if m.errorMsg != "" {
		doc.WriteString(
			lipgloss.NewStyle().Foreground(lipgloss.Color("9")).
				Render(m.errorMsg) + "\n\n")
	}

	// Show help or quick status
	helpView := m.help.View(m.keys)
	if m.showHelp {
		doc.WriteString(helpStyle.Render(helpView))
	} else {
		doc.WriteString(statusBarStyle.Render("F1:Add  F2:Downloads  F3:Queues  ?=Help"))
	}

	return docStyle.Render(doc.String())
}

// -----------------------------------------------------------------------------
// View: Tab 0 (Add)
// -----------------------------------------------------------------------------

func (m Model) viewTabAdd() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(" Add New Download ") + "\n\n")

	// 0) URL
	urlLabel := "URL (required): "
	if m.addFormFocus == 0 {
		urlLabel = "> " + urlLabel
	} else {
		urlLabel = "  " + urlLabel
	}
	b.WriteString(urlLabel + m.urlInput.View() + "\n\n")

	// 1) Queue selection
	queueLabel := "Select Queue: "
	if m.addFormFocus == 1 {
		queueLabel = "> " + queueLabel
	} else {
		queueLabel = "  " + queueLabel
	}
	chosenQueueName := ""
	if m.selectedQForAdd < len(m.realQueues) {
		chosenQueueName = m.realQueues[m.selectedQForAdd].Name
	}
	b.WriteString(fmt.Sprintf("%s[ %s ]  (←/→ to change)\n\n", queueLabel, chosenQueueName))

	// 2) Folder
	folderLabel := "Folder (optional): "
	if m.addFormFocus == 2 {
		folderLabel = "> " + folderLabel
	} else {
		folderLabel = "  " + folderLabel
	}
	b.WriteString(folderLabel + m.folderInput.View() + "\n\n")

	// 3) Filename
	//fileLabel := "Filename (optional): "
	//if m.addFormFocus == 3 {
	//	fileLabel = "> " + fileLabel
	//} else {
	//	fileLabel = "  " + fileLabel
	//}
	//b.WriteString(fileLabel + m.filenameInput.View() + "\n\n")

	b.WriteString("Up/Down to navigate fields, Enter to proceed, Esc to cancel.\n")
	return b.String()
}

// -----------------------------------------------------------------------------
// View: Tab 1 (Downloads)
// -----------------------------------------------------------------------------


//  Render “Queue” AND “Speed” in your downloads tab:
func (m Model) viewTabDownloads() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(" Downloads ") + "\n\n")

	allTasks := m.getAllDownloads() // queuedTask objects

	// We have “Queue” + “URL” + “Status” + “Progress” + “Speed” + “Downloaded”
	b.WriteString(fmt.Sprintf(
		"%-10s %-36s %-12s %-15s %-10s %s\n",
		"Queue",    // 10 chars wide
		"URL",      // 36 chars wide
		"Status",   // 12 chars
		"Progress", // 15 chars
		"Speed",    // 10 chars
		"Downloaded",
	))
	b.WriteString(strings.Repeat("─", 100) + "\n")

	for i, item := range allTasks {
		prefix := "  "
		if i == m.selectedDownload {
			prefix = "> "
		}
		t := item.task

		queueName := truncateString(item.queue.Name, 10)
		urlStr    := truncateString(t.Url(), 36)
		statusStr := statusToString(t.Status())

		total      := t.TotalSize()
		downloaded := t.Downloaded()
		progress   := 0.0
		if total > 0 {
			progress = float64(downloaded) / float64(total)
		}

		// speed is from your m.speeds map:
		speedBps := m.speeds[t]
		speedStr := formatSpeed(speedBps) // "KB/s" etc.

		line := fmt.Sprintf("%s%-10s %-36s %-12s %-15s %-10s %s",
			prefix,
			queueName,                  // queue column
			urlStr,                     // url column
			statusStr,                  // status
			renderProgressBar(progress, 15),
			speedStr,                   // speed column
			fmt.Sprintf("%d/%d bytes", downloaded, total),
		)
		if i == m.selectedDownload {
			line = lipgloss.NewStyle().Foreground(highlightColor).Render(line)
		}

		b.WriteString(line + "\n")
	}

	if len(allTasks) == 0 {
		b.WriteString("\nNo tasks. Press F1 to add.\n")
	}
	b.WriteString("\nD=Cancel, P=Pause/Resume, R=Retry failed\n")
	return b.String()
}

// -----------------------------------------------------------------------------
// View: Tab 2 (Queues)
// -----------------------------------------------------------------------------

func (m Model) viewTabQueues() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(" Queues ") + "\n\n")

	b.WriteString(fmt.Sprintf("%-15s %-20s %-12s %-10s %-12s\n",
		"Name", "Folder", "MaxDls", "Speed", "TimeWindow"))
	b.WriteString(strings.Repeat("─", 80) + "\n")

	for i, q := range m.queues {
		prefix := "  "
		if i == m.selectedQueue {
			prefix = "> "
		}
		line := fmt.Sprintf("%s%-15s %-20s %-12d %-10d %-12s",
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
		b.WriteString("\nNo queues. Press N to add.\n")
	} else {
		b.WriteString("\nUp/Down=Select queue, E=Edit, D=Delete, N=Add\n")
	}
	return b.String()
}

// -----------------------------------------------------------------------------
// View: Edit Queue Form
// -----------------------------------------------------------------------------

func (m Model) viewEditQueueForm() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(" Edit Queue ") + "\n\n")

	label := "Name:"
	if m.queueEditFocus == 0 {
		label = "> " + label
	} else {
		label = "  " + label
	}
	b.WriteString(label + " " + m.queueNameInput.View() + "\n\n")

	label = "Folder:"
	if m.queueEditFocus == 1 {
		label = "> " + label
	} else {
		label = "  " + label
	}
	b.WriteString(label + " " + m.queueFolderInput.View() + "\n\n")

	label = "Max Downloads:"
	if m.queueEditFocus == 2 {
		label = "> " + label
	} else {
		label = "  " + label
	}
	b.WriteString(label + " " + m.queueMaxDlInput.View() + "\n\n")

	label = "Speed Limit (KB/s):"
	if m.queueEditFocus == 3 {
		label = "> " + label
	} else {
		label = "  " + label
	}
	b.WriteString(label + " " + m.queueSpeedInput.View() + "\n\n")

	// Here’s the updated “Time Window” label, clarifying the format
	label = "Time Window (HH:MM:SS-HH:MM:SS):"
	if m.queueEditFocus == 4 {
		label = "> " + label
	} else {
		label = "  " + label
	}
	b.WriteString(label + " " + m.queueTimeInput.View() + "\n\n")

	// Some quick instructions
	b.WriteString("Example: 08:00:00-17:00:00 for an 8am-5pm window\n")
	b.WriteString("Up/Down to navigate, Enter to save on last field, Esc=Cancel.\n")
	return b.String()
}

// -----------------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------------

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func statusToString(s task.DownloadStatus) string {
	switch s {
	case task.Pending:
		return "Pending"
	case task.InProgress:
		return "Downloading"
	case task.Paused:
		return "Paused"
	case task.Completed:
		return "Completed"
	case task.Canceled:
		return "Canceled"
	case task.Failed:
		return "Failed"
	default:
		return "Unknown"
	}
}

func renderProgressBar(progress float64, width int) string {
	filled := int(progress * float64(width))
	if filled > width {
		filled = width
	}
	var sb strings.Builder
	sb.WriteString("[")
	for i := 0; i < width; i++ {
		if i < filled {
			sb.WriteString("=")
		} else {
			sb.WriteString(" ")
		}
	}
	sb.WriteString("]")
	return sb.String()
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
