// ignore this file for now

package torrent

import (
	"context"
	"fmt"
	"time"

	"github.com/JoelVCrasta/clover/download"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	peers       int
	timeElapsed time.Duration
	progress    progress.Model
	percent     float64
	done        int
	total       int
	selected    bool // whether the quit button is selected
	cancel      context.CancelFunc
}

type updateMsg struct {
	peers   int
	percent float64
	done    int
	total   int
	cancel  context.CancelFunc
}

type tickMsg struct{}

// Styles
var (
	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(1, 1)

	activeButton = lipgloss.NewStyle().
			Foreground(lipgloss.Color("0")). // black text
			Background(lipgloss.Color("9")). // red background
			Bold(true).
			Padding(0, 3).
			MarginTop(1)

	inactiveButton = lipgloss.NewStyle().
			Foreground(lipgloss.Color("7")). // gray text
			Background(lipgloss.Color("0")). // black background
			Padding(0, 3).
			MarginTop(1)
)

func initialModel() model {
	return model{
		progress: progress.New(progress.WithDefaultGradient()),
		selected: false,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg { return tickMsg{} })
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		m.timeElapsed += time.Second
		return m, tea.Tick(time.Second, func(time.Time) tea.Msg { return tickMsg{} })

	case updateMsg:
		um := msg
		m.peers = um.peers
		m.percent = um.percent
		m.done = um.done
		m.total = um.total
		m.cancel = um.cancel

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.cancel()
			return m, tea.Quit

		case "tab":
			m.selected = !m.selected

		case "enter":
			if m.selected {
				m.cancel()
				return m, tea.Quit
			}
		}
	}

	return m, nil
}

func (m model) View() string {
	// Download stats
	content := fmt.Sprintf(
		"Peers connected: %d | Time Elapsed: %s\n"+
			"Pieces done: %d / %d\n\n"+
			"Progress: %s\n",
		m.peers,
		m.timeElapsed.Truncate(time.Second),
		m.done,
		m.total,
		m.progress.ViewAs(m.percent),
	)

	// Button style
	var quitButton string
	if m.selected {
		quitButton = activeButton.Render("[ Quit ]")
	} else {
		quitButton = inactiveButton.Render("[ Quit ]")
	}

	return boxStyle.Render(content + "\n" + quitButton)
}

func StartTUI(dm *download.DownloadManager) {
	cancel := dm.CancelDownload()
	m := initialModel()
	m.cancel = cancel
	p := tea.NewProgram(m)
 
     go func() {
         for {
             stats := dm.Stats()
             if stats.Total == 0 {
				time.Sleep(200 * time.Millisecond)
				continue
             }
             percent := float64(stats.Done) / float64(stats.Total)
             p.Send(updateMsg{
                 peers:   int(stats.PeerCount),
                 percent: percent,
                 done:    stats.Done,
                 total:   stats.Total,
                 cancel:  cancel,
             })
             time.Sleep(1 * time.Second)
         }
     }()
 
     if _, err := p.Run(); err != nil {
         fmt.Printf("Error running program: %s\n", err)
     }
 }
