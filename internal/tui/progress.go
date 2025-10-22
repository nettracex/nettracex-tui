// Package tui contains animated progress components
package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nettracex/nettracex-tui/internal/domain"
)

// AnimatedProgress provides animated progress indicators
type AnimatedProgress struct {
	width     int
	height    int
	theme     domain.Theme
	message   string
	spinner   []string
	frame     int
	isActive  bool
	style     lipgloss.Style
}

// ProgressTickMsg is sent to update the progress animation
type ProgressTickMsg struct{}

// NewAnimatedProgress creates a new animated progress indicator
func NewAnimatedProgress() *AnimatedProgress {
	return &AnimatedProgress{
		spinner: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		frame:   0,
		isActive: false,
		style: lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Bold(true),
	}
}

// Start starts the progress animation
func (p *AnimatedProgress) Start(message string) tea.Cmd {
	p.message = message
	p.isActive = true
	p.frame = 0
	return p.tick()
}

// Stop stops the progress animation
func (p *AnimatedProgress) Stop() {
	p.isActive = false
}

// Update handles progress animation updates
func (p *AnimatedProgress) Update(msg tea.Msg) (*AnimatedProgress, tea.Cmd) {
	switch msg.(type) {
	case ProgressTickMsg:
		if p.isActive {
			p.frame = (p.frame + 1) % len(p.spinner)
			return p, p.tick()
		}
	}
	return p, nil
}

// View renders the progress indicator
func (p *AnimatedProgress) View() string {
	if !p.isActive {
		return ""
	}

	spinner := p.spinner[p.frame]
	content := fmt.Sprintf("%s %s", spinner, p.message)
	
	return p.style.Render(content)
}

// SetSize sets the progress indicator dimensions
func (p *AnimatedProgress) SetSize(width, height int) {
	p.width = width
	p.height = height
}

// SetTheme sets the progress indicator theme
func (p *AnimatedProgress) SetTheme(theme domain.Theme) {
	p.theme = theme
}

// SetMessage updates the progress message
func (p *AnimatedProgress) SetMessage(message string) {
	p.message = message
}

// IsActive returns whether the progress indicator is active
func (p *AnimatedProgress) IsActive() bool {
	return p.isActive
}

// tick returns a command to update the animation
func (p *AnimatedProgress) tick() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return ProgressTickMsg{}
	})
}

// ProgressBar provides a determinate progress bar
type ProgressBar struct {
	width     int
	height    int
	theme     domain.Theme
	current   int
	total     int
	message   string
	showPercentage bool
	style     lipgloss.Style
}

// NewProgressBar creates a new progress bar
func NewProgressBar() *ProgressBar {
	return &ProgressBar{
		showPercentage: true,
		style: lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")),
	}
}

// SetProgress sets the current progress
func (p *ProgressBar) SetProgress(current, total int) {
	p.current = current
	p.total = total
}

// SetMessage sets the progress message
func (p *ProgressBar) SetMessage(message string) {
	p.message = message
}

// View renders the progress bar
func (p *ProgressBar) View() string {
	if p.total == 0 {
		return p.message
	}

	// Calculate progress percentage
	percentage := float64(p.current) / float64(p.total)
	if percentage > 1.0 {
		percentage = 1.0
	}

	// Progress bar width (leave space for percentage and message)
	barWidth := p.width - 20
	if barWidth < 10 {
		barWidth = 10
	}

	// Calculate filled width
	filledWidth := int(float64(barWidth) * percentage)

	// Create progress bar
	filled := strings.Repeat("█", filledWidth)
	empty := strings.Repeat("░", barWidth-filledWidth)
	progressBar := filled + empty

	// Style the progress bar
	progressStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("39")).
		Background(lipgloss.Color("240"))

	styledBar := progressStyle.Render(progressBar)

	var content strings.Builder
	
	// Message
	if p.message != "" {
		messageStyle := lipgloss.NewStyle().Bold(true)
		content.WriteString(messageStyle.Render(p.message))
		content.WriteString("\n")
	}

	// Progress bar with percentage
	content.WriteString(styledBar)
	
	if p.showPercentage {
		percentText := fmt.Sprintf(" %3.0f%%", percentage*100)
		content.WriteString(percentText)
	}

	// Counter
	counterText := fmt.Sprintf(" (%d/%d)", p.current, p.total)
	content.WriteString(counterText)

	return content.String()
}

// SetSize sets the progress bar dimensions
func (p *ProgressBar) SetSize(width, height int) {
	p.width = width
	p.height = height
}

// SetTheme sets the progress bar theme
func (p *ProgressBar) SetTheme(theme domain.Theme) {
	p.theme = theme
}

// SetShowPercentage controls whether percentage is shown
func (p *ProgressBar) SetShowPercentage(show bool) {
	p.showPercentage = show
}

// IsComplete returns true if progress is complete
func (p *ProgressBar) IsComplete() bool {
	return p.current >= p.total && p.total > 0
}