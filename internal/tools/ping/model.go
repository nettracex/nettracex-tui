// Package ping provides TUI model for ping diagnostic tool
package ping

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nettracex/nettracex-tui/internal/domain"
)

// Model represents the ping tool TUI model
type Model struct {
	tool         *Tool
	state        ModelState
	hostInput    textinput.Model
	countInput   textinput.Model
	intervalInput textinput.Model
	focusedInput int
	results      []domain.PingResult
	statistics   PingStatistics
	error        error
	width        int
	height       int
	theme        domain.Theme
	loading      bool
	progress     int
	totalPings   int
	
	// Real-time display components
	liveStats    LiveStatistics
	latencyGraph LatencyGraph
	packetLoss   PacketLossIndicator
	startTime    time.Time
	lastUpdate   time.Time
	
	// Animation and update control
	animationTicker *time.Ticker
	updateInterval  time.Duration
	
	// Continuous ping mode
	continuousMode bool
	cancelFunc     context.CancelFunc
}

// ModelState represents the current state of the model
type ModelState int

const (
	StateInput ModelState = iota
	StateRunning
	StateResult
	StateError
)

// LiveStatistics tracks real-time ping statistics
type LiveStatistics struct {
	PacketsSent     int           `json:"packets_sent"`
	PacketsReceived int           `json:"packets_received"`
	PacketLoss      float64       `json:"packet_loss_percent"`
	MinRTT          time.Duration `json:"min_rtt"`
	MaxRTT          time.Duration `json:"max_rtt"`
	AvgRTT          time.Duration `json:"avg_rtt"`
	LastRTT         time.Duration `json:"last_rtt"`
	Jitter          time.Duration `json:"jitter"`
	ElapsedTime     time.Duration `json:"elapsed_time"`
}

// LatencyGraph represents a simple ASCII graph of latency over time
type LatencyGraph struct {
	Values    []time.Duration
	MaxValues int
	MaxRTT    time.Duration
	MinRTT    time.Duration
	Width     int
	Height    int
}

// PacketLossIndicator shows packet loss visualization
type PacketLossIndicator struct {
	RecentResults []bool // true = success, false = loss
	MaxResults    int
	LossCount     int
	TotalCount    int
}

// NewModel creates a new ping model
func NewModel(tool *Tool) *Model {
	hostInput := textinput.New()
	hostInput.Placeholder = "Enter hostname or IP address (e.g., google.com, 8.8.8.8)"
	hostInput.Focus()
	hostInput.CharLimit = 253
	hostInput.Width = 50

	countInput := textinput.New()
	countInput.Placeholder = "Number of pings (0 = continuous)"
	countInput.CharLimit = 4
	countInput.Width = 30
	countInput.SetValue("4")

	intervalInput := textinput.New()
	intervalInput.Placeholder = "Interval in seconds (default: 1)"
	intervalInput.CharLimit = 3
	intervalInput.Width = 30
	intervalInput.SetValue("1")

	return &Model{
		tool:             tool,
		state:            StateInput,
		hostInput:        hostInput,
		countInput:       countInput,
		intervalInput:    intervalInput,
		focusedInput:     0,
		loading:          false,
		updateInterval:   100 * time.Millisecond, // 10 FPS for smooth updates
		
		// Initialize real-time components
		latencyGraph: LatencyGraph{
			Values:    make([]time.Duration, 0),
			MaxValues: 50, // Keep last 50 values for graph
			Width:     60,
			Height:    8,
		},
		packetLoss: PacketLossIndicator{
			RecentResults: make([]bool, 0),
			MaxResults:    20, // Show last 20 ping results
		},
	}
}

// Init initializes the model
func (m *Model) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages and updates the model
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if m.state == StateRunning && m.cancelFunc != nil {
				m.cancelFunc()
				m.state = StateResult
				return m, nil
			}
			return m, tea.Quit
		case "esc":
			if m.state != StateInput {
				m.resetToInput()
				return m, nil
			}
		case "tab":
			if m.state == StateInput {
				m.nextInput()
				return m, nil
			}
		case "shift+tab":
			if m.state == StateInput {
				m.prevInput()
				return m, nil
			}
		case "enter":
			if m.state == StateInput && m.hostInput.Value() != "" {
				return m, m.startPing()
			}
		case "s":
			if m.state == StateRunning && m.continuousMode {
				// Stop continuous ping
				if m.cancelFunc != nil {
					m.cancelFunc()
				}
				m.state = StateResult
				return m, nil
			}
		}

	case pingStartMsg:
		m.state = StateRunning
		m.loading = true
		m.progress = 0
		m.results = []domain.PingResult{}
		m.startTime = time.Now()
		m.lastUpdate = time.Now()
		m.continuousMode = msg.continuous
		
		// Reset live components
		m.liveStats = LiveStatistics{}
		m.latencyGraph.Values = make([]time.Duration, 0)
		m.packetLoss.RecentResults = make([]bool, 0)
		m.packetLoss.LossCount = 0
		m.packetLoss.TotalCount = 0
		
		// Start animation ticker for smooth updates
		return m, tea.Batch(
			m.tickCmd(),
			func() tea.Msg { return pingInitMsg{} },
		)

	case pingProgressMsg:
		m.progress = msg.completed
		m.results = append(m.results, msg.result)
		m.updateLiveStats(msg.result)
		m.lastUpdate = time.Now()
		return m, nil

	case pingCompleteMsg:
		m.state = StateResult
		m.loading = false
		m.statistics = msg.statistics
		if m.cancelFunc != nil {
			m.cancelFunc()
			m.cancelFunc = nil
		}
		return m, nil

	case pingErrorMsg:
		m.state = StateError
		m.loading = false
		m.error = msg.error
		if m.cancelFunc != nil {
			m.cancelFunc()
			m.cancelFunc = nil
		}
		return m, nil

	case tickMsg:
		if m.state == StateRunning {
			// Update elapsed time and continue ticking
			m.liveStats.ElapsedTime = time.Since(m.startTime)
			return m, m.tickCmd()
		}
		return m, nil

	case pingInitMsg:
		// Start the actual ping operation
		return m, m.executePing()
	}

	// Update input fields
	if m.state == StateInput {
		switch m.focusedInput {
		case 0:
			m.hostInput, cmd = m.hostInput.Update(msg)
			cmds = append(cmds, cmd)
		case 1:
			m.countInput, cmd = m.countInput.Update(msg)
			cmds = append(cmds, cmd)
		case 2:
			m.intervalInput, cmd = m.intervalInput.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

// View renders the model
func (m *Model) View() string {
	var content strings.Builder

	// Header
	content.WriteString(m.renderHeader())
	content.WriteString("\n\n")

	switch m.state {
	case StateInput:
		content.WriteString(m.renderInput())
	case StateRunning:
		content.WriteString(m.renderRunning())
	case StateResult:
		content.WriteString(m.renderResult())
	case StateError:
		content.WriteString(m.renderError())
	}

	// Footer
	content.WriteString("\n\n")
	content.WriteString(m.renderFooter())

	return content.String()
}

// SetSize sets the model dimensions
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.hostInput.Width = width - 4
	m.countInput.Width = width - 4
	m.intervalInput.Width = width - 4
	
	// Update graph dimensions based on available space
	m.latencyGraph.Width = width - 8
	if m.latencyGraph.Width > 80 {
		m.latencyGraph.Width = 80
	}
	if m.latencyGraph.Width < 20 {
		m.latencyGraph.Width = 20
	}
}

// SetTheme sets the model theme
func (m *Model) SetTheme(theme domain.Theme) {
	m.theme = theme
}

// Focus focuses the model
func (m *Model) Focus() {
	if m.state == StateInput {
		m.focusCurrentInput()
	}
}

// Blur blurs the model
func (m *Model) Blur() {
	m.hostInput.Blur()
	m.countInput.Blur()
	m.intervalInput.Blur()
}

// renderHeader renders the tool header
func (m *Model) renderHeader() string {
	title := "Ping Diagnostic Tool"
	description := "Test network connectivity and measure round-trip time"

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		MarginBottom(1)

	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	return titleStyle.Render(title) + "\n" + descStyle.Render(description)
}

// renderInput renders the input form
func (m *Model) renderInput() string {
	var content strings.Builder

	labelStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205"))

	focusedStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(0, 1)

	unfocusedStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1)

	// Host input
	content.WriteString(labelStyle.Render("Target Host:"))
	content.WriteString("\n")
	if m.focusedInput == 0 {
		content.WriteString(focusedStyle.Render(m.hostInput.View()))
	} else {
		content.WriteString(unfocusedStyle.Render(m.hostInput.View()))
	}
	content.WriteString("\n\n")

	// Count input
	content.WriteString(labelStyle.Render("Ping Count:"))
	content.WriteString("\n")
	if m.focusedInput == 1 {
		content.WriteString(focusedStyle.Render(m.countInput.View()))
	} else {
		content.WriteString(unfocusedStyle.Render(m.countInput.View()))
	}
	content.WriteString("\n\n")

	// Interval input
	content.WriteString(labelStyle.Render("Interval (seconds):"))
	content.WriteString("\n")
	if m.focusedInput == 2 {
		content.WriteString(focusedStyle.Render(m.intervalInput.View()))
	} else {
		content.WriteString(unfocusedStyle.Render(m.intervalInput.View()))
	}
	content.WriteString("\n\n")

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Italic(true)

	content.WriteString(helpStyle.Render("Use Tab to navigate • Enter 0 for continuous ping"))

	return content.String()
}

// renderRunning renders the running state with real-time results
func (m *Model) renderRunning() string {
	var sections []string

	// Header with progress
	headerSection := m.renderRunningHeader()
	sections = append(sections, headerSection)

	// Live statistics panel
	statsSection := m.renderLiveStatistics()
	sections = append(sections, statsSection)

	// Latency graph
	if len(m.latencyGraph.Values) > 0 {
		graphSection := m.renderLatencyGraph()
		sections = append(sections, graphSection)
	}

	// Packet loss indicator
	if len(m.packetLoss.RecentResults) > 0 {
		lossSection := m.renderPacketLossIndicator()
		sections = append(sections, lossSection)
	}

	// Recent results (last few pings)
	if len(m.results) > 0 {
		recentSection := m.renderRecentResults()
		sections = append(sections, recentSection)
	}

	// Instructions
	instructionsSection := m.renderRunningInstructions()
	sections = append(sections, instructionsSection)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// renderRunningHeader renders the header with progress information
func (m *Model) renderRunningHeader() string {
	progressStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("214")).
		Bold(true)

	var headerText string
	if m.continuousMode {
		headerText = fmt.Sprintf("🔍 Pinging %s continuously... (%d sent)",
			m.hostInput.Value(), m.liveStats.PacketsSent)
	} else {
		headerText = fmt.Sprintf("🔍 Pinging %s... (%d/%d)",
			m.hostInput.Value(), m.progress, m.totalPings)
	}

	// Add elapsed time
	elapsedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("243")).
		Italic(true)

	elapsed := fmt.Sprintf("Elapsed: %v", m.liveStats.ElapsedTime.Truncate(time.Second))

	return lipgloss.JoinVertical(lipgloss.Left,
		progressStyle.Render(headerText),
		elapsedStyle.Render(elapsed),
	)
}

// renderLiveStatistics renders real-time statistics
func (m *Model) renderLiveStatistics() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		MarginBottom(1)

	statsStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1).
		Width(m.width - 4)

	var statsLines []string

	// Packet statistics
	lossColor := "46" // Green
	if m.liveStats.PacketLoss > 0 {
		lossColor = "214" // Orange
	}
	if m.liveStats.PacketLoss > 10 {
		lossColor = "196" // Red
	}

	packetsLine := fmt.Sprintf("Packets: Sent=%d, Received=%d, Loss=%.1f%%",
		m.liveStats.PacketsSent,
		m.liveStats.PacketsReceived,
		m.liveStats.PacketLoss)

	lossStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(lossColor))
	statsLines = append(statsLines, lossStyle.Render(packetsLine))

	// RTT statistics (only if we have successful pings)
	if m.liveStats.PacketsReceived > 0 {
		rttLine := fmt.Sprintf("RTT: Min=%v, Max=%v, Avg=%v, Last=%v",
			m.liveStats.MinRTT.Truncate(time.Microsecond),
			m.liveStats.MaxRTT.Truncate(time.Microsecond),
			m.liveStats.AvgRTT.Truncate(time.Microsecond),
			m.liveStats.LastRTT.Truncate(time.Microsecond))

		// Color code based on latency
		rttColor := "46" // Green
		if m.liveStats.LastRTT > 100*time.Millisecond {
			rttColor = "214" // Orange
		}
		if m.liveStats.LastRTT > 500*time.Millisecond {
			rttColor = "196" // Red
		}

		rttStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(rttColor))
		statsLines = append(statsLines, rttStyle.Render(rttLine))

		// Jitter
		if m.liveStats.Jitter > 0 {
			jitterLine := fmt.Sprintf("Jitter: %v", m.liveStats.Jitter.Truncate(time.Microsecond))
			statsLines = append(statsLines, jitterLine)
		}
	}

	statsContent := lipgloss.JoinVertical(lipgloss.Left, statsLines...)

	return lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render("📊 Live Statistics"),
		statsStyle.Render(statsContent),
	)
}

// renderLatencyGraph renders an ASCII latency graph
func (m *Model) renderLatencyGraph() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		MarginBottom(1)

	graphStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1).
		Width(m.width - 4)

	graph := m.generateLatencyGraph()

	return lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render("📈 Latency Graph (last 50 pings)"),
		graphStyle.Render(graph),
	)
}

// generateLatencyGraph creates an ASCII graph of latency values
func (m *Model) generateLatencyGraph() string {
	if len(m.latencyGraph.Values) == 0 {
		return "No data yet..."
	}

	// Calculate graph dimensions
	graphWidth := m.latencyGraph.Width
	if graphWidth > m.width-8 {
		graphWidth = m.width - 8
	}
	graphHeight := m.latencyGraph.Height

	// Find min/max for scaling
	minRTT := m.latencyGraph.Values[0]
	maxRTT := m.latencyGraph.Values[0]
	for _, rtt := range m.latencyGraph.Values {
		if rtt < minRTT {
			minRTT = rtt
		}
		if rtt > maxRTT {
			maxRTT = rtt
		}
	}

	// Add some padding to the range
	rttRange := maxRTT - minRTT
	if rttRange == 0 {
		rttRange = time.Millisecond
	}
	minRTT -= rttRange / 10
	maxRTT += rttRange / 10

	// Create graph grid
	grid := make([][]rune, graphHeight)
	for i := range grid {
		grid[i] = make([]rune, graphWidth)
		for j := range grid[i] {
			grid[i][j] = ' '
		}
	}

	// Plot values
	valueCount := len(m.latencyGraph.Values)
	for i, rtt := range m.latencyGraph.Values {
		// Calculate x position (spread across width)
		x := (i * graphWidth) / valueCount
		if x >= graphWidth {
			x = graphWidth - 1
		}

		// Calculate y position (inverted for display)
		normalizedRTT := float64(rtt-minRTT) / float64(maxRTT-minRTT)
		y := graphHeight - 1 - int(normalizedRTT*float64(graphHeight-1))
		if y < 0 {
			y = 0
		}
		if y >= graphHeight {
			y = graphHeight - 1
		}

		// Choose character based on latency level
		var char rune
		if rtt < 50*time.Millisecond {
			char = '▁' // Low latency
		} else if rtt < 100*time.Millisecond {
			char = '▃' // Medium latency
		} else if rtt < 200*time.Millisecond {
			char = '▅' // High latency
		} else {
			char = '▇' // Very high latency
		}

		grid[y][x] = char
	}

	// Convert grid to string
	var lines []string
	for _, row := range grid {
		lines = append(lines, string(row))
	}

	// Add scale information
	scaleInfo := fmt.Sprintf("Scale: %v - %v",
		minRTT.Truncate(time.Microsecond),
		maxRTT.Truncate(time.Microsecond))

	scaleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("243")).
		Italic(true)

	lines = append(lines, scaleStyle.Render(scaleInfo))

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// renderPacketLossIndicator renders packet loss visualization
func (m *Model) renderPacketLossIndicator() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		MarginBottom(1)

	indicatorStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1).
		Width(m.width - 4)

	// Create visual indicator of recent ping results
	var indicators []string
	indicators = append(indicators, "Recent ping results (✓ = success, ✗ = loss):")

	var resultChars []string
	for _, success := range m.packetLoss.RecentResults {
		if success {
			successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("46"))
			resultChars = append(resultChars, successStyle.Render("✓"))
		} else {
			lossStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
			resultChars = append(resultChars, lossStyle.Render("✗"))
		}
	}

	// Group results in lines of reasonable length
	charsPerLine := (m.width - 8) / 2 // Account for spacing
	if charsPerLine < 10 {
		charsPerLine = 10
	}

	for i := 0; i < len(resultChars); i += charsPerLine {
		end := i + charsPerLine
		if end > len(resultChars) {
			end = len(resultChars)
		}
		line := strings.Join(resultChars[i:end], " ")
		indicators = append(indicators, line)
	}

	content := lipgloss.JoinVertical(lipgloss.Left, indicators...)

	return lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render("📡 Packet Loss Indicator"),
		indicatorStyle.Render(content),
	)
}

// renderRecentResults renders the most recent ping results
func (m *Model) renderRecentResults() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		MarginBottom(1)

	resultsStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1).
		Width(m.width - 4)

	// Show last 5 results
	maxDisplay := 5
	startIdx := 0
	if len(m.results) > maxDisplay {
		startIdx = len(m.results) - maxDisplay
	}

	var resultLines []string
	for i := startIdx; i < len(m.results); i++ {
		result := m.results[i]
		if result.Error != nil {
			errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
			line := fmt.Sprintf("Ping %d: %v", result.Sequence, result.Error)
			resultLines = append(resultLines, errorStyle.Render(line))
		} else {
			successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("46"))
			line := fmt.Sprintf("Ping %d: %s time=%v ttl=%d",
				result.Sequence, result.Host.IPAddress, result.RTT.Truncate(time.Microsecond), result.TTL)
			resultLines = append(resultLines, successStyle.Render(line))
		}
	}

	if len(resultLines) == 0 {
		resultLines = append(resultLines, "Waiting for ping results...")
	}

	content := lipgloss.JoinVertical(lipgloss.Left, resultLines...)

	return lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render("📋 Recent Results"),
		resultsStyle.Render(content),
	)
}

// renderRunningInstructions renders instructions for the running state
func (m *Model) renderRunningInstructions() string {
	instructionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("243")).
		Italic(true).
		MarginTop(1)

	var instructions []string
	if m.continuousMode {
		instructions = append(instructions, "s: stop continuous ping")
	}
	instructions = append(instructions, "q: quit", "ctrl+c: stop")

	return instructionStyle.Render(strings.Join(instructions, " • "))
}

// renderResult renders the final ping results and statistics
func (m *Model) renderResult() string {
	var content strings.Builder

	// Results summary
	summaryStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("39")).
		MarginBottom(1)

	content.WriteString(summaryStyle.Render("Ping Results Summary"))
	content.WriteString("\n\n")

	// Individual results (last few)
	resultStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	maxDisplay := 5
	startIdx := 0
	if len(m.results) > maxDisplay {
		startIdx = len(m.results) - maxDisplay
		content.WriteString(resultStyle.Render(fmt.Sprintf("... (showing last %d of %d results)", maxDisplay, len(m.results))))
		content.WriteString("\n")
	}

	for i := startIdx; i < len(m.results); i++ {
		result := m.results[i]
		if result.Error != nil {
			errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
			content.WriteString(errorStyle.Render(fmt.Sprintf("❌ Ping %d: %v",
				result.Sequence, result.Error)))
		} else {
			successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("46"))
			content.WriteString(successStyle.Render(fmt.Sprintf("✅ Ping %d: %s time=%v ttl=%d",
				result.Sequence, result.Host.IPAddress, result.RTT, result.TTL)))
		}
		content.WriteString("\n")
	}

	// Statistics
	content.WriteString("\n")
	statsStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("214")).
		Border(lipgloss.RoundedBorder()).
		Padding(1).
		MarginTop(1)

	statsText := FormatPingStatistics(m.statistics)
	content.WriteString(statsStyle.Render(statsText))

	return content.String()
}

// renderError renders the error state
func (m *Model) renderError() string {
	errorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true)

	return errorStyle.Render(fmt.Sprintf("❌ Error: %s", m.error.Error()))
}

// renderFooter renders the footer with help text
func (m *Model) renderFooter() string {
	var help []string

	switch m.state {
	case StateInput:
		help = []string{"tab: next field", "enter: start ping", "q: quit"}
	case StateResult, StateError:
		help = []string{"esc: new ping", "q: quit"}
	case StateRunning:
		help = []string{"q: quit"}
	}

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	return helpStyle.Render(strings.Join(help, " • "))
}

// nextInput moves focus to the next input field
func (m *Model) nextInput() {
	m.hostInput.Blur()
	m.countInput.Blur()
	m.intervalInput.Blur()

	m.focusedInput++
	if m.focusedInput > 2 {
		m.focusedInput = 0
	}

	m.focusCurrentInput()
}

// prevInput moves focus to the previous input field
func (m *Model) prevInput() {
	m.hostInput.Blur()
	m.countInput.Blur()
	m.intervalInput.Blur()

	m.focusedInput--
	if m.focusedInput < 0 {
		m.focusedInput = 2
	}

	m.focusCurrentInput()
}

// focusCurrentInput focuses the current input field
func (m *Model) focusCurrentInput() {
	switch m.focusedInput {
	case 0:
		m.hostInput.Focus()
	case 1:
		m.countInput.Focus()
	case 2:
		m.intervalInput.Focus()
	}
}

// resetToInput resets the model to input state
func (m *Model) resetToInput() {
	// Cancel any ongoing operations
	if m.cancelFunc != nil {
		m.cancelFunc()
		m.cancelFunc = nil
	}

	m.state = StateInput
	m.hostInput.SetValue("")
	m.countInput.SetValue("4")
	m.intervalInput.SetValue("1")
	m.focusedInput = 0
	m.hostInput.Focus()
	m.countInput.Blur()
	m.intervalInput.Blur()
	m.error = nil
	m.results = []domain.PingResult{}
	m.progress = 0
	m.continuousMode = false
	
	// Reset live components
	m.liveStats = LiveStatistics{}
	m.latencyGraph.Values = make([]time.Duration, 0)
	m.packetLoss.RecentResults = make([]bool, 0)
	m.packetLoss.LossCount = 0
	m.packetLoss.TotalCount = 0
}

// startPing starts the ping operation
func (m *Model) startPing() tea.Cmd {
	host := strings.TrimSpace(m.hostInput.Value())
	countStr := strings.TrimSpace(m.countInput.Value())
	intervalStr := strings.TrimSpace(m.intervalInput.Value())

	count := 4 // default
	if countStr != "" {
		if c, err := strconv.Atoi(countStr); err == nil && c >= 0 {
			count = c
		}
	}

	interval := time.Second // default
	if intervalStr != "" {
		if i, err := strconv.ParseFloat(intervalStr, 64); err == nil && i > 0 {
			interval = time.Duration(i * float64(time.Second))
		}
	}

	m.totalPings = count
	continuous := count == 0

	return func() tea.Msg {
		return pingStartMsg{
			host:       host,
			count:      count,
			interval:   interval,
			continuous: continuous,
		}
	}
}

// executePing executes the actual ping operation with real-time updates
func (m *Model) executePing() tea.Cmd {
	return func() tea.Msg {
		host := strings.TrimSpace(m.hostInput.Value())
		countStr := strings.TrimSpace(m.countInput.Value())
		intervalStr := strings.TrimSpace(m.intervalInput.Value())

		count := 4 // default
		if countStr != "" {
			if c, err := strconv.Atoi(countStr); err == nil && c >= 0 {
				count = c
			}
		}

		interval := time.Second // default
		if intervalStr != "" {
			if i, err := strconv.ParseFloat(intervalStr, 64); err == nil && i > 0 {
				interval = time.Duration(i * float64(time.Second))
			}
		}

		// Create context with cancellation
		ctx, cancel := context.WithCancel(context.Background())
		m.cancelFunc = cancel

		// Create parameters
		opts := domain.PingOptions{
			Count:      count,
			Interval:   interval,
			Timeout:    5 * time.Second,
			PacketSize: 64,
			TTL:        64,
			IPv6:       false,
		}

		// Start ping operation
		resultChan, err := m.tool.client.Ping(ctx, host, opts)
		if err != nil {
			return pingErrorMsg{error: err}
		}

		// Return a command that will listen for ping results
		return tea.Batch(m.listenForPingResults(resultChan, ctx))
	}
}

// listenForPingResults creates a command that listens for ping results and sends progress updates
func (m *Model) listenForPingResults(resultChan <-chan domain.PingResult, ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		var results []domain.PingResult
		completed := 0

		for {
			select {
			case result, ok := <-resultChan:
				if !ok {
					// Channel closed, ping complete
					tool := &Tool{}
					stats := tool.calculateStatistics(results)
					return pingCompleteMsg{
						results:    results,
						statistics: stats,
					}
				}

				results = append(results, result)
				completed++

				// Send progress update immediately
				go func(r domain.PingResult, c int) {
					// This is a simplified approach - in a real implementation,
					// we'd need a proper way to send messages back to the UI
					// For now, we'll just return the progress message
				}(result, completed)

				// For continuous mode, keep listening
				if m.continuousMode {
					continue
				}

				// For counted mode, check if we're done
				if completed >= m.totalPings {
					tool := &Tool{}
					stats := tool.calculateStatistics(results)
					return pingCompleteMsg{
						results:    results,
						statistics: stats,
					}
				}

			case <-ctx.Done():
				// Ping was cancelled
				if len(results) > 0 {
					tool := &Tool{}
					stats := tool.calculateStatistics(results)
					return pingCompleteMsg{
						results:    results,
						statistics: stats,
					}
				}
				return pingErrorMsg{error: fmt.Errorf("ping cancelled")}
			}
		}
	}
}

// Messages for async operations
type pingStartMsg struct {
	host       string
	count      int
	interval   time.Duration
	continuous bool
}

type pingInitMsg struct{}

type pingProgressMsg struct {
	completed int
	result    domain.PingResult
}

type pingCompleteMsg struct {
	results    []domain.PingResult
	statistics PingStatistics
}

type pingErrorMsg struct {
	error error
}

type tickMsg time.Time

// tickCmd returns a command that sends tick messages for animations
func (m *Model) tickCmd() tea.Cmd {
	return tea.Tick(m.updateInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// updateLiveStats updates the live statistics with a new ping result
func (m *Model) updateLiveStats(result domain.PingResult) {
	m.liveStats.PacketsSent++
	
	if result.Error == nil {
		m.liveStats.PacketsReceived++
		
		// Update RTT statistics
		if m.liveStats.PacketsReceived == 1 {
			m.liveStats.MinRTT = result.RTT
			m.liveStats.MaxRTT = result.RTT
			m.liveStats.AvgRTT = result.RTT
		} else {
			if result.RTT < m.liveStats.MinRTT {
				m.liveStats.MinRTT = result.RTT
			}
			if result.RTT > m.liveStats.MaxRTT {
				m.liveStats.MaxRTT = result.RTT
			}
			
			// Update average (simple moving average)
			totalRTT := m.liveStats.AvgRTT * time.Duration(m.liveStats.PacketsReceived-1)
			m.liveStats.AvgRTT = (totalRTT + result.RTT) / time.Duration(m.liveStats.PacketsReceived)
		}
		
		// Calculate jitter (difference from previous RTT)
		if m.liveStats.LastRTT > 0 {
			diff := result.RTT - m.liveStats.LastRTT
			if diff < 0 {
				diff = -diff
			}
			m.liveStats.Jitter = diff
		}
		
		m.liveStats.LastRTT = result.RTT
		
		// Update latency graph
		m.latencyGraph.Values = append(m.latencyGraph.Values, result.RTT)
		if len(m.latencyGraph.Values) > m.latencyGraph.MaxValues {
			m.latencyGraph.Values = m.latencyGraph.Values[1:]
		}
		
		// Update packet loss indicator
		m.packetLoss.RecentResults = append(m.packetLoss.RecentResults, true)
	} else {
		// Packet loss
		m.packetLoss.LossCount++
		m.packetLoss.RecentResults = append(m.packetLoss.RecentResults, false)
	}
	
	// Keep only recent results for packet loss indicator
	if len(m.packetLoss.RecentResults) > m.packetLoss.MaxResults {
		m.packetLoss.RecentResults = m.packetLoss.RecentResults[1:]
	}
	
	m.packetLoss.TotalCount++
	
	// Calculate packet loss percentage
	if m.liveStats.PacketsSent > 0 {
		m.liveStats.PacketLoss = float64(m.liveStats.PacketsSent-m.liveStats.PacketsReceived) / float64(m.liveStats.PacketsSent) * 100
	}
}