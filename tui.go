package torrent

import (
	"fmt"
	"time"

	"github.com/JoelVCrasta/clover/download"
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	peers       int
	timeElapsed time.Duration
	progress    progress.Model
	percent     float64
}

type updateMsg struct {
	peers   int
	percent float64
}

type tickMsg struct{}

func initialModel() model {
	return model{
		progress: progress.New(progress.WithDefaultGradient()),
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

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m model) View() string {
	return fmt.Sprintf(
		"Peers: %d | Time Elapsed: %s\nProgress: %s\n",
		m.peers,
		m.timeElapsed.Truncate(time.Second),
		m.progress.ViewAs(m.percent),
	)
}

func StartTUI(statsFunc func() *download.Stats) {
	p := tea.NewProgram(initialModel())

	go func() {
		for {
			stats := statsFunc()
			percent := float64(stats.Done) / float64(stats.Total)
			p.Send(updateMsg{
				peers:   int(stats.PeerCount),
				percent: percent,
			})
			time.Sleep(1 * time.Second)
		}
	}()

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %s\n", err)
	}
}
